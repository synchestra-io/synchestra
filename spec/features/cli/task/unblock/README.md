# Command: `synchestra task unblock`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-unblock](../../../../../skills/synchestra-task-unblock/README.md)

## Synopsis

```
synchestra task unblock --project <project_id> --task <task_path> [--reason <text>]
```

## Description

Resumes a blocked task by transitioning it from `blocked` to `in_progress`. Called when the blocking condition has been resolved and work can continue.

Implicitly uses `--current blocked` as a guard — fails if the task is not currently blocked.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier |
| `--task` | Yes | Task path using `/` as separator |
| `--reason` | No | What resolved the blocker |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task unblocked successfully |
| `1` | Status mismatch — task is not in `blocked` state |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `blocked` status
3. Update status to `in_progress`, record reason and timestamp
4. Commit and push
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
