# Device Flow for Interactive Host Connect

## Summary

Implements the interactive device authorization flow for `synchestra-host hub connect` (Path 2 in the host-auth spec). Allows a host to connect to the Hub without a pre-provisioned token by having the user approve the connection through a browser.

## Context

The host-auth feature (implemented) supports Path 1: pre-provisioned tokens via `synchestra-host hub connect --token {htr_...}`. Path 2 — the interactive device flow — lets users run `synchestra-host hub connect` without a token. The CLI obtains a short user code, the user approves it in a browser, and the CLI receives a registration token.

This design covers two repos:
- **synchestra-cloud** — Hub API endpoints for the device flow
- **synchestra-servers** — Host CLI device flow logic

The browser approval page at `hub.synchestra.io/connect-host` is owned by the **synchestra-hub** repo and is out of scope for this design.

## Flow Overview

### Happy path (callback succeeds)

```
Host CLI                          Cloud API                        Browser (hub.synchestra.io)
   |                                  |                                  |
   | POST /auth/host/device/start     |                                  |
   | { callback_url?, callback_port?} |                                  |
   |--------------------------------->|                                  |
   |   { device_code, user_code,      |                                  |
   |     verification_url,            |                                  |
   |     expires_in_seconds,          |                                  |
   |     poll_interval_seconds }      |                                  |
   |<---------------------------------|                                  |
   |                                  |                                  |
   |  Display:                        |                                  |
   |  "Go to https://hub.synchestra   |                                  |
   |   .io/connect-host               |                                  |
   |   and enter code: ABCD-1234"     |                                  |
   |                                  |                                  |
   |  Start polling every 5s          |    User navigates, logs in       |
   |  Start callback HTTP handler     |    via Firebase, enters code     |
   |                                  |                                  |
   |                                  |  POST /auth/host/device/approve  |
   |                                  |<---------------------------------|
   |                                  |  { user_code }                   |
   |                                  |  Auth: Bearer {firebase_token}   |
   |                                  |                                  |
   |                                  |  (generates reg token, upserts   |
   |                                  |   host, adds user as manager)    |
   |                                  |                                  |
   |                                  |--------------------------------->|
   |                                  |  200 { host_id }                 |
   |                                  |                                  |
   |  POST {callback_url}             |                                  |
   |    /auth/device/callback         |                                  |
   |<---------------------------------|                                  |
   |  { registration_token, host_id,  |                                  |
   |    device_code }                 |                                  |
   |--------------------------------->|                                  |
   |  200 { device_code }             |                                  |
   |                                  |                                  |
   |  (Save credentials, exit)        |                                  |
```

### Degraded path (callback fails, poll delivers token)

1. User approves in browser, Cloud attempts callback to host, it fails.
2. Cloud stores callback error (status code, error message, response body up to 8KB).
3. CLI receives token via poll (`status: "approved"`).
4. CLI saves credentials, waits 5 seconds for callback to arrive.
5. Callback arrives during wait — exit cleanly.
6. No callback after 5 seconds — CLI calls `GET /auth/host/device/:device_code/callback-status`, displays warning with error details, exits.

### Device code state machine

```
pending --(user approves, callback succeeds)--> claimed
pending --(user approves, callback fails)-----> approved --(CLI polls)--> claimed
pending --(15 minutes)------------------------> expired
```

## API Endpoints

All endpoints are on synchestra-cloud under the `/auth/host/device/` prefix.

### `POST /auth/host/device/start`

Initiates the device flow. Returns a device code (for polling) and a user code (for the browser).

**Auth:** None

**Request:**
```json
{
  "callback_url": "https://myhost.example.com",
  "callback_port": 8080
}
```

- `callback_url` — explicit URL for the callback. Takes precedence over `callback_port`.
- `callback_port` — Cloud constructs callback URL as `http://{request_source_ip}:{callback_port}` from the TCP connection's remote address.
- Both are optional. If neither is provided, Cloud skips the callback and CLI relies on polling only.

**Response 200:**
```json
{
  "device_code": "d9f2a1b4c8e7...",
  "user_code": "ABCD-1234",
  "verification_url": "https://hub.synchestra.io/connect-host",
  "expires_in_seconds": 900,
  "poll_interval_seconds": 5
}
```

- `device_code` — 32-byte hex string. Secret, used for polling. Not shown to user.
- `user_code` — 8 alphanumeric characters with hyphen (`XXXX-XXXX`). Shown to user.
- `expires_in_seconds` — 900 (15 minutes).
- `poll_interval_seconds` — 5.

### `POST /auth/host/device/approve`

Called by the browser UI when the user enters the code and confirms.

**Auth:** Firebase ID token (Bearer)

**Request:**
```json
{
  "user_code": "ABCD-1234"
}
```

**Response 200:**
```json
{
  "host_id": "host_abc123"
}
```

**Errors:**
- 400 — invalid or expired code
- 401 — invalid Firebase token
- 404 — code not found

**Side effects on approval:**
1. Generate a registration token (`htr_...`).
2. Upsert the host record in Firestore.
3. Add the approving user as a manager of the host.
4. Update the device code document: status=`approved`, `registration_token`, `host_id`, `approved_by_uid`, `approved_at`.
5. Attempt callback to host (async). Store result in the device code document.

### `POST /auth/host/device/poll`

CLI polls this endpoint to check if the user has approved the code.

**Auth:** None (device_code as proof of possession)

