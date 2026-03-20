# `--depends-on`

Comma-separated list of feature IDs this feature depends on.

| Field | Value |
|---|---|
| **Type** | String (comma-separated) |
| **Required** | No |
| **Default** | — |

## Supported by

- [`feature new`](../README.md)

## Description

When provided, a `## Dependencies` section is included in the generated feature README with a bullet list of the specified feature IDs.

When omitted, no `## Dependencies` section is generated (features without dependencies simply don't have the section).

Feature IDs are paths relative to the features directory, using `/` as separators (e.g., `cli/task`, `task-status-board`).

## Examples

```bash
synchestra feature new --title "Claim" --parent "cli/task" \
  --depends-on "task-status-board,state-store"
```

Generates:

```markdown
## Dependencies

- task-status-board
- state-store
```

## Outstanding Questions

- Should the command validate that the listed feature IDs actually exist?
