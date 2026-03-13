# --http

Starts an HTTP listener on the Synchestra server.

| Detail | Value |
|---|---|
| Type | String (optional value: `host:port`) |
| Required | At least one of `--http`, `--https`, or `--mcp` is required |
| Default | `localhost:8080` when flag is present without a value |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Enables the HTTP listener. Can be used with or without an explicit address:

- **Without value** (`--http`): listens on `localhost:8080`
- **With value** (`--http myhost:9090`): listens on the specified `host:port`

Can be combined with `--https` and/or `--mcp` to run multiple protocols simultaneously.

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `http`.

## Examples

```bash
# Default HTTP on localhost:8080
synchestra serve --http

# Custom address
synchestra serve --http 0.0.0.0:3000

# HTTP + MCP on same server
synchestra serve --http --mcp
```

## Outstanding Questions

None at this time.
