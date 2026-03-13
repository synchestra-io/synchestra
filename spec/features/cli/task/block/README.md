# Command: `synchestra task block`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-block](../../../../../skills/synchestra-task-block/README.md)

## Synopsis

```
synchestra task block --project <project_id> --task <task_path> --reason <reason>
```

## Description

Transitions a task from `in_progress` to `blocked`, recording why the agent cannot proceed. The command implicitly guards on `--current in_progress` — if the task is not in `in_progress` status, the transition is rejected.

This command is used when an agent discovers it cannot continue due to a dependency on another task, missing information, or a decision that needs to be made by a human or another agent. The reason must be specific enough for someone else to understand what is needed to unblock the task.

The transition is atomic: the CLI commits the status change and pushes to the project repo.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../_args/reason.md) | Yes | Explanation of what is blocking the task — must be specific enough for another agent or human to unblock it |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task blocked successfully |
| `1` | Conflict — task status changed since read (e.g., remote update) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition — task is not `in_progress` |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `in_progress` status
3. Update the task status to `blocked` with the reason and timestamp
4. Commit and push
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
