---
name: synchestra-task-abort
description: Requests abort of a claimed or in-progress task. Use when cancelling a task due to changed priorities or superseded work.
---

# Skill: synchestra-task-abort

Request that a task be aborted. This sets the `abort_requested` flag on the task so the working agent knows to wrap up — it does not change the task's status directly.

**CLI reference:** [synchestra task abort](../../spec/features/cli/task/abort/README.md)

## When to use

- You are the orchestrator or a human and need to cancel a task that is currently claimed or in progress
- Priorities have changed and the task is no longer needed
- The task has been superseded by another approach or task
- You need the agent working on this task to stop and release its resources

## Command

```bash
synchestra task abort \
  --project <project_id> \
  --task <task_path> \
  --reason <text>
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| [`--reason`](../../spec/features/cli/task/_args/reason.md) | No | Why the abort is being requested — helps the working agent understand context when wrapping up |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Abort requested successfully | The flag is set — the working agent will see it on its next status check and call `synchestra task aborted` |
| `1` | Conflict — task status changed since read | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition — task is not `claimed` or `in_progress` | Check the task's current status — it may have already completed, failed, or been aborted |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Abort a task that is no longer needed

```bash
synchestra task abort --project synchestra --task implement-cli/parse-arguments \
  --reason "Requirements changed — this subtask is no longer needed after the redesign"
```

### Abort a task without a reason

```bash
synchestra task abort --project my-service --task migrate-database
```

### Handle an invalid state

```bash
synchestra task abort --project synchestra --task fix-bug \
  --reason "Superseded by a different fix"
# Exit code 4: "Invalid state transition: task is in completed status"
# → The task already finished. No action needed.
```

## Notes

- This command sets the `abort_requested` flag — it does NOT change the task's status. The working agent is responsible for seeing the flag and calling `synchestra task aborted` to complete the transition.
- This command is for humans and the orchestrator, not for the working agent. If you are the agent working on the task, check for `abort_requested: true` via `synchestra task status` and then call `synchestra task aborted` to finalize.
- The task must be in `claimed` or `in_progress` status. If the task has already completed, failed, or been aborted, this command will fail with exit code `4`.
- The transition is atomic — it commits the flag change and pushes to the state repository.
- The `--reason` parameter is optional but recommended. It gives the working agent context about why the abort was requested.

## Outstanding Questions

None at this time.
