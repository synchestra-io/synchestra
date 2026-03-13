# --path

Specifies the working directory for commands that resolve a Synchestra context.

| Detail | Value |
|---|---|
| Type | String (directory path) |
| Required | No |
| Default | Current working directory |

## Supported by

| Command | Description |
|---|---|
| [`serve`](../serve/README.md) | Foreground server |
| [`server start`](../server/start/README.md) | Background daemon start |
| [`server stop`](../server/stop/README.md) | Background daemon stop |
| [`server restart`](../server/restart/README.md) | Background daemon restart |
| [`server status`](../server/status/README.md) | Background daemon status |
| [`server projects`](../server/projects/README.md) | List projects from config |
| [`server projects add`](../server/projects/add/README.md) | Add project to config |
| [`mcp`](../mcp/README.md) | stdio MCP server |

## Description

Overrides the current working directory for directory resolution. The CLI traverses up from this path (or CWD if omitted) looking for one of the following marker files:

- `synchestra-server.yaml` — server directory (multi-project)
- `synchestra-project.yaml` — spec repo
- `synchestra-state.yaml` — state repo

If no marker file is found before reaching the filesystem root, the command exits with code `3`.

## Examples

```bash
# Serve from a specific directory
synchestra serve --http --path /home/user/projects/my-project

# Check server status in a different directory
synchestra server status --path /opt/synchestra

# Start MCP for a specific project
synchestra mcp --path ~/projects/my-project
```

## Outstanding Questions

None at this time.
