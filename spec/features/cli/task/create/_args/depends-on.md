# --depends-on

Declares task dependencies at creation time.

| Detail | Value |
|---|---|
| Type | String (comma-separated) |
| Required | No |
| Default | — |

## Supported by

[`task create`](../README.md)

## Description

A comma-separated list of task paths that this task depends on. Dependencies are recorded in the task metadata and can be used by the orchestrator to determine execution order.

Each dependency path must refer to an existing task within the same project.

## Examples

```bash
# Single dependency
synchestra task create --project my-service --task run-migrations \
  --title "Run database migrations" \
  --depends-on setup-db

# Multiple dependencies
synchestra task create --project my-service --task deploy-staging \
  --title "Deploy to staging" \
  --depends-on setup-db,create-schema
```

## Outstanding Questions

None at this time.
