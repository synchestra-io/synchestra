# Skill: synchestra-task-list

List tasks in a project to find available work, check what's in progress, or review overall task state.

**CLI reference:** [synchestra task list](../../spec/features/cli/task/list/README.md)

## When to use

- **Finding available work:** List `queued` tasks to pick one to claim
- **Checking progress:** See which tasks are `in_progress`, `completed`, or `blocked`
- **Surveying a project:** Get an overview of all tasks and their current states
- **Selective output:** Use `--fields` to get only the data you need, `--format` to control structure

## Command

```bash
synchestra task list \
  --project <project_id> \
  [--status <status>] \
  [--format <format>] \
  [--fields <fields>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--status`](../../spec/features/cli/task/list/_args/status.md) | No | Filter by task status (e.g., `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`) |
| [`--format`](../../spec/features/cli/task/_args/format.md) | No | Output format: `yaml` (default), `json`, `md`, `csv` |
| [`--fields`](../../spec/features/cli/task/list/_args/fields.md) | No | Comma-separated list of fields to include (e.g., `path,status,model`). Defaults to all fields |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Parse the output and proceed |
| `2` | Invalid arguments | Check parameter values |
| `3` | Project not found | Verify the project identifier |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### List all tasks (default YAML)

```bash
synchestra task list --project synchestra
# - path: implement-cli
#   status: in_progress
#   title: Implement CLI
#   run: 4821
#   model: sonnet
#   updated_at: 2026-03-12T10:45:00Z
# - path: fix-auth-bug
#   status: claimed
#   title: Fix auth bug
#   run: 9933
#   model: opus
#   updated_at: 2026-03-12T12:15:00Z
# ...
```

### Find queued tasks to claim

```bash
synchestra task list --project synchestra --status queued
```

### Get specific fields as CSV

```bash
synchestra task list --project synchestra --status queued --fields path,title,depends_on --format csv
# path,title,depends_on
# write-tests,Write tests,implement-api
# deploy-staging,Deploy staging,"implement-api,write-tests"
```

### JSON for programmatic use

```bash
synchestra task list --project synchestra --format json --fields path,status
```

### Markdown table for embedding

```bash
synchestra task list --project synchestra --format md
```

## Notes

- This is a read-only command. It pulls latest state but does not modify anything.
- Default format is YAML — structured and easy for both agents and humans to parse.
- Use `--fields` to reduce output to only what you need, especially when piping to other tools.
- The `md` format renders a markdown table matching the [task status board](../../spec/features/task-status-board/README.md) format.
- To get detailed information about a single task (description, parent chain, context), use [`task info`](../../spec/features/cli/task/info/README.md) instead.
