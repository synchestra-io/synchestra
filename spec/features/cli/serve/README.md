# Command: `synchestra serve`

**Parent:** [CLI](../README.md)

## Synopsis

```
synchestra serve --http [host:port] --https [host:port] --mcp [/prefix | protocol://host:port] \
  [--tls-cert <path>] [--tls-key <path>] [--path <dir>]
```

## Description

Starts a foreground Synchestra server for interactive development. Similar to `vite dev` or `ng serve` — logs stream to stdout/stderr, Ctrl+C triggers a clean shutdown.

At least one protocol flag (`--http`, `--https`, or `--mcp`) is required. Each protocol flag accepts an optional address; if omitted, a sensible default is used. Multiple protocols can be active simultaneously.

The command resolves a working directory by traversing up from CWD (or `--path`) to find a marker file. If `synchestra-server.yaml` is found, its configuration is loaded as a base and CLI arguments override individual settings — this supports multi-project mode. If a spec or state repo marker is found, the server operates in single-project mode, resolving the counterpart repo via cross-reference.

For background daemon operation, see [`synchestra server`](../server/README.md).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--http`](_args/http.md) | At least one protocol | Start HTTP listener |
| [`--https`](_args/https.md) | At least one protocol | Start HTTPS listener |
| [`--mcp`](_args/mcp.md) | At least one protocol | Start MCP endpoint |
| [`--tls-cert`](_args/tls-cert.md) | With `--https` | Path to TLS certificate |
| [`--tls-key`](_args/tls-key.md) | With `--https` | Path to TLS private key |
| [`--path`](../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Clean shutdown |
| `2` | Invalid arguments (e.g., `--https` without `--tls-cert`) |
| `3` | Not a synchestra directory |
| `10+` | Unexpected error |

Exit codes `1` (Conflict) and `4` (Invalid state transition) do not apply — `serve` is a long-running process, not a state mutation command.

## Behaviour

1. Resolve working directory: use `--path` or CWD
2. Traverse up to find `synchestra-server.yaml`, `synchestra-spec.yaml`, or `synchestra-state.yaml`
3. If none found: exit with code `3`
4. If `synchestra-server.yaml` found: load config as base, CLI args override
5. If spec/state repo found: resolve counterpart repo via cross-reference, operate as single-project server
6. Validate protocol flags — at least one required; `--https` requires `--tls-cert` and `--tls-key`
7. Start listeners for each specified protocol
8. Run in foreground, stream logs to stdout/stderr
9. On SIGINT (Ctrl+C) or SIGTERM: clean shutdown (drain connections, exit 0)

## Outstanding Questions

- Should `serve` support hot-reload when spec/state files change (file watching)?
- Should there be a `--log-level` argument?
