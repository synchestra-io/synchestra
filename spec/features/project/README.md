# Feature: Project

**Status:** Draft

## Summary

A Synchestra project is a named unit of work identified canonically by its primary repository reference (e.g., `github.com/synchestra-io/synchestra`). Project membership — who may create sessions for the project — is governed by an ACL stored in `.synchestra/project-members.yaml` within the project's state repo. Identity is the OIDC federated tuple `(iss, sub)` so that self-hosted Synchestra hosts can authorize requests against OIDC token claims without mapping through a cloud identity service. synchestra-cloud maintains a Firestore cache (`projects/{id}`) of the yaml's membership for the cloud-mediated flow; bypass-auth hosts read the yaml directly through a local TTL-based cache.

## Problem

Three forces shape this design:

1. **Self-hosted hosts must authorize without synchestra-cloud.** The [user-authentication](../user-authentication/README.md) feature promises that a host configured with its own OIDC issuers can validate requests locally. That promise requires the authoritative ACL to be locally readable — Firestore-only authority forces a runtime dependency on cloud.
2. **Per-user favorites must not be ACLs.** Historical code treated `users/{uid}.projects` as both a UI favorites map and an access-control check. A user granted access by a collaborator couldn't exercise that access until they unilaterally updated their own user document. Authority belongs on the resource, not scattered across members.
3. **Identity must round-trip through OIDC tokens.** A self-hosted host verifying an OIDC token receives `(iss, sub)` claims. Any mapping layer between those claims and an internal Synchestra UID is a lookup against some authoritative source — defeating the bypass-auth goal. Identity-in-ACL must match identity-in-token.

This feature formalizes the project as: (a) a repo-addressable unit of work, (b) with a repo-hosted yaml ACL using `(iss, sub)` identities, (c) cached in Firestore for cloud-mediated flow and in-memory for bypass-auth hosts.

## Behavior

### Project identity

Project IDs are state-repository references of the form `{provider}/{namespace}/{slug}`:

- `github.com/synchestra-io/synchestra`
- `gitlab.company.com/org/repo`
- `git.internal.corp/team/service`

There is no separate "project URL" field. The ID itself addresses the project's state repo. Fetch URLs for the yaml are derived from the ID (see [Fetch mechanism](#fetch-mechanism)).

#### REQ: project-id-is-repo-reference

A project's ID MUST be its primary state repository reference in the form `{provider}/{namespace}/{slug}`. The provider segment MUST be a DNS name. Namespace and slug follow the provider's own path conventions.

#### REQ: project-id-stable

A project's ID is stable across its lifetime. Renaming changes metadata (display name) only; the ID never changes. Migrating a project between providers is equivalent to creating a new project.

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

#### REQ: members-file-location

The authoritative ACL MUST live at `.synchestra/project-members.yaml` at the root of the project's state repository.

#### REQ: schema-version

The file MUST carry a top-level `schema_version` integer. Current format is `1`. Parsers that encounter a higher version SHOULD refuse to authorize rather than guess.

#### REQ: member-entry-shape

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

- Project metadata (name, description, timestamps)
- Membership projected to support `array-contains`–style queries keyed by `(iss, sub)`
- Cache freshness markers (`cache_updated_at`, `cache_source`)

Exact field names and encoding are implementation concerns — see the fix-auth-v2 plan for the concrete Firestore shape.

#### REQ: firestore-is-cache

The Firestore `projects/{id}` document is a cache, not authority. Any value disagreement with the authoritative yaml MUST be resolved in favor of the yaml (by re-fetching and updating the cache).

#### REQ: cache-projection

The cache membership list is projected from yaml members whose `role` grants session-creation access. In v1, that is both `owner` and `member`.

### Lifecycle

A project moves through two phases:

#### Pre-repo phase

The project exists in hub (created via UI or API) but has no state repository yet.

- **Authority:** Firestore. `projects/{id}` is seeded by cloud when the project is created; initial membership is the creator as owner.
- **Authorization flow:** cloud-mediated only. Bypass-auth hosts cannot serve project-bound requests for this project.
- `cache_source: "bootstrap"`.

#### Post-repo phase

The state repository exists and contains `.synchestra/project-members.yaml`.

- **Authority:** yaml.
- **Authorization flow:** both cloud-mediated (via Firestore cache) and bypass-auth (via host-local cache) work.
- `cache_source: "yaml"`.

#### Transition

When the state repo is created for a pre-repo project, synchestra writes an initial `project-members.yaml` seeded from the current Firestore contents, commits, and pushes. From that commit onward, yaml is authoritative. The transition is a one-way door: yaml never reverts to bootstrap.

#### REQ: two-phase-lifecycle

A project MUST be in exactly one of two phases: pre-repo (Firestore-authoritative) or post-repo (yaml-authoritative). Transition occurs on state-repo creation and is one-way.

#### REQ: bypass-auth-requires-post-repo

A bypass-auth host MUST NOT authorize project-bound requests against a pre-repo project (no yaml to fetch). Projectless requests and host-scoped requests are unaffected.

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

- Populated by a webhook handler in synchestra-cloud on state-repo push (via the [github-app](../github-app/README.md) feature for GitHub state repos).
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

### User favorites map

`users/{uid}.projects` is a per-user favorites / recently-used list for UI ordering. It is NOT an ACL.

#### REQ: user-favorites-not-acl

