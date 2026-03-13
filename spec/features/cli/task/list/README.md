# Command: `synchestra task list`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-list](../../../../../skills/synchestra-task-list/README.md)

## Synopsis

```
synchestra task list --project <project_id> [--status <status>] [--format <format>] [--fields <fields>]
```

## Description

Lists tasks in a project. By default, all tasks are shown in YAML format with all fields. Use `--status` to filter by task status, `--format` to control the output format, and `--fields` to select specific fields.

This is a read-only command. It pulls the latest state from the project repo but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--status`]($args/status.md) | No | Filter by task status (e.g., `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`) |
| [`--format`](../$args/format.md) | No | Output format: `yaml` (default), `json`, `md`, `csv` |
| [`--fields`]($args/fields.md) | No | Comma-separated list of fields to include (e.g., `path,status,model`). Defaults to all fields |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Project not found |

## Behaviour

1. Pull latest state from the project repo
2. Read all tasks in the project
3. If `--status` is provided, filter tasks to only those matching the given status
4. Output the task list in the requested format

### Available fields

`path`, `status`, `title`, `run`, `model`, `requester`, `depends_on`, `branch`, `claimed_at`, `updated_at`, `abort_requested`

### YAML output (default)

```yaml
- path: implement-cli
  status: in_progress
  title: Implement CLI
  run: 4821
  model: sonnet
  updated_at: 2026-03-12T10:45:00Z
- path: implement-cli/parse-arguments
  status: completed
  title: Parse arguments
  run: 4821
  model: sonnet
  updated_at: 2026-03-12T11:02:00Z
- path: fix-auth-bug
  status: claimed
  title: Fix auth bug
  run: 9933
  model: opus
  updated_at: 2026-03-12T12:15:00Z
```

### JSON output

Same structure as YAML, rendered as a JSON array.

### Markdown output (`md`)

Renders a markdown table matching the [task status board](../../task-status-board/README.md) format, suitable for embedding in READMEs.

### CSV output

Flat comma-separated values with a header row. Useful for external tooling and spreadsheets.

### Selective fields

```
synchestra task list --project synchestra --status queued --fields path,title,depends_on --format csv
```

```csv
path,title,depends_on
write-tests,Write tests,implement-api
deploy-staging,Deploy staging,"implement-api,write-tests"
```

## Outstanding Questions

- Should there be a `--depth` flag to limit how deep in the task hierarchy to list?
- Should there be `--assigned-to` filtering by run ID?
