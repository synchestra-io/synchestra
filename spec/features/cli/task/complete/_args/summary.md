# --summary

Brief description of what was accomplished when completing a task.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | — |

## Supported by

[`task complete`](../README.md)

## Description

A short record of what was accomplished during the task. This is stored in the task metadata and helps other agents and humans understand what was done without reading the full diff or task history.

While optional, providing a summary is recommended — it creates a useful audit trail.

## Examples

```bash
synchestra task complete --project synchestra --task implement-cli/parse-arguments \
  --summary "Implemented argument parser with validation and help text generation"

synchestra task complete --project my-service --task fix-auth-bug \
  --summary "Fixed empty token bypass in auth middleware; added regression test"
```

## Outstanding Questions

None at this time.