**Request:**
```json
{
  "device_code": "d9f2a1b4c8e7..."
}
```

**Response 200 (pending):**
```json
{
  "status": "pending"
}
```

**Response 200 (approved):**
```json
{
  "status": "approved",
  "registration_token": "htr_...",
  "host_id": "host_abc123"
}
```

**Response 410 (expired):**
```json
{
  "error": "device code expired"
}
```

On successful retrieval of an approved code, Cloud updates status to `claimed`.

### `GET /auth/host/device/:device_code/callback-status`

CLI calls this if no callback arrived within 5 seconds after receiving the token via poll.

**Auth:** None (device_code as proof)

**Response 200 (not yet attempted):**
```json
{
  "status": "pending"
}
```

**Response 200 (success):**
```json
{
  "status": "success",
  "attempted_at": "2026-03-26T12:00:00Z"
}
```

**Response 200 (failed):**
```json
{
  "status": "failed",
  "attempted_at": "2026-03-26T12:00:00Z",
  "error": "connection refused",
  "status_code": 502,
  "response_body": "..."
}
```

`response_body` truncated to 8KB.

### Callback: `POST {callback_url}/auth/device/callback`

Cloud calls this on the host when the user approves.

**Request (Cloud to Host):**
```json
{
  "registration_token": "htr_...",
  "host_id": "host_abc123",
  "device_code": "d9f2a1b4c8e7..."
}
```

**Expected response 200 (Host to Cloud):**
```json
{
  "device_code": "d9f2a1b4c8e7..."
}
```

Cloud verifies the `device_code` in the response matches. Callback is considered successful only if: HTTP 200 + valid JSON + `device_code` matches.

Any other result (wrong status, invalid JSON, mismatched device_code) is stored as a failure with:
- `status_code` — HTTP status code received
- `error` — error description
- `body` — response body (truncated to 8KB)

## Firestore Storage

### New collection: `/device_codes/{device_code}`

```
device_code          string      (32-byte hex, document ID)
user_code            string      (ABCD-1234 format)
callback_url         string      (resolved URL, optional)
status               string      (pending | approved | claimed | expired)
registration_token   string      (set on approval)
host_id              string      (set on approval)
approved_by_uid      string      (Firebase UID of approving user)
approved_at          timestamp   (set on approval)
callback_status      string      (pending | success | failed)
callback             {           (set on callback attempt)
  error: string,
  body: string,
  status_code: int
}
callback_at          timestamp   (set on callback attempt)
created_at           timestamp
expires_at           timestamp
```

Documents are short-lived. Can be cleaned up by a background job or Firestore TTL policy after expiration.

A secondary index on `user_code` is needed for the approve endpoint to look up by user code.

## CLI Flow (synchestra-servers)

### Command syntax

```
synchestra-host hub connect                                # device flow, no callback
synchestra-host hub connect --host-url http://host:8080    # device flow with callback
synchestra-host hub connect --token htr_...                # pre-provisioned (existing)
```

`--host-url` is optional. If omitted, no `callback_url` is sent. Cloud falls back to source IP + host's listening port if `callback_port` is derivable, otherwise skips callback entirely.

### Device flow sequence

1. Call `POST /auth/host/device/start` with `callback_url` (from `--host-url`) or `callback_port` (from host's configured port).
2. Display: `"Go to https://hub.synchestra.io/connect-host and enter code: ABCD-1234"`
3. Start temporary HTTP handler for `POST /auth/device/callback`.
4. Start polling `POST /auth/host/device/poll` every 5 seconds.
5. **Whichever arrives first** (callback or poll with `status: "approved"`):
   - Save credentials (`host_id`, `registration_token`, `hub_url`) to `~/.synchestra-host/credentials.json`.
   - If token came via **callback** — connectivity confirmed, exit cleanly.
   - If token came via **poll** — keep callback handler alive for 5 more seconds.
     - Callback arrives during wait — exit cleanly.
     - No callback after wait — call `GET /auth/host/device/:device_code/callback-status`, display warning with error details if available, exit.
6. On poll returning 410 (expired) — display "Device code expired. Please try again." and exit with error.

### User code format

8 alphanumeric uppercase characters with a hyphen: `XXXX-XXXX` (e.g., `ABCD-1234`). Generated from charset `ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`. Characters `0/O` and `1/I` excluded to avoid ambiguity. Effective charset: 32 characters, 32^8 = ~1.1 trillion combinations.

## Scope

**In scope (this design):**
- synchestra-cloud: 4 API endpoints + Firestore collection + callback client
- synchestra-servers: CLI device flow logic + callback handler

**Out of scope:**
- synchestra-hub: `/connect-host` browser approval page (separate spec)
- Device code cleanup/TTL (can be added later)
- Rate limiting on poll endpoint (can be added later)

## Acceptance Criteria

1. `synchestra-host hub connect` without `--token` initiates device flow and displays user code with verification URL.
2. User can approve the code through the `/auth/host/device/approve` endpoint (called by browser UI).
3. CLI receives registration token via callback or poll and saves credentials.
4. After poll-based token retrieval, CLI waits 5 seconds for callback and warns if it doesn't arrive (with error details from Cloud).
5. Device codes expire after 15 minutes.
6. Pre-provisioned flow (`--token`) continues to work unchanged.
7. `--host-url` flag allows explicit callback URL.
8. Cloud constructs callback URL from source IP + `callback_port` when no explicit URL is provided.
