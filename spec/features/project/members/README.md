# Feature: Project Members

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project/members?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project/members?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project/members?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project/members?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

The authoritative ACL for a Synchestra project is a yaml file at `.synchestra/project-members.yaml` in the project's state repository. Members are identified by the OIDC federated tuple `(iss, sub)` so that self-hosted hosts can authorize requests against access-token claims without any cloud-side mapping. synchestra-cloud maintains a Firestore cache (`projects/{id}`) projected from the yaml for cloud-mediated authorization; bypass-auth hosts fetch the yaml via HTTPS and keep a local TTL-bounded cache. Roles are a flat enum (`owner | member`) in v1, with forward-compatible room for additional roles.

## Problem

Three forces shape this design:

1. **Self-hosted hosts must authorize without synchestra-cloud.** The [user-authentication](../../user-authentication/README.md) feature promises that a host configured with its own OIDC issuers can validate requests locally. That promise requires the authoritative ACL to be locally readable — Firestore-only authority forces a runtime dependency on cloud.
2. **Identity must round-trip through OIDC tokens.** A self-hosted host verifying an OIDC token receives `(iss, sub)` claims. Any mapping layer between those claims and an internal Synchestra UID is a lookup against some authoritative source — defeating the bypass-auth goal. Identity-in-ACL must match identity-in-token.
3. **Membership grants and revokes must be server-enforceable.** Treating a per-user favorites map as the ACL (the prior model) meant users granted themselves membership; revocation required cooperating client updates. Authority belongs on the resource.

The members sub-feature implements (a) a repo-hosted yaml ACL using `(iss, sub)` identities, (b) a Firestore cache for cloud-mediated flow, (c) an in-memory cache for bypass-auth hosts, and (d) a webhook-triggered reconciler plus CLI fallback that keeps the cloud cache consistent with the yaml.

## Behavior

### Authoritative ACL: `.synchestra/project-members.yaml`

The project's membership list is a yaml file at the root of its state repository:

```yaml
schema_version: 1
members:
  - iss: https://securetoken.google.com/synchestra-hub
    sub: abc123
    role: owner
    alias: "122@synchestra.io"          # optional; display-only

  - iss: https://securetoken.google.com/synchestra-hub
    sub: def456
    role: owner                          # multi-owner supported

  - iss: https://token.actions.githubusercontent.com
    sub: "1234567"
    role: member
    alias: "user1@github.com"
```

#### REQ: file-location

The authoritative ACL MUST live at `.synchestra/project-members.yaml` at the root of the project's state repository.

#### REQ: schema-version

The file MUST carry a top-level `schema_version` integer. Current format is `1`. Parsers that encounter a higher version SHOULD refuse to authorize rather than guess.

#### REQ: entry-shape

Each entry in `members` MUST include `iss` (full issuer URL), `sub` (OIDC subject within that issuer), and `role`. `alias` is optional and display-only.

#### REQ: role-enum-v1

Under `schema_version: 1`, `role` MUST be one of `owner | member`. Parsers MUST reject unknown role values. Additional roles (e.g., `admin`, `viewer`) are forward-compatible extensions bound to future schema versions.

#### REQ: owner-required

At least one member with `role: owner` MUST be present. A project without any owner is invalid and MUST be rejected by parsers.

#### REQ: owner-authority

Members with `role: owner` have Synchestra-level admin authority (destroy project, rotate secrets, change metadata, transfer ownership). Access to edit the yaml itself is governed by the state repo's own ACL (e.g., GitHub push access) and is the **meta-ACL** above yaml-role authority. Operators MUST keep state-repo push access aligned with intended yaml editors; the two form a two-layer governance model.

### Identity model: `(iss, sub)`

ACL entries identify users by their OIDC federated tuple. Authorization matches access-token claims directly against yaml entries with no intermediate mapping.

#### REQ: iss-sub-tuple

Authorization matches a token's (`iss`, `sub`) claims against yaml entries. Both MUST match exactly for a member match.

