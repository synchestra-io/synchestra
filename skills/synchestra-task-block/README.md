# Skill: synchestra-task-block

Mark a task as blocked when you cannot proceed. This tells the orchestrator and other agents that the task needs something resolved before work can continue.

**CLI reference:** [synchestra task block](../../spec/features/cli/task/block/README.md)

## When to use

- You discover a dependency on another task that is not yet complete
- You need information that is not available (e.g., a design decision, credentials, clarification)
- You encounter a problem that requires human intervention or a decision from another agent
- You cannot make further progress on the task for any reason

## Command

```bash
synchestra task block \
  --project <project_id> \
  --task <task_path> \
  --reason <reason>
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| [`--reason`](../../spec/features/cli/task/_args/reason.md) | Yes | Explanation of what is blocking — be specific enough for another agent or human to unblock it |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task blocked successfully | Stop working on the task and move on |
| `1` | Conflict — task status changed remotely | Pull latest state and reassess |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition (task is not `in_progress`) | The task may not have been started, or was already blocked/completed — check current status with `synchestra task status` |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Block a task due to a dependency

```bash
synchestra task block --project synchestra --task implement-cli/parse-arguments \
  --reason "Depends on task implement-cli/define-schema which defines the argument format — cannot proceed until schema is finalized"
```

### Block a task needing a human decision

```bash
synchestra task block --project my-service --task migrate-database \
  --reason "Migration requires choosing between PostgreSQL and MySQL — need a decision from the team before writing migration scripts"
```

### Block a task due to missing information

```bash
synchestra task block --project my-service --task integrate-api \
  --reason "API credentials for the staging environment are not available — need ops to provision and share them"
```

## Notes

- The task must be `in_progress` to be blocked. The command implicitly uses a `--current in_progress` guard.
- The `--reason` parameter is required and should be actionable. A vague reason like "stuck" is not helpful — explain what is needed to unblock.
- Blocking is atomic — it commits the status change and pushes to the project repo.
- After blocking a task, the agent should stop working on it and pick up another task or exit.
