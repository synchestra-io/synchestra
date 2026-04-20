# Feature: Project

**Status:** Draft

## Summary

A Synchestra project is a named unit of work — a logical grouping of specs, sessions, and code repositories that users collaborate on. Projects are first-class entities in synchestra-cloud's Firestore with their own ACL. Membership in a project's `user_ids` list authorizes a user to create sessions targeting that project. The `users/{uid}.projects` map is a per-user favorites / recently-used list for UI display, **not** an access-control mechanism.

## Problem

The concept of a "project" has existed across the codebase without a canonical data model. Three symptoms of this gap:

1. **`user.projects` is used as both favorites and ACL.** Today, `synchestra-cloud/internal/api/sessions.go` rejects session creation with `"project not owned by caller"` when `user.projects[project_id]` is missing. That check treats a per-user map as an access-control list, which means a user who has been granted project access by a collaborator cannot exercise it until they manually add the project to their own `user.projects` map. Ownership and favoriting are conflated.
2. **No top-level project document exists.** Runners declare `project_ids` (the projects they serve), but there is no `projects/{project_id}` Firestore document describing the project itself. Project membership cannot be granted, revoked, or enumerated except by reading every user document.
3. **Membership is unilateral.** Because "is user U a member of project P?" is answered by reading `users/U.projects[P]`, users effectively grant themselves membership. There is no server-enforced grant / revoke semantics.

This feature formalizes the project data model and ACL, and establishes the correct split: `user.projects` is a UI-display favorites list; `project.user_ids` is the authoritative ACL.

## Behavior

### Project data model

Projects live at `projects/{project_id}` in synchestra-cloud's Firestore.

```
Project {
  project_id: string        // canonical ID — see "Project ID format"
  owner_uid: string         // creator; default admin; always in user_ids
  user_ids: []string        // ACL: UIDs permitted to use the project (includes owner)
  name: string              // human-readable display name
  description: string       // optional
  created_at: timestamp
  updated_at: timestamp
}
```

The shape deliberately mirrors `runners/{id}` and `hosts/{id}` (both of which already use `owner_uid` + `user_ids: []string`). This is the consistent Synchestra ACL pattern.

#### REQ: project-acl-is-user_ids

A user is permitted to operate on a project if and only if the user's UID is present in `project.user_ids`. All authorization checks that previously relied on `user.projects[id]` MUST be migrated to read `project.user_ids`.

#### REQ: user-projects-is-favorites

The `users/{uid}.projects` map is a per-user favorites / recently-used list for UI ordering and quick access. It MUST NOT be used as an ACL primitive. The map MAY be stale, empty, or missing projects the user has access to — none of these conditions affect authorization. A user MUST NOT be denied access solely because a project is absent from their `user.projects` map.

#### REQ: owner-in-user_ids

When a project is created, the creator's UID MUST be placed in both `owner_uid` and `user_ids`. Removing the owner's UID from `user_ids` while `owner_uid` still references it is a data-integrity error.

### Project ID format

Project IDs are stable strings of the form `{source}/{namespace}/{slug}` — e.g., `github.com/acme/webapp`. The full string is used in UI paths, authorization checks, and runner scoping.

#### REQ: project-id-stable

A project's ID is stable across its lifetime. Renaming a project changes its `name`, never its `project_id`.

### Membership management

Membership is managed server-side via synchestra-cloud API endpoints (exact endpoint shape is out of scope for this spec — documented here to establish the responsibility model).

#### REQ: membership-grant

An existing member of a project (any UID in `project.user_ids`) MAY grant membership to another user by appending the target UID to `project.user_ids`. Grant is effective immediately for new authorization checks.

#### REQ: membership-revoke

A member MAY remove another UID from `project.user_ids`. Revocation is effective immediately for new sessions. Sessions already running are NOT terminated by revocation — session termination is an independent operation.

#### REQ: owner-revoke-protection

The `owner_uid` MUST NOT be removed from `user_ids` by a normal revoke. Transferring ownership is a separate operation that updates both fields atomically.

