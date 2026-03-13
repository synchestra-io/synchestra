# Skill: synchestra-task-complete

Mark a task as completed after finishing work successfully. This is the last thing an agent does when it has accomplished everything the task required.

**CLI reference:** [synchestra task complete](../../spec/features/cli/task/complete/README.md)

## When to use

- You have finished all work for the task successfully
- All changes are ready and tests pass (if applicable)
- You want to signal that the task is done

## Command

```bash
synchestra task complete \
  --project <project_id> \
  --task <task_path> \
  [--summary <text>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/$args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/$args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| [`--summary`](../../spec/features/cli/task/complete/$args/summary.md) | No | Brief description of what was accomplished (e.g., `"Implemented argument parser with validation"`) |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task completed successfully | You're done — no further action needed |
| `1` | Status conflict — task is no longer `in_progress` | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition (task is not in `in_progress` status) | Check the task's current status — it may have been aborted or already completed |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Complete a task

```bash
synchestra task complete --project synchestra --task implement-cli
```

### Complete a task with a summary

```bash
synchestra task complete --project synchestra --task implement-cli/parse-arguments \
  --summary "Implemented argument parser with validation and help text generation"
```

### Handle a status conflict

```bash
synchestra task complete --project synchestra --task fix-auth-bug
# Exit code 1: "Status conflict: expected in_progress, actual is aborted"
# → The task was aborted while you were working. Do not continue.
```

## Notes

- The `in_progress` guard is applied implicitly — you do not need to pass `--current in_progress`.
- Completion is atomic — it commits a status change and pushes to the project repo. If the push fails due to a conflict, the command retries or fails.
- Use the `--summary` parameter to leave a brief record of what was accomplished. This helps other agents and humans understand what was done without reading the full diff.
