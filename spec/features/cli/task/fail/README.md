# Command: `synchestra task fail`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-fail](../../../../../skills/synchestra-task-fail/README.md)

## Synopsis

```
synchestra task fail --project <project_id> --task <task_path> --reason <text>
```

## Description

Marks a task as failed. This is what an agent calls when it cannot complete the work. The command transitions the task from `in_progress` to `failed`, recording the reason and timestamp.

The `--reason` parameter is required and must explain why the task failed with enough detail for another agent or human to understand what happened and decide on next steps.

The command implicitly uses a `--current in_progress` guard — it will fail with exit code `1` if the task is not currently `in_progress`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier |
| [`--task`](../$args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../$args/reason.md) | Yes | Why the task failed — must include enough detail for another agent or human to understand what happened |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task marked as failed successfully |
| `1` | Status mismatch — task is not `in_progress` (someone changed it) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `in_progress` status
3. Update the task status to `failed` with reason and timestamp
4. Commit and push
5. On push conflict: pull, re-check `in_progress` guard, retry or fail

## Outstanding Questions

None at this time.
