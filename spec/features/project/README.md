# Feature: Project

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/project?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

A Synchestra project is a named unit of work identified canonically by its primary repository reference (e.g., `github.com/synchestra-io/synchestra`). Projects have a two-phase lifecycle — pre-repo (Firestore-authoritative membership) and post-repo (yaml-authoritative membership) — that enables both cloud-mediated and bypass-auth session authorization. Membership semantics, authorization primitives, yaml schema, and caching are specified in the [members](https://github.com/synchestra-io/synchestra/blob/main/spec/features/project/members/README.md) sub-feature.

## Contents

| Directory | Description |
|---|---|
| [members/](members/README.md) | Project membership primitive: `.synchestra/project-members.yaml` schema, `(iss, sub)` federated identity, role enum, Firestore cache, HTTPS fetch mechanism, host-side TTL cache semantics. |

### members/

The members sub-feature owns the implementation substance of project ACL: the yaml schema at `.synchestra/project-members.yaml` (the authoritative source), `(iss, sub)` OIDC-federated identity, the `owner | member` role enum, the Firestore cache projection for cloud-mediated authorization, HTTPS single-file GET fetch with provider-specific URL templates, and the host-side lazy TTL cache with no external invalidation API in v1. Everything mechanical about *"how does the system know who's a member"* lives there.

## Problem

Synchestra's session model binds work to projects: sessions run under a project, runners scope to projects, authorization checks project membership. A durable spec pins three things that would otherwise drift across call sites:

1. **The ID format** — a stable repo-reference shape (`{provider}/{namespace}/{slug}`) so code repos, URLs, and integrations can name a project consistently.
2. **The lifecycle** — projects live in hub before they have a state repo; the model must accommodate both phases without a separate "draft project" concept.
3. **The state boundary** — project-level state (ACL, metadata) lives on the project; per-user state (favorites, recency) lives on the user document. The membership mechanics live in the [members](members/README.md) sub-feature.

## Behavior

### Project identity

Project IDs are state-repository references of the form `{provider}/{namespace}/{slug}`:

- `github.com/synchestra-io/synchestra`
- `gitlab.company.com/org/repo`
- `git.internal.corp/team/service`

There is no separate "project URL" field. The ID itself addresses the project's state repo; fetch URLs for the members yaml derive from the ID via provider-specific templates (see [members#req:fetch-url-derivation](members/README.md#req-fetch-url-derivation)).

#### REQ: project-id-is-repo-reference

A project's ID MUST be its primary state repository reference in the form `{provider}/{namespace}/{slug}`. The provider segment MUST be a DNS name. Namespace and slug follow the provider's own path conventions.

#### REQ: project-id-stable

A project's ID is stable across its lifetime. Renaming a project changes metadata (display name) only; the ID never changes. Migrating a project between providers is equivalent to creating a new project.

### Lifecycle

A project moves through two phases:

#### Pre-repo phase

The project exists in hub (created via UI or API) but has no state repository yet.

- **Membership authority:** Firestore. `projects/{id}` is seeded by cloud on project creation with the creator as owner.
- **Authorization flow:** cloud-mediated only. Bypass-auth hosts cannot serve project-bound requests for this project.

#### Post-repo phase

The state repository exists and contains `.synchestra/project-members.yaml`.

- **Membership authority:** the yaml. See [members](members/README.md).
- **Authorization flow:** both cloud-mediated (via Firestore cache) and bypass-auth (via host-local cache) work.

#### Transition

When the state repo is created for a pre-repo project, synchestra writes an initial `project-members.yaml` seeded from the Firestore cache, commits, and pushes. From that commit onward, yaml is authoritative. The transition is a one-way door: yaml never reverts to bootstrap.

#### REQ: two-phase-lifecycle

A project MUST be in exactly one of two phases: pre-repo (Firestore-authoritative) or post-repo (yaml-authoritative). Transition occurs on state-repo creation and is one-way.

#### REQ: bypass-auth-requires-post-repo

A bypass-auth host MUST NOT authorize project-bound requests against a pre-repo project (no yaml to fetch). Projectless requests and host-scoped requests are unaffected.

### User favorites map

`users/{uid}.projects` is a per-user favorites / recently-used list for UI ordering. It is NOT an ACL.

#### REQ: user-favorites-not-acl

The `users/{uid}.projects` map MUST NOT be used as an authorization primitive in any code path. Stale or missing entries MUST NOT cause authorization failures. UI MAY use the map for sorting and quick access.

### Runner / host scoping against projects

Runners and hosts declare `project_ids: []string` to scope *willingness* to serve projects — independent of ACL membership (which is a project-level concern; see [members](members/README.md)).

#### REQ: project_ids-literal

Each entry in a runner or host's `project_ids` MUST be a literal project ID or the wildcard `"*"` (serve any). A runner serves a project iff a literal match or the wildcard is present.

#### REQ: project_ids-regex-future

A future extension of the matcher MAY treat `project_ids` entries as regular expressions matching the full project ID. Because project IDs are themselves repo references, this maps naturally to provider/org-wide matching — e.g., `github.com/acme/(.+)` matches any project under the `acme` GitHub organization. Plain strings continue to match literally (backwards compatible); patterns are anchored against the full project ID.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [members](members/README.md) | Owns the authoritative membership primitive referenced by project-bound authorization everywhere. |
| [host-auth](../host-auth/README.md) | Hosts and runners have their own `user_ids` ACL. Project-bound requests are gated by both host-level and project-level ACL; projectless and host-scoped requests bypass project ACL entirely. |
| [user-authentication](../user-authentication/README.md) | Defines how hosts validate OIDC tokens. The `(iss, sub)` tuple produced by validation is matched against the project yaml ACL — see [members](members/README.md). Bypass-auth mode requires post-repo project state. |
| [runner](../runner/README.md) | Runners declare `project_ids` (scoping). A session binds a project to a runner; the runner's `project_ids` must include the session's project ID. |
| [channels](../channels/README.md) | Session messages carry the session's `project_id`; downstream authorization uses the project ACL documented in [members](members/README.md). |

## Dependencies

None at this level. The [members](members/README.md) sub-feature has its own dependencies (notably [github-app](../github-app/README.md) for webhook-triggered cache reconciliation).

## Acceptance Criteria

### AC: project-identity

**Requirements:** project#req:project-id-is-repo-reference, project#req:project-id-stable

Project IDs are `{provider}/{namespace}/{slug}` repo references; the ID is stable; renames change metadata only.

### AC: lifecycle-phases

**Requirements:** project#req:two-phase-lifecycle, project#req:bypass-auth-requires-post-repo

Projects have pre-repo (Firestore-authoritative) and post-repo (yaml-authoritative) phases. Bypass-auth hosts serve project-bound requests only for post-repo projects.

### AC: favorites-not-acl

**Requirements:** project#req:user-favorites-not-acl

`users/{uid}.projects` is a favorites map only; its contents (or absence) have no authorization effect.

### AC: project_ids-literal

**Requirements:** project#req:project_ids-literal

The runner/host `project_ids` matcher accepts literal IDs and the `"*"` wildcard today.

## Open Questions

- Is the pre-repo → post-repo transition automatic (synchestra writes the initial yaml on state-repo creation) or operator-triggered? Current position: automatic; needs explicit documentation of which action triggers it.
- Should projects optionally be private (invite-only, current implicit default) vs. discoverable (any user with the ID can request access)? No discovery model exists today.
- When the regex extension to `project_ids` is delivered, should it support negative patterns (exclusions) or only inclusions?

---
*This document follows the https://specscore.md/feature-specification*
