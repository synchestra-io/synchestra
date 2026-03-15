# Serve, Server & MCP Commands

## Summary

Three independent CLI commands for running the Synchestra API server in different modes:

- **`synchestra serve`** — foreground dev server, CLI-arg-driven
- **`synchestra server`** — background daemon, config-driven (`synchestra-server.yaml`)
- **`synchestra mcp`** — stdio MCP server for AI agent tools

All three resolve a working directory by traversing up from CWD (or `--path`) to find `synchestra-state.yaml`, `synchestra-spec.yaml`, or `synchestra-server.yaml`. Error (exit code 3) if none found.

## Command: `synchestra serve`

**Purpose:** Interactive foreground server for development. Similar to `vite dev` or `ng serve`.

### Synopsis

```
synchestra serve --http [host:port] --https [host:port] --mcp [/prefix | protocol://host:port] \
  [--tls-cert <path>] [--tls-key <path>] [--path <dir>]
```

### Protocol Flags

At least one of `--http`, `--https`, or `--mcp` is required.

| Flag | Without value | With value |
|---|---|---|
| `--http` | `localhost:8080` | Listen on given `host:port` |
| `--https` | `localhost:8443` | Listen on given `host:port` |
| `--mcp` | Piggyback on HTTP/HTTPS at `/mcp/` | `/custom-prefix/` for custom path, or `sse://host:port` / `http://host:port` for separate listener |

### TLS Arguments

Required when `--https` is used.

| Flag | Description |
|---|---|
| `--tls-cert` | Path to TLS certificate file |
| `--tls-key` | Path to TLS private key file |

### Behavior

1. Resolve working directory: use `--path` or CWD
2. Traverse up to find `synchestra-state.yaml`, `synchestra-spec.yaml`, or `synchestra-server.yaml`
3. If none found: exit code 3 ("Not a synchestra directory")
4. If `synchestra-server.yaml` found: load config as base, CLI args override — supports multi-project
5. If spec/state repo found: behave as single-project server, resolve counterpart repo via cross-reference
6. Start listeners for each specified protocol
7. Run in foreground, stream logs to stdout/stderr, Ctrl+C for clean shutdown

### Exit Codes

| Exit code | Meaning |
|---|---|
| `0` | Clean shutdown |
| `2` | Invalid arguments (e.g., `--https` without `--tls-cert`) |
| `3` | Not a synchestra directory |
| `10+` | Unexpected error |

Note: exit codes `1` (Conflict) and `4` (Invalid state transition) do not apply — `serve` is a long-running process, not a state mutation command.

## Command Group: `synchestra server`

**Purpose:** Background daemon management. Similar to `nginx` or `systemctl`. All configuration comes from `synchestra-server.yaml` — no protocol/port/TLS CLI args. Only `--path` is accepted to locate the config directory.

Note: `server` deviates from the `<resource> <action>` pattern used by commands like `task claim`. The lifecycle subcommands (`start`, `stop`, `restart`, `status`) manage a daemon process, not a domain resource. The `projects` sub-group nests a resource under this namespace, creating a three-level command (`synchestra server projects add`), which is justified by the tight coupling between project management and server configuration.

The `start`, `stop`, `restart`, and `status` subcommands have no API equivalents — they are local-only operations. The daemon itself *is* the API server, so managing it over HTTP would be circular.

### Subcommands

| Command | Description |
|---|---|
| `start [--path <dir>]` | Start daemon, write PID file, logs to file |
| `stop [--path <dir>]` | Stop daemon by PID, clean up PID file (idempotent) |
| `restart [--path <dir>]` | Stop then start |
| `status [--path <dir>]` | Report if running, PID, uptime, listening addresses |
| `projects [--path <dir>]` | List projects from config |
| `projects add --spec <path> --state <path> [--path <dir>]` | Add a project to config |

Read subcommands (`status`, `projects`) support `--format` (yaml/json/md/csv) for structured output, consistent with other read commands like `task list`.

### `synchestra-server.yaml` Config

Fields are cross-linked with `serve` CLI args.

```yaml
# Listeners — mirrors serve's --http, --https, --mcp flags
http: "localhost:8080"          # omit to disable
https: "localhost:8443"         # omit to disable
mcp: "/mcp/"                   # path prefix, or "sse://host:port" for separate listener

# TLS — mirrors serve's --tls-cert, --tls-key
tls:
  cert: "/path/to/cert.pem"
  key: "/path/to/key.pem"

# Projects
projects:
  - spec: "/path/to/spec-repo"
    state: "/path/to/state-repo"
  - spec: "/path/to/another-spec"
    state: "/path/to/another-state"

# Daemon
pid_file: "./synchestra.pid"              # default: relative to config dir
log_file: "./synchestra.log"              # default: relative to config dir
```

### Exit Codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `1` | Conflict (e.g., `projects add` for an already-configured project) |
| `2` | Invalid config |
| `3` | Config not found |
| `10+` | Unexpected error |

