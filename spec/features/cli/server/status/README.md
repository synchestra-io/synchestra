# Command: `synchestra server status`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server status [--path <dir>] [--format <format>]
```

## Description

Reports whether the Synchestra daemon is running, its PID, uptime, and listening addresses. Supports [`--format`](../../_args/format.md) for structured output.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |
| [`--format`](../../_args/format.md) | No | Output format: `text` (default), `json`, `yaml` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Status reported (regardless of whether daemon is running) |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find `synchestra-server.yaml`; exit `3` if not found
3. Read PID file
4. Check if process is alive
5. Output status: running/stopped, PID, uptime, configured listeners, project count

## Outstanding Questions

None at this time.
