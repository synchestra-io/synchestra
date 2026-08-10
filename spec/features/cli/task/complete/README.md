# Command: `synchestra task complete`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/task/complete?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/task/complete?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/task/complete?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/task/complete?op=request-change) |
**Parent:** [task](../README.md)
**Skill:** [task: complete](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/complete.md)

## Synopsis

```
synchestra task complete --project <project_id> --task <task_path> [--summary <text>]
```

## Description

Marks a task as completed. The command transitions the task from `in_progress` to `completed`, recording a timestamp and an optional summary of what was accomplished.

This is what an agent calls when it finishes work successfully. The `--current in_progress` guard is applied implicitly — the command fails with exit code `1` if the task is not currently `in_progress`.

Completion is atomic: the CLI commits the status change and pushes to the state repository. If the push fails due to a remote conflict, the CLI pulls and checks whether the task is still in `in_progress`. If yes, it retries. If no (the status changed), it exits with code `1`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--summary`](_args/summary.md) | No | Brief description of what was accomplished |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task completed successfully |
| `1` | Status conflict — task is no longer `in_progress` (someone changed it) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition (task is not in `in_progress` status) |

## Behaviour

1. Pull latest state from the state repository
2. Verify the task exists and is in `in_progress` status
3. Update the task status to `completed` with timestamp and optional summary
4. Commit and push
5. On push conflict: pull, re-check status, retry or fail

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
