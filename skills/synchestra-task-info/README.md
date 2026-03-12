# Skill: synchestra-task-info

Display the full context for a task — description, status, parent chain, siblings, outstanding questions, and linked feature spec. Use this to understand what a task requires before claiming or starting it.

**CLI reference:** [synchestra task info](../../spec/features/cli/task/info/README.md)

## When to use

- **Before claiming a task:** Read the full task context to decide if you can handle it
- **Before starting work:** Understand requirements, outstanding questions, and how the task fits into the larger hierarchy
- **When switching context:** Refresh your understanding of a task you previously set aside
- **When reviewing sibling tasks:** See what related work is happening at the same level

## Command

```bash
synchestra task info \
  --project <project_id> \
  --task <task_path> \
  [--format <text|json|yaml>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| `--task` | Yes | Task path using `/` as separator (e.g., `implement-cli/parse-arguments`) |
| `--format` | No | Output format: `text` (default), `json`, `yaml` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Read the output and proceed |
| `2` | Invalid arguments | Check parameter values |
| `3` | Task not found | Verify the project and task path |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### View full task context

```bash
synchestra task info --project synchestra --task implement-cli/parse-arguments
# Task: implement-cli/parse-arguments
# Status: pending
#
# Description:
#   Implement argument parsing for the synchestra CLI...
#
# Parent chain:
#   implement-cli (in_progress)
#
# Siblings:
#   implement-cli/setup-project (completed)
#   implement-cli/parse-arguments (pending)    ← you are here
#   implement-cli/run-command (pending)
#
# Outstanding questions:
#   - Should we support short flags (-p) in addition to long flags (--project)?
#
# Feature spec: spec/features/cli/README.md
```

### Get task info as JSON

```bash
synchestra task info --project synchestra --task implement-cli/parse-arguments --format json
```

### Decide whether to claim a task

```bash
# 1. Read the full context first
synchestra task info --project synchestra --task fix-auth-bug

# 2. If it looks manageable, claim it
synchestra task claim --project synchestra --task fix-auth-bug
```

## Notes

- This is a **read-only** command — it never mutates task state. Safe to call at any time.
- More detailed than `task status`. Use `task info` for the full picture; use `task status` when you only need to check or update the status.
- The parent chain provides hierarchical context so you understand where this task fits. Sibling tasks show related work at the same level for coordination awareness.
- Always review outstanding questions before starting work — they may affect your approach or require clarification first.
