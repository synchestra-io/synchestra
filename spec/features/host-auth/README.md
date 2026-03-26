# Feature: Host-Hub Authentication

**Status:** Conceptual

## Summary

Mutual authentication between runner hosts and the Synchestra Hub. Hosts authenticate to the Hub using short-lived access tokens derived from a permanent registration token. The Hub authenticates to hosts by signing requests with a private key whose public counterpart is published at a well-known URL.

## Problem

Hosts currently send a static API key (`SYNCHESTRA_API_KEY`) as a Bearer token on every request to the Hub. This has several weaknesses:

1. **No token hierarchy** --- A single long-lived token is used directly for API calls. If intercepted, it grants permanent access until manually revoked.
2. **No Hub authentication** --- Hosts accept any inbound request without verifying it came from the Hub. A compromised network could inject forged session-start or message-push commands.
3. **No provisioning flow** --- The API key must be manually placed in an environment variable. There is no guided registration experience.
4. **No ownership model** --- There is no concept of who manages a host or how management authority is shared.

## Behavior

### Token Hierarchy

Three credential types govern host-hub communication:

| Credential | Belongs to | Stored by | Lifetime | Purpose |
|---|---|---|---|---|
| Registration token (`htr_...`) | Host record | Host (config file on disk) | Permanent until revoked or replaced | Exchange for access tokens |
| Access token (`hta_...`) | Host record | Host (in memory only) | Server-dictated TTL via `expires_in_seconds` | Host -> Hub API calls |
| Hub signing key pair | Hub | Hub (private key); public key at well-known URL | Rotatable | Hub -> Host request signing |

### Host -> Hub Authentication

1. On startup, the host calls `POST /auth/host/refresh` with its registration token in the `Authorization: Bearer {htr_...}` header.
2. The Hub validates the registration token and returns:
   ```json
   {
     "access_token": "hta_...",
     "token_type": "Bearer",
     "expires_in_seconds": 900
   }
   ```
3. The host uses the access token as `Authorization: Bearer {hta_...}` for all subsequent Hub API calls (registration heartbeats, outbound message forwarding, status reports).
4. The host refreshes the access token at ~80% of TTL (e.g., at 720s for a 900s TTL).
5. If an API call returns 401, the host refreshes immediately and retries the request once.
6. If refresh itself returns 401, the registration token has been revoked --- the host logs the error, stops accepting work, and exits with: "Registration token revoked. Run `synchestra-host hub connect` to re-register."
7. If the Hub is unreachable during refresh, the host retries with exponential backoff (1s -> 30s cap) and continues operating with the current access token until it expires.

### Hub -> Host Authentication

The Hub signs every request to a host using its Ed25519 private key.

**Signature header:**
```
X-Hub-Signature: t=1711468800,kid=hub_key_001,sig=base64encodedSignature
```

**Signed payload** (concatenated with `.`):
```
{timestamp}.{request_body}
```

**Host verification:**
1. Parse `t` (timestamp), `kid` (key ID), and `sig` (signature) from the header.
2. Reject if timestamp is older than 5 minutes (replay protection).
3. If `kid` does not match the cached key's `key_id`, fetch the new public key before verifying.
4. Reconstruct the signed payload: `{t}.{request_body}`.
4. Verify `sig` against the cached public key.
5. If verification fails, fetch a fresh public key from the Hub and verify again (handles key rotation).
6. If verification still fails, reject the request with 401.

**Public key endpoint:** `GET /public-key`

```json
{
  "key": "base64encodedPublicKey",
  "key_id": "hub_key_001",
  "algorithm": "ed25519"
}
```

The host fetches the public key on startup and caches it. A signature verification failure triggers a single re-fetch before rejecting --- this handles key rotation without explicit coordination.

### Host Registration Paths

Both paths produce the same outcome: a registration token stored on disk and the user added as a host manager.

#### Path 1: Pre-provisioned token

A user creates a host via the Synchestra CLI or Web UI. The Hub creates (or upserts) the host record and returns a registration token. The user then runs:

```
synchestra-host hub connect --token {htr_...}
```

The host stores the token, exchanges it for an access token, fetches the Hub public key, and starts operating.

#### Path 2: Interactive device flow

```
synchestra-host hub connect
```

