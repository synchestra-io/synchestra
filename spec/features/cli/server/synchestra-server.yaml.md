# synchestra-server.yaml

Configuration file for the Synchestra background server daemon ([`synchestra server`](README.md)). Also used as a base configuration when [`synchestra serve`](../serve/README.md) is run in a server directory (CLI arguments override).

## Location

Must be in the root of the server directory. The CLI finds it by traversing up from CWD or the `--path` value.

## Schema

```yaml
# Listeners
# Each field corresponds to a `serve` CLI flag.
# Omit a field to disable that protocol.

http: "localhost:8080"              # --http (host:port)
https: "localhost:8443"             # --https (host:port)
mcp: "/mcp/"                       # --mcp (path prefix or protocol://host:port)

# TLS — required when https is set
tls:                                # --tls-cert, --tls-key
  cert: "/path/to/cert.pem"
  key: "/path/to/key.pem"

# Projects — list of spec/state repo pairs
projects:
  - spec: "/path/to/spec-repo"
    state: "/path/to/state-repo"

# Daemon settings
pid_file: "./synchestra.pid"        # default: relative to config dir
log_file: "./synchestra.log"        # default: relative to config dir
```

## Fields

| Field | Type | Required | Default | CLI equivalent |
|---|---|---|---|---|
| `http` | String (`host:port`) | No | — | [`--http`](../serve/_args/http.md) |
| `https` | String (`host:port`) | No | — | [`--https`](../serve/_args/https.md) |
| `mcp` | String (prefix or address) | No | — | [`--mcp`](../serve/_args/mcp.md) |
| `tls.cert` | String (file path) | With `https` | — | [`--tls-cert`](../serve/_args/tls-cert.md) |
| `tls.key` | String (file path) | With `https` | — | [`--tls-key`](../serve/_args/tls-key.md) |
| `projects` | Array of `{spec, state}` | Yes | — | — |
| `projects[].spec` | String (directory path) | Yes | — | [`--spec`](projects/add/_args/spec.md) |
| `projects[].state` | String (directory path) | Yes | — | [`--state`](projects/add/_args/state.md) |
| `pid_file` | String (file path) | No | `./synchestra.pid` | — |
| `log_file` | String (file path) | No | `./synchestra.log` | — |

### Validation

- At least one listener (`http`, `https`, or `mcp` with a full address) must be configured.
- If `https` is set, `tls.cert` and `tls.key` are required.
- `projects` must contain at least one entry.
- `spec` and `state` paths must point to valid directories containing the expected marker files.
- Relative paths are resolved relative to the directory containing `synchestra-server.yaml`.

## Outstanding Questions

- Should the config support environment variable interpolation (e.g., `${TLS_CERT_PATH}`)?
- Should there be an `auth` section for configuring API authentication?
- Should log rotation settings be configurable?
