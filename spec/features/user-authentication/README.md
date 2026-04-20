# Feature: User Authentication

**Status:** Draft

> **Reconciliation pending.** This spec predates the [fix-auth plan](../../../../synchestra-cloud/spec/plans/fix-auth/README.md) and the canonical [project feature](../project/README.md). Once fix-auth ships, this spec MUST be reworked to:
>
> - Drop the bespoke `served_projects` primitive. Authorization will derive from `project.user_ids` (project ACL) and `host.user_ids` / `runner.user_ids` (host ACL), per the established two-axis model.
> - Align token-claim shape with the reconciled membership model (likely carrying the user's `project.user_ids` memberships at token-issue time, or a signed manifest that hosts cache).
> - Re-examine the "non-hub issuer trust-all-served-projects" fallback in light of the new ACL primitives.
>
> Until that rework, every `served_projects` reference below is a placeholder that will be replaced by the project / host ACL primitives. See also the merge proposal at [proposals/merge-with-host-auth](proposals/merge-with-host-auth/README.md).

## Summary

The host authenticates inbound HTTP requests by validating OIDC ID tokens locally against one or more configured issuers. `hub.synchestra.io` is the default issuer; operators may configure additional OIDC-compliant providers (Auth0, Keycloak, Okta, Google, Microsoft, a self-hosted IdP, etc.) to run alongside hub or replace it. Authorization is project-scoped: tokens issued by hub carry a `synchestra.projects` claim that is intersected with the host's `served_projects`; tokens issued by any non-hub issuer grant access to all of the host's `served_projects` once authenticated.

## Problem

Today, every request to a synchestra-host must originate from an authenticated session on `hub.synchestra.io`. This couples runtime availability to hub availability and contradicts the ecosystem's open-core promise: `synchestra-host` is free, open-source, and self-hostable, but cannot actually be used end-to-end without a paid-product account.

Three concrete consequences follow from the current design:

- **OSS users running a host in a homelab, CI environment, or air-gapped network cannot use it.** Every request path runs through hub.
- **Every request adds a network hop through `api.synchestra.io`**, increasing latency and making hub API availability a hard runtime dependency rather than a login-time dependency.
- **Enterprise buyers cannot verify there is no hidden lock-in.** Procurement conversations that ask "what happens if synchestra.io goes away?" get an unsatisfactory answer.
- **Teams that have standardized on a corporate IdP (Google Workspace, Microsoft Entra, Okta) cannot point their host at it.** Even when they accept hub as the default, they often want to layer their existing SSO on top — or run multiple IdPs concurrently for mixed-audience hosts.

The host needs to authenticate requests on its own, using a standard protocol, supporting multiple issuers at once so that hub remains a useful default without being a monopoly.

## Behavior

### Default configuration

When the host starts with no explicit authentication configuration, it trusts `hub.synchestra.io` as its sole OIDC issuer. No config file, no environment variable, and no additional setup is required.

#### REQ: default-issuer-hub

With no authentication configuration, the host MUST behave as if `auth.oidc.issuers` contained a single entry with `url: "https://hub.synchestra.io"`. The host MUST discover endpoints via `https://hub.synchestra.io/.well-known/openid-configuration`.

#### REQ: default-served-projects

In the default configuration, `served_projects` is populated during host registration with `synchestra-cloud` (see [cloud-runner-mode](../cloud-runner-mode/README.md)). A hand-written `served_projects` list is not required when hub is the only issuer — hub is the source of truth for which projects the host serves.

### Issuer configuration

Operators may declare one or more OIDC issuers. Declaring any explicit list overrides the default, so operators who want hub *alongside* other issuers must include hub in the list.

#### REQ: issuers-list

The `auth.oidc.issuers` key MUST accept a list of issuer entries. Each entry has:

| Field | Required | Description |
|---|---|---|
| `url` | Yes | The issuer URL. MUST be HTTPS and MUST serve `/.well-known/openid-configuration`. |
| `audience` | No | The expected `aud` claim. Defaults to the host's `host_id` assigned at registration. |
| `jwks_uri` | No | Overrides the JWKS URL if the issuer's discovery document is non-standard. |
| `jwks_cache_ttl` | No | Per-issuer JWKS cache TTL. Defaults to 24h. |
| `allowed_email_domains` | No | If set, only tokens whose `email` claim ends with one of these domains are accepted under this issuer. |

#### REQ: non-hub-issuer-requires-served-projects

If the `auth.oidc.issuers` list contains any entry whose `url` is not `https://hub.synchestra.io`, `served_projects` MUST be explicitly configured. The host MUST refuse to start if any non-hub issuer is configured and `served_projects` is empty.

#### REQ: explicit-list-replaces-default

If `auth.oidc.issuers` is set, the host MUST use exactly that list. Hub is NOT implicitly appended. Operators who want hub alongside other issuers MUST include hub explicitly.

#### REQ: config-example

The following configuration file demonstrates the supported modes.

```yaml
# synchestra-host.yaml

# ---- Mode 1: Default (hub.synchestra.io only) ---------------------
# No auth section required. The host registers with
# synchestra-cloud, receives its host_id and served_projects,
# and trusts hub as the OIDC issuer.

# ---- Mode 2: Hub + corporate SSO (multi-issuer) -------------------
auth:
  oidc:
    issuers:
      - url: "https://hub.synchestra.io"
        # Hub is special: authz comes from the synchestra.projects
        # claim. No audience override needed; host_id is used.

      - url: "https://accounts.google.com"
        audience: "synchestra-host-prod-01"
        allowed_email_domains: ["mycompany.com"]

      - url: "https://login.microsoftonline.com/{tenant}/v2.0"
        audience: "synchestra-host-prod-01"

served_projects:
  - proj_abc123
  - proj_def456

# ---- Mode 3: Corporate SSO only (hub replaced) --------------------
# auth:
#   oidc:
#     issuers:
#       - url: "https://your-tenant.auth0.com/"
#         audience: "synchestra-host-prod-01"
# served_projects:
#   - proj_abc123
```

Equivalent environment-variable form for a single-issuer override:

```
SYNCHESTRA_OIDC_ISSUER=https://your-tenant.auth0.com/
SYNCHESTRA_OIDC_AUDIENCE=synchestra-host-prod-01
SYNCHESTRA_SERVED_PROJECTS=proj_abc123,proj_def456
```

Multi-issuer configuration via environment variables is not supported — operators with more than one issuer MUST use a config file.

### Token validation

Every inbound HTTP request carrying `Authorization: Bearer <token>` is validated before any handler runs. Validation resolves the correct issuer from the token itself and runs per-issuer rules.

#### REQ: issuer-resolution

The host MUST read the token's `iss` claim (without verifying the signature first) and look up a matching entry in `auth.oidc.issuers`. If no entry matches, the host MUST reject the request with `401 Unauthorized` and MUST NOT attempt validation against any other configured issuer.

#### REQ: signature-verification

Once the issuer entry is resolved, the host MUST verify the JWT signature using that issuer's JWKS. Tokens signed with a key not present in the current JWKS after an on-demand refresh MUST be rejected.

#### REQ: standard-claims

The host MUST verify `iss` matches the resolved issuer's URL exactly, `aud` matches the resolved issuer's expected audience (see [Audience binding](#audience-binding)), `exp` is in the future, and `nbf` (if present) is in the past. Any failure MUST result in `401 Unauthorized`.

#### REQ: clock-skew-tolerance

The host MUST permit up to 60 seconds of clock skew when evaluating `exp` and `nbf`.

### Audience binding

Tokens are bound to a specific host to prevent a valid token for host A from being used against host B. Each issuer may have its own expected audience.

#### REQ: audience-default

When an issuer entry does not specify `audience`, the host's expected audience for that issuer MUST be the host's `host_id` assigned during registration with `synchestra-cloud`.

#### REQ: cross-host-replay-rejection

A token whose `aud` claim does not match the audience expected by its resolved issuer MUST be rejected with `401 Unauthorized`, regardless of whether the signature is valid.

### Authorization under the hub issuer

Tokens minted by hub carry the user's project memberships as a claim. Authorization is the intersection of claim and config.

#### REQ: hub-project-claim

When the resolved issuer is hub, validated tokens MUST contain a `synchestra.projects` claim whose value is an array of project IDs. Tokens from hub without this claim MUST be rejected with `401 Unauthorized`.

#### REQ: hub-project-intersection

Under the hub issuer, the host MUST grant access if and only if the intersection of `token.synchestra.projects` and the host's `served_projects` is non-empty. The request handler MUST receive the list of permitted projects as context for downstream authorization.

### Authorization under non-hub issuers

Non-hub issuers have no standard way to convey Synchestra project membership. The host falls back to a simpler trust model.

#### REQ: non-hub-trust-model

When the resolved issuer is any non-hub entry, the host MUST grant access to all projects in `served_projects` to any successfully authenticated user. The host MUST NOT require a `synchestra.projects` claim under a non-hub issuer.

#### REQ: email-domain-filter

When the resolved non-hub issuer entry specifies `allowed_email_domains`, the host MUST reject tokens whose `email` claim is missing or whose domain is not in the list.

#### REQ: non-hub-subject-logging

The host MUST log the authenticated `iss`, `sub`, and `email` (if present) for every granted request under a non-hub issuer, for operator audit.

### JWKS caching

JWKS are cached per issuer, independently.

#### REQ: jwks-cache-ttl

The host MUST cache each issuer's JWKS for at least 24 hours by default. The TTL MUST be overridable per issuer via the issuer entry's `jwks_cache_ttl`.

#### REQ: jwks-on-demand-refresh

When a token presents a `kid` not in the cached JWKS for its resolved issuer, the host MUST refresh that issuer's JWKS once before rejecting the token. Refresh failures MUST NOT purge the existing cache.

#### REQ: jwks-startup-tolerance

The host MUST start successfully even if one or more JWKS endpoints are unreachable at startup. Validation of the first token from each issuer triggers that issuer's initial fetch; failure rejects the request but leaves the host available to validate tokens from other configured issuers.

### Session continuation

OIDC validation on every request is expensive. After the first validated request in a session, the host issues its own short-lived session token to avoid round-tripping to the IdP.

#### REQ: session-token-issuance

After a successfully validated OIDC token, the host MUST issue an HttpOnly, Secure session cookie scoped to the host's origin. The cookie payload MUST be a signed token containing the authenticated subject, resolved issuer URL, permitted projects, and expiry.

#### REQ: session-token-lifetime

Session tokens MUST expire no later than the OIDC token's `exp` claim, or 8 hours, whichever is earlier.

#### REQ: session-token-scope

Session tokens MUST NOT be valid across hosts. The token MUST be signed with a per-host secret and MUST include the host's `host_id`.

### Authentication flow

#### REQ: authorization-code-flow-pkce

Browser-based authentication MUST use OAuth 2.0 Authorization Code Flow with PKCE. The host MUST NOT accept implicit flow or password grant tokens.

#### REQ: no-host-side-login-orchestration

The host MUST NOT expose login pages, IdP picker UIs, or redirect endpoints that initiate the OAuth flow. The host's responsibility is limited to validating tokens presented on inbound requests. Choosing an IdP, running the authorization code flow, and obtaining tokens are client concerns (Hub UI, CLI, or a third-party client).

#### REQ: m2m-out-of-scope

Machine-to-machine authentication (CI pipelines, scripts, long-lived agents) is out of scope for this spec. Operators requiring M2M access should run a client-credentials-capable IdP and wait for a future spec.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [cloud-runner-http](../cloud-runner-http/SPEC.md) | Defines the HTTP API that this feature guards. Authentication runs as middleware ahead of all session and project endpoints. |
| [cloud-runner-mode](../cloud-runner-mode/README.md) | Cloud Run hosts perform the same OIDC validation as VM hosts. The Cloud Run self-registration flow provides `host_id` and, under the default configuration, `served_projects`. |
| [installation](../installation/README.md) | The host's configuration file lives at an XDG-compliant path defined by the installation spec. The `auth` and `served_projects` keys documented here extend that file. |

## Dependencies

- installation
- cloud-runner-http

## Acceptance Criteria

### AC: default-mode-works

**Requirements:** host-authentication#req:default-issuer-hub, host-authentication#req:hub-project-claim, host-authentication#req:hub-project-intersection

A freshly registered host with no auth configuration accepts a valid hub-issued token whose `synchestra.projects` intersects the host's `served_projects`, rejects tokens from any unconfigured issuer, and rejects hub tokens whose `synchestra.projects` does not intersect.

### AC: multi-issuer-routing

**Requirements:** host-authentication#req:issuers-list, host-authentication#req:issuer-resolution, host-authentication#req:non-hub-trust-model

A host configured with hub plus Google plus Microsoft validates each token against the issuer named in its `iss` claim, applies claim-based authz for hub tokens and trust-all authz for Google and Microsoft tokens, and rejects any token whose `iss` does not match a configured entry.

### AC: explicit-list-replaces-default

**Requirements:** host-authentication#req:explicit-list-replaces-default, host-authentication#req:non-hub-issuer-requires-served-projects

A host configured with `issuers: [<single-non-hub-entry>]` rejects hub tokens (hub is not implicitly retained), and the host refuses to start if `served_projects` is empty under any non-hub configuration.

### AC: token-rejection-paths

**Requirements:** host-authentication#req:signature-verification, host-authentication#req:standard-claims, host-authentication#req:cross-host-replay-rejection

Tokens with invalid signatures, wrong issuer, expired `exp`, or mismatched `aud` are all rejected with `401 Unauthorized` before any handler runs. A valid token minted for host A cannot be replayed against host B, even when both hosts trust the same issuer.

### AC: email-domain-filter

**Requirements:** host-authentication#req:email-domain-filter

Under a non-hub issuer entry with `allowed_email_domains: ["mycompany.com"]`, tokens from `alice@mycompany.com` are accepted and tokens from `bob@other.com` are rejected.

### AC: jwks-resilience

**Requirements:** host-authentication#req:jwks-cache-ttl, host-authentication#req:jwks-on-demand-refresh, host-authentication#req:jwks-startup-tolerance

The host caches JWKS per issuer for at least 24h, refreshes on unknown `kid`, tolerates transient outages of one issuer's JWKS endpoint without affecting validation against other issuers, and starts successfully even if every configured IdP is unreachable at boot.

### AC: session-continuation

**Requirements:** host-authentication#req:session-token-issuance, host-authentication#req:session-token-lifetime, host-authentication#req:session-token-scope

After the first OIDC-validated request, subsequent requests in the same browser session authenticate via the host-issued session cookie without a round-trip to the IdP. Session tokens expire at or before the underlying OIDC token's `exp` (capped at 8h), are bound to the issuing host's `host_id`, and cannot be used against a different host.

### AC: host-serves-no-login-ui

**Requirements:** host-authentication#req:no-host-side-login-orchestration

The host exposes no `/login`, `/oauth/authorize`, IdP picker, or equivalent endpoint. Clients requesting such paths receive `404 Not Found`.

## Outstanding Questions

- Should the host support per-issuer `served_projects` overrides, so different IdPs grant access to different project subsets on the same host? Current spec says no (one `served_projects` list applies to all non-hub issuers); revisit if a real use case appears.
- Should a config validation step catch misconfigured `allowed_email_domains` (e.g., operator types a typo), or is runtime rejection of mismatched tokens sufficient?
- What is the canonical config file location and format? The example uses YAML at an implied path; this needs to be reconciled with the existing [installation](../installation/README.md) spec and may require a proposal against it.
- How does this interact with the future direct-UI-to-runner path (Hub UI sending messages to the runner bypassing `api.synchestra.io`)? The feature is designed to enable it, but the browser-side flow (token acquisition, CORS, cookie scope for cross-subdomain hub UI) is not covered here.
- What is the expected rollout path for existing hosts? New hosts can start with OIDC enabled; existing hosts using the current mechanism need a migration that this spec does not prescribe.
- Should the host expose a `/.well-known/synchestra-host` metadata endpoint advertising its `host_id`, configured issuers (URLs only, no secrets), and served projects, so clients can discover which IdPs are acceptable?

---
*This document follows the https://specscore.md/feature-specification*
