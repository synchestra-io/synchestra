# Skill: synchestra-task-enqueue

**CLI reference:** [synchestra task enqueue](../../spec/features/cli/task/enqueue/README.md)

## When to use

Use this skill when a task has been fully defined during planning and is ready
for an agent to pick up. Enqueuing marks the task as available for discovery
and claiming.

## Command

```
synchestra task enqueue --project <project> --task <task>
```

## Parameters

| Parameter   | Required | Description                |
|-------------|----------|----------------------------|
| [`--project`](../../spec/features/cli/$args/project.md) | Yes      | Project identifier         |
| [`--task`](../../spec/features/cli/task/$args/task.md)    | Yes      | Task identifier to enqueue |

The command implicitly guards on `--current planning` -- it will only succeed
if the task is currently in `planning` status.

## Exit codes

| Code | Meaning                  | What to do                                                                 |
|------|--------------------------|----------------------------------------------------------------------------|
| 0    | Success                  | Task is now queued. Agents can discover it via `task list --status queued`. |
| 1    | Status mismatch          | The task is not in `planning` status. Check its current status first.      |
| 2    | Invalid arguments        | Verify that both `--project` and `--task` are provided and valid.          |
| 3    | Task not found           | Confirm the task identifier and project are correct.                       |
| 4    | Invalid state transition | The transition was rejected. Investigate the task's current state.         |

## Examples

### Basic enqueue

```shell
synchestra task enqueue --project my-project --task implement-auth
# exit 0 -- task is now queued
```

### Enqueue a subtask

```shell
synchestra task enqueue --project my-project --task implement-auth/write-tests
# exit 0 -- subtask is now queued
```

### Handle an already-queued task

```shell
synchestra task enqueue --project my-project --task implement-auth
# exit 1 -- task is not in planning status (already queued or further along)
```

## Notes

- The status change is performed as an atomic commit-and-push.
- Only tasks in `planning` status can be enqueued. Tasks in any other status
  will cause exit code 1.
- After enqueuing, agents discover available work with
  `synchestra task list --status queued` and claim a task with
  `synchestra task claim`.
