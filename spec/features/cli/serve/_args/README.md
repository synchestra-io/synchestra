# Serve Arguments

**Parent:** [serve](../README.md)

Arguments specific to the `synchestra serve` command.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--http`](http.md) | String (optional value) | At least one protocol | Start HTTP listener |
| [`--https`](https.md) | String (optional value) | At least one protocol | Start HTTPS listener |
| [`--mcp`](mcp.md) | String (optional value) | At least one protocol | Start MCP endpoint |
| [`--tls-cert`](tls-cert.md) | String (file path) | With `--https` | TLS certificate file |
| [`--tls-key`](tls-key.md) | String (file path) | With `--https` | TLS private key file |

### `--http`

Starts an HTTP listener. Without a value, listens on `localhost:8080`. With a value, listens on the given `host:port`. See [http.md](http.md).

### `--https`

Starts an HTTPS listener. Without a value, listens on `localhost:8443`. Requires `--tls-cert` and `--tls-key`. See [https.md](https.md).

### `--mcp`

Starts an MCP endpoint. Without a value, piggybacks on the HTTP/HTTPS listener at `/mcp/`. With a path value (`/custom/`), uses that prefix. With a full address (`sse://host:port`), starts a separate MCP listener. See [mcp.md](mcp.md).

### `--tls-cert`

Path to the TLS certificate file. Required when `--https` is used. See [tls-cert.md](tls-cert.md).

### `--tls-key`

Path to the TLS private key file. Required when `--https` is used. See [tls-key.md](tls-key.md).

## Outstanding Questions

None at this time.
