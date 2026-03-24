# Command: `synchestra state sync`

**Parent:** [state](../README.md)
**Environment:** Coordination (Agent)

## Synopsis

```
synchestra state sync [--project <project_id>]
```

## Description

Full bidirectional sync — pull from origin then push local changes back. This is a manual, immediate operation — it ignores the project's sync policy and always executes.

Equivalent to running `synchestra state pull` followed by `synchestra state push`, with automatic conflict retry on push.

This is the recommended command when you want to ensure the local and remote state repos are fully synchronized.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No (autodetected) | Project identifier |

The CLI resolves the project's state repo path using the `state_repo` field in `synchestra-spec-repo.yaml`. When run from within the state repo directory itself, the CLI detects `synchestra-state-repo.yaml` and resolves the project directly.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — local and origin are synchronized |
| `1` | Unresolvable conflict — a merge conflict that could not be automatically resolved |
| `3` | Project not found |

## Behaviour

1. Pull from origin (same as [`state pull`](../pull/README.md))
2. Push to origin (same as [`state push`](../push/README.md))
3. On push conflict: pull again, re-merge, retry push
4. If conflict persists after retry: exit `1`

**Output:** Human-readable status to stdout combining pull and push results (e.g., "Pulled 3 commits, pushed 2 commits", "Already in sync").

**Safety:** This command never leaves the repo in a dirty state. On unresolvable conflict, all partial operations are rolled back and the repo is restored to its pre-sync state.

## Outstanding Questions

None at this time.
