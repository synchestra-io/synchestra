# Command: `synchestra task list`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-list](../../../../../skills/synchestra-task-list/README.md)

## Synopsis

```
synchestra task list --project <project_id> [--status <status>] [--format <format>]
```

## Description

Lists tasks in a project. By default, all tasks are shown in table format. Use `--status` to filter by task status and `--format` to control the output format.

This is a read-only command. It pulls the latest state from the project repo but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| `--status` | No | Filter by task status (e.g., `pending`, `in_progress`, `claimed`, `completed`, `failed`, `blocked`, `aborted`) |
| `--format` | No | Output format: `table` (default), `json`, `yaml` |

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

### Table output

```
PATH                              STATUS        RUN    MODEL    UPDATED_AT
implement-cli                     in_progress   4821   sonnet   2026-03-12T10:45:00Z
implement-cli/parse-arguments     completed     4821   sonnet   2026-03-12T11:02:00Z
implement-cli/validate-config     pending        —      —       2026-03-12T09:00:00Z
fix-auth-bug                      claimed       9933   opus     2026-03-12T12:15:00Z
write-tests                       blocked       5501   haiku    2026-03-12T11:30:00Z
```

### JSON / YAML output

Structured output includes the same fields: `path`, `status`, `run`, `model`, and `updated_at`. Null values are used when a field is not set (e.g., `run` and `model` for a `pending` task).

## Outstanding Questions

- Should there be a `--depth` flag to limit how deep in the task hierarchy to list?
- Should there be `--assigned-to` filtering by run ID?
