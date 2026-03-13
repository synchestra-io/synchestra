# --reason

Records why a status transition is happening.

| Detail | Value |
|---|---|
| Type | String |
| Required | For `fail` and `block`; optional otherwise |
| Default | — |

## Supported by

| Command | Required | Purpose |
|---|---|---|
| [`status`](../status/README.md) | No | Reason for the transition |
| [`fail`](../fail/README.md) | Yes | Why the task failed |
| [`block`](../block/README.md) | Yes | What is blocking the task |
| [`unblock`](../unblock/README.md) | No | What resolved the blocker |
| [`release`](../release/README.md) | No | Why the task is being released |
| [`abort`](../abort/README.md) | No | Why the abort is being requested |
| [`aborted`](../aborted/README.md) | No | What was done before aborting |

## Description

A human-readable explanation of why a status transition is occurring. When required (for `fail` and `block`), the reason must include enough detail for another agent or human to understand what happened and decide on next steps.

Even when optional, providing a reason is recommended — it creates an audit trail that helps with debugging and coordination.

## Examples

```bash
# Required: explaining a failure
synchestra task fail --project synchestra --task fix-bug \
  --reason "Build fails due to missing dependency: github.com/foo/bar v2 not published"

# Required: explaining a blocker
synchestra task block --project my-service --task migrate-db \
  --reason "Needs DBA approval for schema change before proceeding"

# Optional but helpful: explaining a release
synchestra task release --project synchestra --task refactor-logging \
  --reason "Higher-priority security fix appeared"
```

## Outstanding Questions

None at this time.
