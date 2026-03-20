# --enqueue

Creates the task in `queued` status instead of the default `planning`.

| Detail | Value |
|---|---|
| Type | Flag (boolean) |
| Required | No |
| Default | `false` |

## Supported by

[`task new`](../README.md)

## Description

By default, `task new` sets the initial status to `planning`, signalling that the task still needs refinement before agents can pick it up. Passing `--enqueue` skips the planning phase and creates the task directly in `queued` status, making it immediately available for agents to discover and claim.

Use this when the task is already fully defined and ready for work.

## Examples

```bash
# Create and immediately queue
synchestra task new --project synchestra --task fix-auth-bug \
  --title "Fix authentication bypass bug" \
  --description "Users can bypass auth by sending an empty token header" \
  --enqueue
```

## Outstanding Questions

None at this time.
