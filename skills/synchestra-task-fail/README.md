# Skill: synchestra-task-fail

Mark a task as failed when you cannot complete the work. This records why the task failed so another agent or human can understand what happened and decide on next steps.

**CLI reference:** [synchestra task fail](../../spec/features/cli/task/fail/README.md)

## When to use

- You've hit an unrecoverable error and cannot complete the task
- A dependency is missing, broken, or incompatible and you cannot work around it
- The task requirements are unclear or contradictory and you cannot proceed
- You've exhausted your approaches and the task remains incomplete

## Command

```bash
synchestra task fail \
  --project <project_id> \
  --task <task_path> \
  --reason <text>
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| `--task` | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| `--reason` | Yes | Why the task failed — include enough detail for another agent or human to understand what happened |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task marked as failed successfully | No further action needed — the failure is recorded |
| `1` | Status mismatch — task is not `in_progress` | Re-query the status and reassess |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition | Check the valid transitions — the task may not be in `in_progress` state |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Report a build failure

```bash
synchestra task fail --project synchestra --task implement-cli/parse-arguments \
  --reason "Build fails due to missing dependency: github.com/foo/bar v2 not published yet"
```

### Report an unclear requirement

```bash
synchestra task fail --project my-service --task redesign-auth \
  --reason "Task requires OAuth2 integration but does not specify which provider. Attempted Google and GitHub but both need configuration values not present in the project."
```

### Handle a status mismatch

```bash
synchestra task fail --project synchestra --task fix-bug \
  --reason "Tests still failing after three attempts"
# Exit code 1: "Status mismatch: expected in_progress, actual is aborted"
# → The task was aborted while you were working. Do not continue.
```

## Notes

- The `--reason` parameter is required. Include enough detail for another agent or human to understand what went wrong and what was tried.
- This command implicitly guards on `--current in_progress`. You can only fail a task that is currently in progress.
- The transition is atomic — it commits the status change and pushes to the project repo.
- If you are blocked but believe the task can be completed once the blocker is resolved, consider using `synchestra task status --current in_progress --new blocked --reason <text>` instead.
