# Skill: synchestra-task-aborted

Confirm that a task has been aborted after wrapping up in-flight work. This is the last thing an agent does after it sees `abort_requested: true` and has finished cleaning up.

**CLI reference:** [synchestra task aborted](../../spec/features/cli/task/aborted/README.md)

## When to use

- You detected `abort_requested: true` via `synchestra task status`
- You have wrapped up or rolled back any in-flight work
- You want to confirm the abort and move the task to a terminal state

## Command

```bash
synchestra task aborted \
  --project <project_id> \
  --task <task_path> \
  [--reason <text>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/$args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/$args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| [`--reason`](../../spec/features/cli/task/$args/reason.md) | No | Explain what was done before aborting (e.g., `"Reverted partial schema migration and restored backup"`) |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task marked as aborted successfully | You're done — no further action needed |
| `1` | Status conflict — task is not `claimed` or `in_progress` | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition (task is not in `claimed` or `in_progress` status) | Check the task's current status — it may have already been aborted or completed |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Confirm an abort

```bash
synchestra task aborted --project synchestra --task implement-cli
```

### Confirm an abort with a reason

```bash
synchestra task aborted --project synchestra --task implement-cli/parse-arguments \
  --reason "Reverted partial changes to argument parser; tests pass on clean state"
```

### Handle a status conflict

```bash
synchestra task aborted --project synchestra --task fix-auth-bug \
  --reason "Rolled back OAuth changes"
# Exit code 1: "Status conflict: expected claimed or in_progress, actual is completed"
# → The task was completed by another agent. Do not continue.
```

## Notes

- This is a **shorthand** — the typical flow is: `task abort` (by human/orchestrator) -> agent detects `abort_requested` via `task status` -> agent wraps up -> agent calls `task aborted`.
- The `claimed` / `in_progress` guard is applied implicitly — you do not need to pass `--current`.
- The command also clears the `abort_requested` flag since the task is now in a terminal state.
- The transition is atomic — it commits the status change and pushes to the project repo.
- Use the `--reason` parameter to describe what cleanup was performed before aborting. This helps other agents and humans understand what state the work was left in.

## Outstanding Questions

None at this time.
