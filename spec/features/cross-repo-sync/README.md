# Feature: Cross-Repo Sync

**Status:** Conceptual

## Summary

When a task requires changes across multiple repositories (e.g., updating an API endpoint in the backend and consuming it in the frontend), Synchestra coordinates the work by decomposing the task, reserving branch names, and managing the integration lifecycle across repos.

## Problem

Multi-repo changes are one of the hardest coordination problems in software development — even for humans. For AI agents, it's worse: each agent session is scoped to one repo, has no awareness of the other repo's state, and has no protocol for synchronizing changes that must land together.

The typical failure mode: backend merges first, frontend hasn't started, and now there's a broken API contract in production.

## Proposed Behavior

When Synchestra detects or is told that a task spans multiple repositories, it decomposes the work into a coordinated set of sub-tasks:

### Example: Add a field to an API response

**Parent task:** "Add `avatar_url` field to the user profile endpoint response and display it in the frontend."

Synchestra creates:

1. **Update interface/API specification** — Modify the shared API spec (OpenAPI, protobuf, or shared types) to include the new field. A branch name is reserved: `synchestra/add-avatar-url-to-user-profile`.
2. **Update backend** — Implement the change in the backend repo using the reserved branch name.
3. **Update frontend** — Implement the change in the frontend repo using the reserved branch name.
4. **Integration testing and merge** — Run integration tests across both branches, merge both into main, and update all task statuses.

```
Parent Task: Add avatar_url to user profile
├── Sub-task 1: Update API spec          → branch: synchestra/add-avatar-url-to-user-profile
├── Sub-task 2: Implement backend        → branch: synchestra/add-avatar-url-to-user-profile (backend repo)
├── Sub-task 3: Implement frontend       → branch: synchestra/add-avatar-url-to-user-profile (frontend repo)
└── Sub-task 4: Integration test & merge → depends on 1, 2, 3
```

### Branch naming convention

Reserved branch names follow the pattern:
```
synchestra/{task-slug}
```

The same branch name is used across all affected repositories, making it easy to identify related changes across repos.

### Task dependencies

- Sub-task 1 (spec update) must complete before sub-tasks 2 and 3 can start.
- Sub-tasks 2 and 3 (backend/frontend implementation) can run in parallel.
- Sub-task 4 (integration) depends on all previous sub-tasks completing.

### Merge strategy

The integration sub-task is responsible for:
1. Running cross-repo integration tests (if configured)
2. Merging branches in the correct order (spec first, then backend, then frontend — or as configured)
3. Updating task statuses across all sub-tasks and the parent task
4. Cleaning up branches

## Configuration (Draft)

A cross-repo sync specification file defines which repositories are involved and how they relate:

```yaml
# To be designed — this is a placeholder structure
repos:
  api-spec:
    url: github.com/org/api-spec
    role: contract  # Changes here gate other repos
  backend:
    url: github.com/org/backend
    depends_on: [api-spec]
  frontend:
    url: github.com/org/frontend
    depends_on: [api-spec]

merge_order:
  - api-spec
  - backend
  - frontend

branch_prefix: synchestra/
```

## Open Design Decisions

- Where does the cross-repo spec file live? In the Synchestra project repo? Per-project config?
- How does Synchestra get write access to multiple repos? GitHub App installation? User's OAuth token?
- What happens when one repo's branch merges successfully but another fails? Rollback? Block?
- How are cross-repo integration tests configured and where do they run?
- Should the branch naming convention be configurable or strictly enforced?
- How does this interact with repos that have branch protection rules?

## Outstanding Questions

- What does the full specification file format look like? (Needs dedicated design work.)
- How does Synchestra handle repos with different branching models (e.g., one uses trunk-based, another uses gitflow)?
- What is the story for cross-repo changes that require database migrations or infrastructure changes?
- How does the claim-and-push protocol work when the "push" needs to happen to a different repo than the Synchestra repo?
