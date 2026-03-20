# --title

Human-readable title for a new task.

| Detail | Value |
|---|---|
| Type | String |
| Required | Yes |
| Default | — |

## Supported by

[`task new`](../README.md)

## Description

A short, descriptive title that identifies the task. Displayed in task boards and listings. Should be clear enough for both agents and humans to understand the task's purpose at a glance.

## Examples

```bash
synchestra task new --project synchestra --task implement-cli \
  --title "Implement CLI framework"

synchestra task new --project my-service --task fix-auth-bug \
  --title "Fix authentication bypass bug"
```

## Outstanding Questions

None at this time.
