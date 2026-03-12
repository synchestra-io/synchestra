# Skill: synchestra-claim-task

Claim a task so you can begin working on it. This is the first thing an agent does before starting any work on a Synchestra-managed task.

**CLI reference:** [synchestra task claim](../../spec/features/cli/task/claim/README.md)

## When to use

- You've been assigned or have selected a task to work on
- Before writing any code or making any changes for a task
- After reading the task description and confirming you can do the work

## Command

```bash
synchestra task claim \
  --project <project_id> \
  --task <task_path> \
  --run <run_id> \
  --model <model_id>
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| `--task` | Yes | Task path using `/` as separator (e.g., `task1`, `task1/subtask2`, `task1/subtask2/subtask3`) |
| `--run` | Yes | Unique identifier for this agent run (provided by the orchestrator or generated) |
| `--model` | Yes | Model being used for this work (e.g., `haiku`, `sonnet`, `opus`) |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task claimed successfully | Proceed with the work |
| `1` | Claim conflict — another agent claimed this task | Pick a different task or exit |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Task not found | Verify the project and task path |
| `4` | Invalid state transition (task is not in a claimable state) | The task may already be in progress, completed, or blocked — pick a different task |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Claim a top-level task

```bash
synchestra task claim --project synchestra --task implement-cli --run 4821 --model sonnet
```

### Claim a nested subtask

```bash
synchestra task claim --project synchestra --task implement-cli/parse-arguments --run 4821 --model haiku
```

### Handle a failed claim

```bash
synchestra task claim --project my-service --task fix-auth-bug --run 9933 --model opus
# Exit code 1: "Claim conflict: task fix-auth-bug was claimed by run 9901 at 2026-03-12T10:32:00Z"
# → Pick the next available task
```

## Notes

- Claiming is atomic — it commits a status change and pushes to the project repo. If the push fails due to a conflict, the claim fails.
- A claimed task must be followed by work. If you claim a task and cannot complete it, use `synchestra task release` to return it to the queue.
- The `--run` and `--model` parameters are recorded in the task status for auditability and debugging.
