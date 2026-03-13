# --fields

Selects specific fields to include in the output.

| Detail | Value |
|---|---|
| Type | String (comma-separated) |
| Required | No |
| Default | All fields |

## Supported by

[`task list`](../README.md)

## Description

A comma-separated list of field names to include in the output. When omitted, all available fields are shown. Use this to reduce output to only the data you need, especially when piping to other tools.

### Available fields

`path`, `status`, `title`, `run`, `model`, `requester`, `depends_on`, `branch`, `claimed_at`, `updated_at`, `abort_requested`

## Examples

```bash
# Only path and status
synchestra task list --project synchestra --fields path,status

# CSV with selected fields
synchestra task list --project synchestra --status queued \
  --fields path,title,depends_on --format csv
```

## Outstanding Questions

None at this time.
