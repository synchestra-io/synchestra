# Command: `synchestra task claim`

**Parent:** [task](../README.md)
**Skill:** [synchestra-claim-task](../../../../../skills/synchestra-claim-task/README.md)

## Synopsis

```
synchestra task claim --project <project_id> --task <task_path> --run <run_id> --model <model_id>
```

## Description

Claims a queued task so an agent can begin working on it. The command transitions the task from `queued` to `claimed`, recording the agent's run ID, model, and timestamp.

Claiming is atomic: the CLI commits the status change and pushes to the project repo. If the push fails due to a remote conflict, the CLI pulls and checks whether the task is still claimable. If yes, it retries. If no (another agent claimed it), it exits with code `1`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--run`](_args/run.md) | Yes | Unique identifier for this agent run |
| [`--model`](_args/model.md) | Yes | Model being used (e.g., `haiku`, `sonnet`, `opus`) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task claimed successfully |
| `1` | Claim conflict — another agent claimed this task |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Task is not in a claimable state (not `queued`) |

## Behaviour

1. Pull latest state from the project repo
2. Verify the task exists and is in `queued` status
3. Update the task status to `claimed` with run ID, model, and timestamp
4. Commit and push
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
