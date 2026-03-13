# --mcp

Starts an MCP (Model Context Protocol) endpoint on the Synchestra server.

| Detail | Value |
|---|---|
| Type | String (optional value: path prefix or `protocol://host:port`) |
| Required | At least one of `--http`, `--https`, or `--mcp` is required |
| Default | Piggyback on HTTP/HTTPS at `/mcp/` when flag is present without a value |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Enables the MCP endpoint for AI agent access. Three modes depending on the value:

- **Without value** (`--mcp`): adds MCP endpoints to the HTTP/HTTPS listener under the `/mcp/` prefix. Requires `--http` or `--https` to also be specified.
- **Path prefix** (`--mcp /custom-prefix/`): same as above but uses a custom URL prefix instead of `/mcp/`.
- **Full address** (`--mcp sse://host:port` or `--mcp http://host:port`): starts a separate MCP listener on the specified address.

For stdio-based MCP (used by AI agent tools like Claude Code), see [`synchestra mcp`](../../mcp/README.md).

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `mcp`.

## Examples

```bash
# MCP piggybacks on HTTP at /mcp/
synchestra serve --http --mcp

# Custom prefix
synchestra serve --http --mcp /ai/

# Separate MCP listener
synchestra serve --http --mcp sse://localhost:3001

# MCP without HTTP (separate listener required)
synchestra serve --mcp sse://localhost:3001
```

## Outstanding Questions

- Should `--mcp` without a value and without `--http`/`--https` be an error, or should it implicitly start an HTTP listener?
