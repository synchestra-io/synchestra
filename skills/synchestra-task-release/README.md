# Skill: synchestra-task-release

Release a claimed task back to the queue when you decide not to work on it. This returns the task to `queued` so another agent can claim it.

**CLI reference:** [synchestra task release](../../spec/features/cli/task/release/README.md)

## When to use

- You claimed a task but realise you are the wrong model for the job
- You lack a capability required to complete the task
- Higher-priority work appeared and you need to free this task for another agent
- You claimed a task by mistake

## Command

```bash
synchestra task release \
  --project <project_id> \
  --task <task_path> \
  [--reason <text>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| [`--reason`](../../spec/features/cli/task/_args/reason.md) | No | Why you are releasing the task — helps other agents or humans understand context |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task released successfully | No further action needed — the task is back in the queue |
| `1` | Status mismatch — task is not `claimed` | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition | Check the valid transitions — the task may not be in `claimed` state |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Release a task you lack capability for

```bash
synchestra task release --project synchestra --task implement-cli/generate-ui \
  --reason "This task requires front-end expertise; releasing so a more suitable agent can pick it up"
```

### Release a task for higher-priority work

```bash
synchestra task release --project my-service --task refactor-logging \
  --reason "Higher-priority security fix appeared; releasing this task to focus on the urgent work"
```

### Handle a status mismatch

```bash
synchestra task release --project synchestra --task fix-bug \
  --reason "Claimed by mistake"
# Exit code 1: "Status mismatch: expected claimed, actual is in_progress"
# → The task has already moved to in_progress. Do not attempt to release it.
```

## Notes

- The `--reason` parameter is optional but recommended. Include enough context for another agent or human to understand why you released the task.
- This command implicitly guards on `--current claimed`. You can only release a task that is currently claimed.
- The transition is atomic — it commits the status change and pushes to the project repo.
- Releasing a task returns it to `queued` status, clearing the assignee so any agent can claim it.

## Outstanding Questions

None at this time.
