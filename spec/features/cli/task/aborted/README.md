# Command: `synchestra task aborted`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-aborted](../../../../../skills/synchestra-task-aborted/README.md)

## Synopsis

```
synchestra task aborted --project <project_id> --task <task_path> [--reason <text>]
```

## Description

Marks a task as aborted. This is a **shorthand** called by the working agent after it sees `abort_requested: true` and has wrapped up its in-flight work. The command transitions the task from `claimed` or `in_progress` to `aborted`, recording a timestamp and an optional reason describing what was done before aborting.

The typical flow is:

1. A human or orchestrator calls `synchestra task abort` to request an abort.
2. The agent detects the `abort_requested` flag via `synchestra task status`.
3. The agent wraps up any in-flight work.
4. The agent calls `synchestra task aborted` to confirm the abort is complete.

The command implicitly guards that the current status is `claimed` or `in_progress` — it fails with exit code `1` if the task is in any other status. It also clears the `abort_requested` flag since the task is now in a terminal state.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../_args/reason.md) | No | Explain what was done before aborting (e.g., `"Reverted partial schema migration and restored backup"`) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task marked as aborted successfully |
| `1` | Status conflict/mismatch — task is not `claimed` or `in_progress` (someone changed it) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition (task is not in `claimed` or `in_progress` status) |

## Behaviour

1. Pull latest state from the state repository
2. Verify the task exists and is in `claimed` or `in_progress` status
3. Update the task status to `aborted` with timestamp and optional reason
4. Clear the `abort_requested` flag
5. Commit and push
6. On push conflict: pull, re-check status guard, retry or fail

## Outstanding Questions

None at this time.
