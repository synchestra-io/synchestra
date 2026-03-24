# Command: `synchestra state pull`

**Parent:** [state](../README.md)
**Environment:** Coordination (Agent)

## Synopsis

```
synchestra state pull [--project <project_id>]
```

## Description

Pulls the latest state from the remote origin to local main. This is a manual, immediate operation — it ignores the project's sync policy and always executes.

After updating local main, active agent branches are rebased onto the updated main to incorporate remote changes.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No (autodetected) | Project identifier |

The CLI resolves the project's state repo path using the `state_repo` field in `synchestra-spec-repo.yaml`. When run from within the state repo directory itself, the CLI detects `synchestra-state-repo.yaml` and resolves the project directly.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — local main is up to date with origin |
| `1` | Conflict — a merge conflict occurred during agent branch rebase; repo is left in a clean state (conflict aborted) |
| `3` | Project not found |

## Behaviour

1. Fetch from origin
2. Fast-forward local main to match `origin/main`
3. Rebase active agent branches onto updated main (if any)
4. If rebase conflict: abort rebase, leave agent branch at its previous state, exit `1`

**Output:** Human-readable status to stdout (e.g., "Pulled 3 commits from origin", "Already up to date").

**Safety:** This command never leaves the repo in a dirty state. On conflict, the rebase is aborted and the repo is restored to its pre-pull state. The user can inspect and resolve manually.

## Outstanding Questions

None at this time.
