# --current

Guards against stale state by asserting the expected current status.

| Detail | Value |
|---|---|
| Type | String |
| Required | For update mode |
| Default | — |

## Supported by

[`task status`](../README.md) (update mode)

## Description

Specifies the status the task is expected to be in before the transition. If the actual status does not match `--current`, the command exits with code `1` (status mismatch) instead of applying the change.

This is a concurrency guard — it prevents agents from blindly overwriting a status that another agent changed since the last read. Always provide `--current` when updating status.

Valid values: `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`

## Examples

```bash
# Transition from claimed to in_progress
synchestra task status --project synchestra --task impl-cli \
  --current claimed --new in_progress

# Fails if status doesn't match
synchestra task status --project synchestra --task impl-cli \
  --current in_progress --new completed
# Exit 1: "Status mismatch: expected in_progress, actual is aborted"
```

## Outstanding Questions

None at this time.
