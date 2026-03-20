# `--status`

Initial status for the new feature.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | No |
| **Default** | `Conceptual` |

## Supported by

- [`feature new`](../README.md)

## Description

Sets the `**Status:**` field in the generated feature README. The default is `Conceptual`, indicating the feature is in its earliest definition stage.

Accepted values are the feature lifecycle statuses used in the project. Common values include:

- `Conceptual` — idea stage, minimal definition
- `Not Started` — defined but no work begun
- `In Progress` — actively being developed
- `Implemented` — development complete

The command does not enforce a fixed set of status values — any string is accepted to allow project-specific conventions.

## Examples

```bash
# Default status: Conceptual
synchestra feature new --title "Micro Tasks"

# Explicit status
synchestra feature new --title "Micro Tasks" --status "Not Started"
```

## Outstanding Questions

- Should the command enforce a fixed set of valid status values, or accept any string?