#### REQ: iss-is-full-url

`iss` MUST be the full issuer URL. Short names, aliases, or abbreviations are display-only and are NOT the authoritative key.

#### REQ: issuer-short-name-registry

Synchestra ships with bundled short-name defaults for well-known issuers:

| Short name | Full `iss` |
|---|---|
| `synchestra.io` | `https://securetoken.google.com/synchestra-hub` |
| `github.com` | `https://token.actions.githubusercontent.com` |
| `google.com` | `https://accounts.google.com` |
| `microsoft.com` | `https://login.microsoftonline.com/common/v2.0` |

Operators MAY override or extend this registry via host config `auth.oidc.issuers[*].short_name`. The registry is used ONLY for display (alias rendering, UI labels); authorization matches on full `iss`.

### Firestore cache

For the cloud-mediated authorization flow, synchestra-cloud maintains a Firestore document at `projects/{project_id}` projected from the yaml. The document carries:

- Project metadata (name, description, timestamps).
- Membership projected to support `array-contains`–style queries keyed by `(iss, sub)`.
- Cache freshness markers (`cache_updated_at`, `cache_source`).

Exact field names and encoding are implementation concerns — see the [2026-04-20-fix-auth plan](../../../../../synchestra-cloud/spec/plans/2026-04-20-fix-auth/README.md) for the concrete Firestore shape.

#### REQ: firestore-is-cache

The Firestore `projects/{id}` document is a cache, not authority. Any value disagreement with the authoritative yaml MUST be resolved in favor of the yaml (by re-fetching and updating the cache).

#### REQ: cache-projection

The cache membership list is projected from yaml members whose `role` grants session-creation access. In v1, that is both `owner` and `member`.

### Fetch mechanism

Both the cloud-side cache reconciler and host-side cache fetch the yaml via HTTPS single-file GET. No git clone, no working copy.

#### REQ: fetch-url-derivation

The fetch URL is derived from the project ID using a provider-specific template:

| Provider | Template |
|---|---|
| `github.com` | `https://raw.githubusercontent.com/{org}/{repo}/main/.synchestra/project-members.yaml` |
| `gitlab.com` | `https://gitlab.com/{org}/{repo}/-/raw/main/.synchestra/project-members.yaml` |
| self-hosted | provider-specific; operator-configured per host |

#### REQ: private-repo-tokens

For private state repos, fetchers MUST present a bearer token. synchestra-cloud uses the synchestra GitHub App's installation token. Bypass-auth hosts use `git_tokens: { "github.com": "...", ... }` keyed by provider domain in host config.

### Cache semantics

#### Cloud cache (Firestore)

- Populated by a webhook handler in synchestra-cloud on state-repo push (via the [github-app](../../github-app/README.md) feature for GitHub state repos).
- Read by cloud-mediated session creation.
- Updated out-of-band by the `synchestra project members reload <project_id>` CLI as a webhook-failure fallback. The CLI is cloud-only — it does not propagate to hosts.
- Near-real-time propagation under normal operation.

#### Host cache (in-memory)

- Populated lazily on the first project-bound request for a given project ID.
- TTL-based refresh (default 5 minutes, configurable via host config `members_cache_ttl`).
- LRU size cap (default 1000 entries) to bound memory.
- No pre-fetch list; the host's `project_ids` config continues to serve as scoping (*is this host willing to serve project P?*) — orthogonal to ACL.
- **No host-side invalidation API in v1.** Propagation latency for yaml changes is bounded by TTL. Host admin API and cloud-to-host push channel are flagged as possible future extensions if real-world latency requirements demand it.

#### REQ: cloud-cache-reconciler

synchestra-cloud MUST maintain the `projects/{id}` Firestore cache consistent with the authoritative yaml. Reconciliation is triggered by state-repo push webhook (normal path) or by `synchestra project members reload <project_id>` (manual fallback).

#### REQ: host-cache-lazy

