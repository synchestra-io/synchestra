# Command: `synchestra server start`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server start [--path <dir>]
```

## Description

Starts the Synchestra server as a background daemon. All settings (listeners, TLS, projects) are read from [`synchestra-server.yaml`](../synchestra-server.yaml.md) — no protocol or port arguments are accepted.

The daemon writes its PID to the configured `pid_file` and redirects output to the configured `log_file`. If a daemon is already running (PID file exists and process is alive), the command exits with code `1` (conflict).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Daemon started successfully |
| `1` | Conflict — daemon already running |
| `2` | Invalid configuration |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find and parse `synchestra-server.yaml`; exit `3` if not found, exit `2` if invalid
3. Check PID file — if process is alive, exit `1`
4. Fork daemon process
5. Write PID file
6. Start configured listeners
7. Parent process exits `0`

## Outstanding Questions

None at this time.
