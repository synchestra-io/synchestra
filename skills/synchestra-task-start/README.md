---
name: synchestra-task-start
description: Starts a claimed task by transitioning it to in-progress. Use after claiming a task and before beginning actual work.
---

# Skill: synchestra-task-start

Transition a claimed task to in-progress. This is what an agent calls after claiming a task and before beginning actual work.

**CLI reference:** [synchestra task start](../../spec/features/cli/task/start/README.md)

## When to use

- You have successfully claimed a task (exit code `0` from `synchestra task claim`)
- You are ready to begin working on the task
- Before writing any code or making any changes

## Command

```bash
synchestra task start \
  --project <project_id> \
  --task <task_path>
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task started successfully | Proceed with the work |
| `1` | Conflict — task state changed since last read | Re-read the task state and decide how to proceed |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition (task is not `claimed`) | The task may not have been claimed yet, or it is already in progress, completed, or blocked — check the current state |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Start a claimed task

```bash
synchestra task start --project synchestra --task implement-cli
```

### Start a nested subtask

```bash
synchestra task start --project synchestra --task implement-cli/parse-arguments
```

### Handle a failed start

```bash
synchestra task start --project my-service --task fix-auth-bug
# Exit code 4: "Invalid state transition: task fix-auth-bug is in pending state, expected claimed"
# → Claim the task first with synchestra task claim
```

## Notes

- Starting is atomic — it commits a status change and pushes to the state repository. If the push fails due to a conflict, the start fails.
- The command implicitly uses a `--current claimed` guard. Only tasks in `claimed` status can be started.
- This command does not require `--run` or `--model` parameters — those were already recorded during the claim step.

## Outstanding Questions

None at this time.
