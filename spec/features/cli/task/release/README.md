# Command: `synchestra task release`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-release](../../../../../skills/synchestra-task-release/README.md)

## Synopsis

```
synchestra task release --project <project_id> --task <task_path> [--reason <text>]
```

## Description

Releases a claimed task back to the queue. This is what an agent calls when it claimed a task but decides not to work on it — for example, wrong model, lacks the required capability, or higher-priority work appeared. The command transitions the task from `claimed` to `queued`, optionally recording a reason and timestamp.

The `--reason` parameter is optional but recommended. When provided it should explain why the agent is releasing the task so that other agents or humans have context.

The command implicitly uses a `--current claimed` guard — it will fail with exit code `1` if the task is not currently `claimed`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier |
| [`--task`](../$args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../$args/reason.md) | No | Why the task is being released — helps other agents or humans understand context |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task released successfully |
| `1` | Status mismatch — task is not `claimed` (someone changed it) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `claimed` status
3. Update the task status to `queued`, clearing the assignee, with optional reason and timestamp
4. Commit and push
5. On push conflict: pull, re-check `claimed` guard, retry or fail

## Outstanding Questions

None at this time.
