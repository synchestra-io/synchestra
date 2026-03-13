# Command: `synchestra server restart`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server restart [--path <dir>]
```

## Description

Restarts the Synchestra daemon by stopping then starting it. If the daemon is not running, starts it. If `stop` succeeds but `start` fails, the error from `start` is reported.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Daemon restarted successfully |
| `2` | Invalid configuration (start failed) |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Execute `synchestra server stop --path <dir>` logic
2. Execute `synchestra server start --path <dir>` logic
3. Report the exit code from `start` (stop is idempotent)

## Outstanding Questions

- Should restart support graceful connection draining before stopping?
