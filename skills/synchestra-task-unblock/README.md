# Skill: synchestra-task-unblock

Resume a blocked task when the blocking condition has been resolved.

**CLI reference:** [synchestra task unblock](../../spec/features/cli/task/unblock/README.md)

## When to use

- A dependency that was blocking the task has been completed
- Missing information has been provided
- A decision that was pending has been made
- An external blocker has been resolved

## Command

```bash
synchestra task unblock \
  --project <project_id> \
  --task <task_path> \
  [--reason <text>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--reason`](../../spec/features/cli/task/_args/reason.md) | No | What resolved the blocker |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task unblocked, now `in_progress` | Continue working on the task |
| `1` | Status mismatch — task is not blocked | Re-query the status to see current state |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition | Check valid transitions |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Unblock after dependency completes

```bash
synchestra task unblock --project synchestra --task implement-api/auth-endpoint \
  --reason "Dependency implement-api/user-model completed"
```

### Unblock after decision made

```bash
synchestra task unblock --project my-service --task migrate-database \
  --reason "Team decided on PostgreSQL over MySQL"
```

### Handle a task that's not blocked

```bash
synchestra task unblock --project synchestra --task implement-cli
# Exit code 1: "Status mismatch: expected blocked, actual is in_progress"
# → Task was already unblocked by someone else
```

## Notes

- The implicit `--current blocked` guard prevents accidentally transitioning a task that's not blocked.
- After unblocking, the task returns to `in_progress` — the same agent or a new one can continue the work.
- Include a reason describing what resolved the blocker so there's an audit trail of how blockers get resolved.
