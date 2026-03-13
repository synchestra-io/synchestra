# Command: `synchestra task start`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-start](../../../../../skills/synchestra-task-start/README.md)

## Synopsis

```
synchestra task start --project <project_id> --task <task_path>
```

## Description

Transitions a claimed task to `in_progress`, signalling that the agent is beginning actual work. This is the step an agent takes after claiming a task and before making any changes.

The command implicitly guards on `--current claimed` — if the task is not in `claimed` status, the command fails with exit code `4`.

Like all mutation commands, `task start` is atomic: the CLI commits the status change and pushes to the project repo. If the push fails due to a remote conflict, the CLI pulls and checks whether the task is still startable. If yes, it retries. If no, it exits with code `1`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task started successfully |
| `1` | Conflict — task state changed since last read |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Task is not in a startable state (not `claimed`) |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `claimed` status
3. Update the task status to `in_progress` with timestamp
4. Commit and push
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
