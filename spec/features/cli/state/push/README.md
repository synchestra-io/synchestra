# Command: `synchestra state push`

**Parent:** [state](../README.md)
**Environment:** Coordination (Agent)

## Synopsis

```
synchestra state push [--project <project_id>]
```

## Description

Pushes local main to the remote origin. This is a manual, immediate operation — it ignores the project's sync policy and always executes.

Before pushing, any pending agent branch commits are merged to local main to ensure all local work is included.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No (autodetected) | Project identifier |

The CLI resolves the project's state repo path using the `state_repo` field in `synchestra-spec-repo.yaml`. When run from within the state repo directory itself, the CLI detects `synchestra-state-repo.yaml` and resolves the project directly.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — origin is up to date with local main |
| `1` | Push conflict — origin has diverged; use `synchestra state sync` to resolve |
| `3` | Project not found |

## Behaviour

1. Merge any pending agent branch commits to local main
2. Push local main to origin
3. On push rejection: exit `1` with a message suggesting `synchestra state sync`

**Output:** Human-readable status to stdout (e.g., "Pushed 5 commits to origin", "Nothing to push").

**Safety:** This command never leaves the repo in a dirty state. On push conflict, no local state is modified — the push simply fails and the user is informed.

## Outstanding Questions

None at this time.