### Runner / host scoping against projects

Runners and hosts declare `project_ids: []string` to scope which projects they serve. This is a scoping concern, orthogonal to project ACL.

#### REQ: project_ids-literal

Today, each entry in a runner or host's `project_ids` MUST be either a literal project ID or the wildcard `"*"` (meaning all projects). A runner serves a project if either the literal project ID is in its `project_ids` or `"*"` is.

#### REQ: project_ids-regex-future

A future extension of the matcher MAY treat `project_ids` entries as regular expressions matching the full project ID. Example: `github.com/acme/(.+)` would match any project under the `acme` GitHub organization. Implementation of regex support is **out of scope for this spec's first delivery** and is recorded here to establish forward-compatibility:

- When regex support lands, plain strings continue to match literally (backwards compatibility).
- Regex patterns are anchored against the full project ID (implicit `^...$`).
- The `"*"` wildcard continues to be recognized as the any-project shortcut.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [host-auth](../host-auth/README.md) | Hosts and runners have their own `user_ids` ACL. A user accessing a project-bound resource is checked against both `project.user_ids` and the runner / host's `user_ids`. |
| [runner](../runner/README.md) | Runners declare `project_ids` (scoping). A session binds a project to a runner; the runner's `project_ids` must include the session's project (literal match or wildcard). |
| [channels](../channels/README.md) | Session messages carry the session's `project_id` (when present); downstream authorization uses the project ACL documented here. |

## Dependencies

None. This feature formalizes an existing implicit concept in the codebase.

## Acceptance Criteria

### AC: projects-collection-exists

**Requirements:** project#req:project-acl-is-user_ids, project#req:owner-in-user_ids

A top-level `projects/{project_id}` Firestore collection exists with the documented shape. Creating a project initializes `owner_uid` and places the owner's UID in `user_ids`. Reading the doc returns the ACL.

### AC: acl-migrated-off-user-map

**Requirements:** project#req:project-acl-is-user_ids, project#req:user-projects-is-favorites

All authorization checks in synchestra-cloud that previously read `user.projects[id]` are migrated to read `project.user_ids`. A user who has been added to `project.user_ids` by a collaborator (without updating their own `user.projects` map) can successfully create a session in that project.

### AC: membership-grant-revoke

**Requirements:** project#req:membership-grant, project#req:membership-revoke, project#req:owner-revoke-protection

A member can grant membership to another user by appending to `user_ids`; the new member gains access immediately. A member can revoke another user by removing their UID; the revoked user is denied on next session creation but running sessions are not terminated. The owner's UID cannot be removed from `user_ids` by the revoke operation.

### AC: project_ids-regex-reserved

**Requirements:** project#req:project_ids-literal, project#req:project_ids-regex-future

The runner/host `project_ids` matcher accepts literal IDs and the `"*"` wildcard today. The spec documents regex patterns as a forward-compatible future extension. No implementation of regex matching is required for this feature's first delivery.

## Outstanding Questions

- Should `user_ids` evolve into a role-based structure (e.g., a `projects/{id}/members/{uid}` subcollection with role fields like `owner`, `admin`, `member`) as collaboration needs grow? The flat list is adequate for current scale but may not carry the semantics needed for team deployments. (Same question applies to `runners.user_ids` and `hosts.user_ids`.)
- Should projects optionally be private (invite-only, current implicit default) vs discoverable (any user with the ID can request access)? No discovery model exists today.
- Should a migration script convert every distinct `user.projects[id]` entry across all users into a `projects/{id}` document with that user in `user_ids`, or should projects bootstrap lazily on first access under the new model?
- When the regex extension to `project_ids` is delivered, should it support negative patterns (exclusions) or only inclusions?
- Should host-level authorization cache recent ACL denials to reject repeat attempts at the edge before hitting synchestra-cloud? See [fix-auth plan](../../../../synchestra-cloud/spec/plans/fix-auth/README.md) for current position.

---
*This document follows the https://specscore.md/feature-specification*
