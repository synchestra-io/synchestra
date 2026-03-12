# Skill: synchestra-task-list

List tasks in a project to find available work, check what's in progress, or review overall task state.

**CLI reference:** [synchestra task list](../../spec/features/cli/task/list/README.md)

## When to use

- **Finding available work:** List `pending` tasks to pick one to claim
- **Checking progress:** See which tasks are `in_progress`, `completed`, or `blocked`
- **Surveying a project:** Get an overview of all tasks and their current states
- **Machine-readable output:** Use `--format json` or `--format yaml` when you need to parse the results programmatically

## Command

```bash
synchestra task list \
  --project <project_id> \
  [--status <status>] \
  [--format <format>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| `--status` | No | Filter by task status (e.g., `pending`, `in_progress`, `claimed`, `completed`, `failed`, `blocked`, `aborted`) |
| `--format` | No | Output format: `table` (default), `json`, `yaml` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Parse the output and proceed |
| `2` | Invalid arguments | Check parameter values |
| `3` | Project not found | Verify the project identifier |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### List all tasks in a project

```bash
synchestra task list --project synchestra
# PATH                              STATUS        RUN    MODEL    UPDATED_AT
# implement-cli                     in_progress   4821   sonnet   2026-03-12T10:45:00Z
# implement-cli/parse-arguments     completed     4821   sonnet   2026-03-12T11:02:00Z
# implement-cli/validate-config     pending        —      —       2026-03-12T09:00:00Z
# fix-auth-bug                      claimed       9933   opus     2026-03-12T12:15:00Z
# write-tests                       blocked       5501   haiku    2026-03-12T11:30:00Z
```

### Find pending tasks to claim

```bash
synchestra task list --project synchestra --status pending
# PATH                              STATUS    RUN    MODEL    UPDATED_AT
# implement-cli/validate-config     pending    —      —       2026-03-12T09:00:00Z
```

### Get task list as JSON for programmatic use

```bash
synchestra task list --project synchestra --format json
```

### Check what's currently in progress

```bash
synchestra task list --project my-service --status in_progress
```

## Notes

- This is a read-only command. It pulls latest state but does not modify anything.
- The output includes task path, status, assigned run, model, and `updated_at` for each task.
- Use `--format json` or `--format yaml` when piping output to other tools or parsing results in a script.
- To get detailed information about a single task, use `synchestra task status` instead.
