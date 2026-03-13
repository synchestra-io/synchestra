# Command: `synchestra server project list`

**Parent:** [project](../README.md)

## Synopsis

```
synchestra server project list [--path <dir>] [--format <format>]
```

## Description

Lists projects configured in `synchestra-server.yaml`. Does not require the server to be running — reads directly from the config file.

Cross-linked with the API endpoint [`GET /api/v1/project/list`](../../../../../api/project/README.md).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../../_args/path.md) | No | Working directory override |
| [`--format`](../../../_args/format.md) | No | Output format: `yaml` (default), `json`, `md`, `csv` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Projects listed |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find and parse `synchestra-server.yaml`; exit `3` if not found
3. Output the list of configured projects

## Outstanding Questions

None at this time.
