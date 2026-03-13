# Command: `synchestra mcp`

**Parent:** [CLI](../README.md)

## Synopsis

```
synchestra mcp [--path <dir>]
```

## Description

Starts a stdio-based MCP (Model Context Protocol) server for AI agent tools. Designed to be launched as a subprocess by tools like Claude Code, Cursor, or other MCP-compatible clients.

Exposes the same API capabilities as the HTTP MCP endpoint (started via [`synchestra serve --mcp`](../serve/README.md)), but over the stdio transport (stdin/stdout) instead of HTTP.

Operates in single-project mode, determined by the resolved directory.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Clean shutdown |
| `2` | Invalid arguments |
| `3` | Not a synchestra directory |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find `synchestra-state.yaml`, `synchestra-project.yaml`, or `synchestra-server.yaml`; exit `3` if not found
3. If spec/state repo: resolve counterpart, operate as single-project
4. If server dir: operate as single-project using the first project in config
5. Start MCP server on stdio (stdin for requests, stdout for responses)
6. On EOF or SIGTERM: clean shutdown, exit `0`

## Usage Example

MCP client configuration (e.g., Claude Code `settings.json`):

```json
{
  "mcpServers": {
    "synchestra": {
      "command": "synchestra",
      "args": ["mcp", "--path", "/path/to/project"]
    }
  }
}
```

## Outstanding Questions

- Should `synchestra mcp` support multi-project if run in a server dir, or always single-project?
- What MCP tools and resources should be exposed? (Presumably mirrors CLI commands as MCP tools.)
