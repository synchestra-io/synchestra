# Skill: synchestra-task-status

Query or update the status of a task. Use this to check task state, report progress, mark completion, or handle failures.

**CLI reference:** [synchestra task status](../../spec/features/cli/task/status/README.md)

## When to use

- **Before starting work:** Check the task status and whether `abort_requested` is set
- **After claiming:** Transition from `claimed` to `in_progress` when you begin actual work
- **On completion:** Transition from `in_progress` to `completed`
- **On failure:** Transition from `in_progress` to `failed` with a reason
- **When blocked:** Transition to `blocked` with a description of what's blocking
- **Periodically:** Check for `abort_requested` during long-running work

## Command

### Query status

```bash
synchestra task status \
  --project <project_id> \
  --task <task_path>
```

### Update status

```bash
synchestra task status \
  --project <project_id> \
  --task <task_path> \
  --current <expected_status> \
  --new <target_status> \
  [--reason <text>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `task1/subtask2`) |
| [`--current`](../../spec/features/cli/task/status/_args/current.md) | For update | Expected current status — fails if actual status differs |
| [`--new`](../../spec/features/cli/task/status/_args/new.md) | For update | Target status to transition to |
| [`--reason`](../../spec/features/cli/task/_args/reason.md) | For `failed`/`blocked` | Why the task failed or what's blocking it |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Continue with your work |
| `1` | Status mismatch — someone changed the status since you last checked | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition | Check the valid transitions — you may be skipping a step |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Check task status

```bash
synchestra task status --project synchestra --task implement-cli/parse-arguments
# status: in_progress
# run: 4821
# model: sonnet
# claimed_at: 2026-03-12T10:32:00Z
# updated_at: 2026-03-12T10:45:00Z
# abort_requested: false
```

### Start working (claimed → in_progress)

```bash
synchestra task status --project synchestra --task implement-cli --current claimed --new in_progress
```

### Complete a task

```bash
synchestra task status --project synchestra --task implement-cli --current in_progress --new completed
```

### Report failure

```bash
synchestra task status --project synchestra --task implement-cli --current in_progress --new failed \
  --reason "Build fails due to missing dependency: github.com/foo/bar v2 not published"
```

### Handle a stale status

```bash
synchestra task status --project synchestra --task fix-bug --current in_progress --new completed
# Exit code 1: "Status mismatch: expected in_progress, actual is aborted"
# → The task was aborted while you were working. Do not continue.
```

### Check for abort request during work

```bash
synchestra task status --project synchestra --task long-running-task
# status: in_progress
# abort_requested: true
# → Wrap up, then transition to aborted:

synchestra task status --project synchestra --task long-running-task --current in_progress --new aborted \
  --reason "Abort requested by user"
```

## Notes

- The `--current` guard is essential for safe concurrent operation. Always provide it when updating — never update without knowing the current state.
- When you see `abort_requested: true`, stop work as soon as practical and transition to `aborted`.
- The `--reason` parameter is required for `failed` and `blocked` transitions. Include enough detail for another agent or human to understand what happened.
