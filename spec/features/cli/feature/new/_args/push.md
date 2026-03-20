# `--push`

Commit and push atomically to the remote repository.

| Field | Value |
|---|---|
| **Type** | Boolean (flag) |
| **Required** | No |
| **Default** | `false` |

## Supported by

- [`feature new`](../README.md)

## Description

When passed, all scaffolded and updated files are staged, committed, and pushed to the remote repository in a single atomic operation. Implies `--commit`.

On push conflict (exit code `1`): the CLI pulls the latest state, re-validates that the parent feature still exists and the target path is still free, then retries the commit and push. If preconditions no longer hold after the pull, the command fails.

This flag is useful for automated workflows and agent-driven feature creation where immediate synchronization is needed. For interactive use, omitting `--push` allows reviewing generated files before committing.

## Examples

```bash
# Scaffold, commit, and push atomically
synchestra feature new --title "Task Status Board" --push

# Equivalent to --commit --push
synchestra feature new --title "Task Status Board" --commit --push
```

## Outstanding Questions

None at this time.
