# Command: `synchestra task abort`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-abort](../../../../../skills/synchestra-task-abort/README.md)

## Synopsis

```
synchestra task abort --project <project_id> --task <task_path> [--reason <text>]
```

## Description

Requests that a task be aborted by setting the `abort_requested` flag on it. This does NOT change the task's status — the working agent is responsible for seeing the flag, wrapping up, and calling `synchestra task aborted` to complete the transition.

This command is called by humans or the orchestrator, not by the working agent itself. The task must be in `claimed` or `in_progress` status; otherwise the command fails with exit code `4`.

The `--reason` parameter is optional but recommended — it tells the working agent why the abort was requested so it can wrap up appropriately.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier |
| [`--task`](../$args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../$args/reason.md) | No | Why the abort is being requested — helps the working agent understand context when wrapping up |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Abort requested successfully (`abort_requested` flag set) |
| `1` | Conflict — task status changed since read (e.g., remote update) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition — task is not in `claimed` or `in_progress` status |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `claimed` or `in_progress` status
3. Set the `abort_requested` flag to `true`, with optional reason and timestamp
4. Commit and push
5. On push conflict: pull, re-check status guard, retry or fail

## Outstanding Questions

None at this time.