Bypass-auth hosts MUST fetch project-members.yaml lazily on the first project-bound request for a given project. Hosts MUST NOT pre-fetch based on any configuration list.

#### REQ: host-cache-ttl-default

The default host cache TTL is 5 minutes, operator-configurable. Entries past TTL MUST be refreshed on next access before authorization proceeds.

#### REQ: no-host-invalidation-api-v1

In v1, bypass-auth hosts MUST NOT expose an external cache-invalidation API. TTL is the sole invalidation mechanism. Operator tooling that needs faster propagation reduces TTL.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [project](../README.md) | Parent feature. Owns project identity, lifecycle, favorites semantics, and runner/host scoping. This sub-feature is the ACL primitive referenced by lifecycle and by cross-feature authorization checks. |
| [user-authentication](../../user-authentication/README.md) | Defines OIDC token validation. The `(iss, sub)` tuple produced by validation is matched against the yaml ACL defined here. |
| [host-auth](../../host-auth/README.md) | Hosts have their own `user_ids` ACL (host-level). Project-bound requests are gated by both host-level and project-level membership. |
| [github-app](../../github-app/README.md) | Provides the webhook infrastructure for triggering cloud cache reconciliation on state-repo push. |

## Dependencies

- [github-app](../../github-app/README.md) — webhook-triggered cache reconciliation for GitHub-hosted state repos. Non-GitHub providers work but rely on CLI reload as the sole reconciliation trigger until provider-specific webhook handlers are added.

## Acceptance Criteria

### AC: yaml-is-authoritative

**Requirements:** project/members#req:file-location, project/members#req:schema-version, project/members#req:entry-shape, project/members#req:role-enum-v1, project/members#req:owner-required, project/members#req:owner-authority

The project ACL is the yaml at `.synchestra/project-members.yaml` with `schema_version: 1`, structured members carrying `(iss, sub, role)`, role from the v1 enum, at least one owner present. Yaml-role authority is meta-gated by state-repo push access.

### AC: iss-sub-identity

**Requirements:** project/members#req:iss-sub-tuple, project/members#req:iss-is-full-url, project/members#req:issuer-short-name-registry

Authorization matches on the full `(iss, sub)` tuple from OIDC token claims. Short-name registry is display-only.

### AC: cache-is-not-authority

**Requirements:** project/members#req:firestore-is-cache, project/members#req:cache-projection

Firestore `projects/{id}` is a cache; yaml is the authority. Disagreement resolves in favor of the yaml. The cache projects members whose role grants access.

### AC: fetch-mechanism

**Requirements:** project/members#req:fetch-url-derivation, project/members#req:private-repo-tokens, project/members#req:cloud-cache-reconciler, project/members#req:host-cache-lazy, project/members#req:host-cache-ttl-default, project/members#req:no-host-invalidation-api-v1

Yaml is fetched via HTTPS single-file GET with provider-specific URL templates. Private repos use bearer tokens. Cloud reconciler is webhook-triggered; host cache is lazy + TTL-refreshed (default 5 min) with no external invalidation API in v1.

## Open Questions

- Should `role` expand to `admin` and `viewer` as first-class v2 enum values? The flat `owner | member` split is sufficient for initial collaboration but may not carry the semantics needed for team deployments.
- What is the default behavior when a project's yaml cannot be fetched (state repo down, 404, network error)? Current position: deny the request. Alternatives: allow previously-cached entries past TTL as a grace period, or fall back to the Firestore cache for bypass-auth hosts that have network to cloud.
- Should bundled issuer short-name defaults be versioned alongside `schema_version`, or maintained separately with their own cadence? Versioning with schema keeps updates atomic; separate versioning allows hotfix-style additions of newly-supported IdPs without a schema bump.
- How are non-GitHub state repos reconciled into the Firestore cache without webhooks? Current position: CLI reload only. Revisit when a real non-GitHub use case appears.

---
*This document follows the https://specscore.md/feature-specification*