1. The host CLI calls the Hub's device authorization endpoint and receives a `device_code` and `user_code`.
2. The CLI displays: "Go to hub.synchestra.io/device and enter code: ABCD-1234"
3. The user authenticates in the browser and approves the request.
4. The host CLI polls the Hub until approved and receives the registration token.
5. The Hub upserts the host record (creates it if it does not exist) and adds the user as a manager.
6. The host stores the token and proceeds as in Path 1.

### Credential Storage

**Primary:** `~/.synchestra-host/credentials.json`

```json
{
  "hub_url": "https://hub.synchestra.io",
  "host_id": "host_abc123",
  "registration_token": "htr_..."
}
```

- File permissions: `0600` (owner read/write only)
- Directory permissions: `0700`

**Override:** The `SYNCHESTRA_API_KEY` environment variable takes precedence over the config file if set. This maintains backwards compatibility with the current deployment model.

**Lifecycle:**
- `synchestra-host hub connect` writes the credentials file.
- `synchestra-host hub disconnect` deletes the file and revokes the token on the Hub.
- Each `connect` issues a new registration token and revokes the previous one.

### Host CLI Commands

Following the `{resource} {command}` pattern:

| Command | Description |
|---|---|
| `synchestra-host hub connect` | Interactive device flow registration |
| `synchestra-host hub connect --token {htr_...}` | Pre-provisioned token registration |
| `synchestra-host hub disconnect` | Revoke token and remove credentials |
| `synchestra-host hub status` | Show connection state and manager info |

### Host Managers

Managers are users with authority over a host record. The management model is:

- A host has one or more managers.
- When a user registers a host (via either path), they become a manager. The token confirms the user has access to the machine --- after that, the token belongs to the host, not the user.
- Any manager can view all managers, remove other managers, and reconnect the host.
- Removing a manager removes their management authority only --- it does not invalidate the host's registration token.
- To invalidate a token, a manager reconnects the host (issuing a new token and revoking the old one) or explicitly revokes it via the UI or CLI.
- A host is independent of projects --- it is a shared compute resource that multiple users and projects can use.

### Error Handling

| Scenario | Behavior |
|---|---|
| Registration token revoked | Host logs error, stops accepting work, exits with message to re-register |
| Access token expired mid-request | Refresh access token, retry request once |
| Hub unreachable during refresh | Exponential backoff (1s -> 30s cap); continue with current access token until expiry; stop accepting new work if token expires |
| Public key fetch fails on startup | Host cannot verify Hub requests; rejects them until key is fetched |
| Public key fetch fails on retry | Continue using cached key; reject request if no cached key |
| Duplicate `hub connect` on same host | New registration token issued, old one revoked; user added as manager if not already one |

## Dependencies

- [runner](../runner/README.md) --- Host registration is a prerequisite for runner functionality; this feature answers runner outstanding question #2
- [channels](../channels/README.md) --- Host-hub message routing depends on authenticated communication
- [api](../api/README.md) --- Token management endpoints (`/auth/host/refresh`, `/public-key`, device flow)

## Acceptance Criteria

1. A user can register a host via `synchestra-host hub connect --token {token}` and the host exchanges the registration token for a short-lived access token.
2. A user can register a host via `synchestra-host hub connect` (device flow) without a pre-provisioned token.
3. The host stores credentials in `~/.synchestra-host/credentials.json` with `0600` permissions.
4. `SYNCHESTRA_API_KEY` environment variable overrides the credentials file.
5. The host refreshes its access token before TTL expiry and retries on 401.
6. The Hub signs all requests to hosts; hosts verify signatures using the public key from `/public-key`.
7. Signature verification failure triggers a single public key re-fetch before rejecting.
8. Requests with timestamps older than 5 minutes are rejected (replay protection).
9. `synchestra-host hub disconnect` revokes the token on the Hub and deletes the local credentials file.
10. Multiple users can be managers of the same host; removing a manager does not invalidate the host's token.
11. Reconnecting a host (`hub connect` on an already-connected host) issues a new registration token and revokes the old one.
12. If the Hub is unreachable, the host retries with exponential backoff and continues operating until the access token expires.

## Outstanding Questions

1. Should the Hub public key endpoint support multiple active keys (for overlap during rotation), or is the single-key-with-retry-on-failure approach sufficient?
2. What is the maximum number of managers per host? Should there be a limit?
3. Should `synchestra-host hub status` show active sessions and resource usage in addition to connection state?