## Command: `synchestra mcp`

**Purpose:** stdio MCP server for AI agent tools. Designed to be launched as a subprocess by tools like Claude Code, Cursor, etc.

### Synopsis

```
synchestra mcp [--path <dir>]
```

### Behavior

1. Resolve directory (same traversal logic as `serve`)
2. Start MCP server over stdio (stdin/stdout)
3. Expose the same API capabilities as the HTTP MCP endpoint
4. Single-project only (determined by directory)

### Usage Example

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

### Exit Codes

| Exit code | Meaning |
|---|---|
| `0` | Clean shutdown |
| `2` | Invalid arguments |
| `3` | Not a synchestra directory |
| `10+` | Unexpected error |

## API Endpoints

### `GET /api/v1/projects/list`

List projects served by the server. Cross-linked with `synchestra server projects`.

Note: Uses action-oriented path (`/projects/list`) for consistency with the task API pattern (`/task/list`). See implementation plan for rationale.

```yaml
/projects/list:
  get:
    operationId: projectList
    summary: List projects served by this server
    responses:
      200:
        description: List of projects
        content:
          application/json:
            schema:
              type: object
              properties:
                projects:
                  type: array
                  items:
                    type: object
                    properties:
                      id:
                        type: string
                      spec:
                        type: string
                        description: Path or URL to spec repo
                      state:
                        type: string
                        description: Path or URL to state repo
```

In single-project mode, returns a list with one item.

In multi-project mode, all other API endpoints (e.g., task operations) use the existing `?project=` query parameter to select which project to operate on, consistent with the `--project` CLI arg.

### `POST /api/v1/projects/add`

Add a project. Cross-linked with `synchestra server projects add`.

```yaml
/projects/add:
  post:
    operationId: projectAdd
    summary: Add a project to the server
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [spec, state]
            properties:
              spec:
                type: string
                description: Path or URL to spec repo
              state:
                type: string
                description: Path or URL to state repo
    responses:
      201:
        description: Project added
      409:
        description: Project already exists
```

## Directory Resolution

All three commands share the same directory resolution logic:

1. Start from `--path` value or CWD
2. Traverse up the directory tree
3. Look for (in priority order):
   - `synchestra-server.yaml` → server directory (multi-project)
   - `synchestra-spec.yaml` → spec repo (resolve state via cross-reference)
   - `synchestra-state.yaml` → state repo (resolve spec via cross-reference)
4. If root (`/`) reached without finding any marker → exit code 3

When a spec or state repo is found, the counterpart repo is resolved via the cross-reference defined in the marker file. The server operates as if it were a single-project server directory.

## Spec File Structure

```
spec/features/cli/
  serve/
    README.md
    _args/
      README.md
      http.md
      https.md
      mcp.md
      tls-cert.md
      tls-key.md
  server/
    README.md
    synchestra-server.yaml.md       # config format spec
    start/README.md
    stop/README.md
    restart/README.md
    status/README.md
    projects/
      README.md                     # default: list projects
      add/
        README.md
        _args/
          README.md
          spec.md
          state.md
  mcp/
    README.md

spec/features/cli/_args/
  path.md                           # global arg, used by serve, server, mcp

spec/api/
  projects/
    README.md
    openapi.yaml
```

## Key Design Decisions

1. **`serve` vs `server` is purely lifecycle** — foreground vs daemon. Both support single-project and multi-project.
2. **`server` subcommands take no protocol/port/TLS args** — purely config-driven from `synchestra-server.yaml`.
3. **`serve` in a server dir loads config as base, CLI args override** — allows quick dev overrides.
4. **`--mcp` without value piggybacks on HTTP/HTTPS** at `/mcp/` prefix — avoids extra port allocation for simple setups.
5. **`synchestra mcp` is stdio-only** — designed for AI agent subprocess integration, no host/port needed.
6. **`--path` is shared across `serve`, `server`, and `mcp`** — same traversal logic. Placed at the global `_args/` level since it may apply to future commands; the `Supported by` section in the arg doc limits its scope.
7. **`server projects` reads config directly** — does not require the server to be running.

## Outstanding Questions

- Should `synchestra serve` support hot-reload when spec/state files change (file watching)?
- Should `server projects add` validate that spec and state repos cross-reference each other before adding?
- What authentication/authorization model should the HTTP API use (bearer token, API key, mTLS)?
- Should `server projects remove` be specced now or deferred?
- What MCP tools/resources should be exposed? (Presumably mirrors the CLI commands as MCP tools.)
- Should `synchestra mcp` support multi-project if run in a server dir, or always single-project?
- How should a running server handle spec/state repo updates pushed by other agents? (Poll, file watch, or re-read on each request?)
- Should `server restart` drain active connections before stopping (graceful) or stop immediately (hard)?
- ~~API endpoint style~~ — **Resolved:** projects API uses action-oriented paths (`/projects/list`, `/projects/add`) for consistency with the task API. See implementation plan for rationale.
