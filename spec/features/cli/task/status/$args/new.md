# --new

Specifies the target status for a transition.

| Detail | Value |
|---|---|
| Type | String |
| Required | For update mode |
| Default | — |

## Supported by

[`task status`](../README.md) (update mode)

## Description

The status to transition the task to. The transition must be valid according to the [status transition diagram](../../../README.md#task-statuses) — invalid transitions exit with code `4`.

Valid values: `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`

## Examples

```bash
# Mark as completed
synchestra task status --project synchestra --task fix-bug \
  --current in_progress --new completed

# Report failure with reason
synchestra task status --project synchestra --task fix-bug \
  --current in_progress --new failed \
  --reason "Dependency not available"
```

## Outstanding Questions

None at this time.
