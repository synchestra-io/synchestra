# Command: `synchestra server stop`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server stop [--path <dir>]
```

## Description

Stops a running Synchestra daemon by reading the PID file and sending SIGTERM. Idempotent — stopping an already-stopped server (no PID file or stale PID) exits with code `0`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Daemon stopped (or was not running) |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find `synchestra-server.yaml`; exit `3` if not found
3. Read PID file — if missing or process not alive, exit `0`
4. Send SIGTERM to the process
5. Wait for process to exit (with timeout)
6. Remove PID file
7. Exit `0`

## Outstanding Questions

- What is the timeout for waiting for the process to exit before sending SIGKILL?
