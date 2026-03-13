# --status

Filters the task list to show only tasks with a specific status.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | — (all statuses) |

## Supported by

[`task list`](../README.md)

## Description

When provided, only tasks matching the given status are included in the output. When omitted, tasks in all statuses are listed.

Valid values: `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`

## Examples

```bash
# Find available work
synchestra task list --project synchestra --status queued

# Check what's in progress
synchestra task list --project synchestra --status in_progress

# Review blocked tasks
synchestra task list --project synchestra --status blocked
```

## Outstanding Questions

None at this time.