The `users/{uid}.projects` map MUST NOT be used as an authorization primitive in any code path. Stale or missing entries MUST NOT cause authorization failures. UI MAY use the map for sorting and quick access.

### Runner / host scoping against projects

Runners and hosts declare `project_ids: []string` to scope *willingness* to serve projects — independent of ACL membership.

#### REQ: project_ids-literal

Each entry in a runner or host's `project_ids` MUST be a literal project ID or the wildcard `"*"` (serve any). A runner serves a project iff a literal match or the wildcard is present.

#### REQ: project_ids-regex-future

A future extension of the matcher MAY treat `project_ids` entries as regular expressions matching the full project ID. Because project IDs are themselves repo references, this maps naturally to provider/org-wide matching — e.g., `github.com/acme/(.+)` matches any project under the `acme` GitHub organization. Plain strings continue to match literally (backwards compatible); patterns are anchored against the full project ID.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [host-auth](../host-auth/README.md) | Hosts and runners have their own `user_ids` ACL. Project-bound requests are gated by both host-level and project-level ACL. Projectless and host-scoped requests bypass project ACL entirely. |
| [user-authentication](../user-authentication/README.md) | Defines how hosts validate OIDC tokens from configured issuers. The `(iss, sub)` tuple produced by validation is matched against the project yaml ACL. Bypass-auth mode requires post-repo project state. |
| [runner](../runner/README.md) | Runners declare `project_ids` (scoping). A session binds a project to a runner; the runner's `project_ids` must include the session's project ID. |
| [github-app](../github-app/README.md) | Provides the webhook infrastructure for triggering cloud reconciler updates on state-repo push. |
| [channels](../channels/README.md) | Session messages carry the session's `project_id`; downstream authorization uses the project ACL documented here. |

## Dependencies

- [github-app](../github-app/README.md) — webhook-triggered cache reconciliation for GitHub-hosted state repos. Non-GitHub providers work but rely on CLI reload as the sole reconciliation trigger until provider-specific webhook handlers are added.

## Acceptance Criteria

### AC: project-identity

**Requirements:** project#req:project-id-is-repo-reference, project#req:project-id-stable

Project IDs are `{provider}/{namespace}/{slug}` repo references; the ID is stable; renames change metadata only.

### AC: yaml-is-authoritative

**Requirements:** project#req:members-file-location, project#req:schema-version, project#req:member-entry-shape, project#req:role-enum-v1, project#req:owner-required, project#req:owner-authority

The project ACL is the yaml at `.synchestra/project-members.yaml` with `schema_version: 1`, structured members carrying `(iss, sub, role)`, role from the v1 enum, at least one owner present. Yaml-role authority is meta-gated by state-repo push access.

### AC: iss-sub-identity

**Requirements:** project#req:iss-sub-tuple, project#req:iss-is-full-url, project#req:issuer-short-name-registry

Authorization matches on the full `(iss, sub)` tuple from OIDC token claims. Short-name registry is display-only.

### AC: cache-is-not-authority

**Requirements:** project#req:firestore-is-cache, project#req:cache-projection

Firestore `projects/{id}` is a cache; yaml is the authority. Disagreement resolves in favor of the yaml. The cache projects members whose role grants access.

### AC: lifecycle-phases

**Requirements:** project#req:two-phase-lifecycle, project#req:bypass-auth-requires-post-repo

Projects have pre-repo (Firestore-authoritative) and post-repo (yaml-authoritative) phases. Bypass-auth hosts serve project-bound requests only for post-repo projects.

### AC: fetch-mechanism

**Requirements:** project#req:fetch-url-derivation, project#req:private-repo-tokens, project#req:cloud-cache-reconciler, project#req:host-cache-lazy, project#req:host-cache-ttl-default, project#req:no-host-invalidation-api-v1

Yaml is fetched via HTTPS single-file GET with provider-specific URL templates. Private repos use bearer tokens. Cloud reconciler is webhook-triggered; host cache is lazy + TTL-refreshed (default 5 min) with no external invalidation API in v1.

### AC: favorites-not-acl

**Requirements:** project#req:user-favorites-not-acl

`users/{uid}.projects` is a favorites map only; its contents (or absence) have no authorization effect.

## Outstanding Questions

- Should `role` expand to `admin` and `viewer` as first-class v2 enum values? The flat `owner | member` split is sufficient for initial collaboration but may not carry the semantics needed for team deployments.
- What is the default behavior when a project's yaml cannot be fetched (state repo down, 404, network error)? Current position: deny the request. Alternatives: allow previously-cached entries past TTL as a grace period, or fall back to the Firestore cache for bypass-auth hosts that have network to cloud.
- Should bundled issuer short-name defaults be versioned alongside `schema_version`, or maintained separately with their own cadence? Versioning with schema keeps updates atomic; separate versioning allows hotfix-style additions of newly-supported IdPs without a schema bump.
- How are non-GitHub state repos reconciled into the Firestore cache without webhooks? Current position: CLI reload only. Revisit when a real non-GitHub use case appears.
- Is the pre-repo → post-repo transition automatic (synchestra writes the initial yaml on state-repo creation) or operator-triggered? Current position: automatic; needs explicit documentation of which action triggers it.
- Should projects optionally be private (invite-only, current implicit default) vs. discoverable (any user with the ID can request access)? No discovery model exists today.
- When the regex extension to `project_ids` is delivered, should it support negative patterns (exclusions) or only inclusions?

---
*This document follows the https://specscore.md/feature-specification*
