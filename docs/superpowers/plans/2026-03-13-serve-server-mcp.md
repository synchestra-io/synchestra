# Serve, Server & MCP — Specification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the full CLI spec, API spec, and argument documentation for the `serve`, `server`, and `mcp` commands.

**Architecture:** Three independent command groups documented in `spec/features/cli/` with corresponding API specs in `spec/api/projects/`. All follow existing project conventions — every directory has a `README.md` with Outstanding Questions, arguments live in `_args/` directories, and parent READMEs index their children.

**Tech Stack:** Markdown, OpenAPI 3.1 YAML

**Design Spec:** [`docs/superpowers/specs/2026-03-13-serve-server-mcp-design.md`](../specs/2026-03-13-serve-server-mcp-design.md)

---

## Design Decisions Made During Planning

- **API endpoint paths use action-oriented style** (`/projects/list`, `/projects/add`) rather than RESTful (`GET /projects`, `POST /projects`). This matches the existing task API pattern (`/task/list`, `/task/create`) for internal consistency. The design spec raised this as an outstanding question; it is resolved here.
- **`--format` promoted to global `_args/`** — now used by both `task` and `server` groups, so it belongs at the global level per placement rules. The existing `task/_args/format.md` is moved up.

---

## Chunk 1: Global Argument and CLI Root Updates

### Task 1: Add `--path` global argument and promote `--format` to global

**Files:**
- Create: `spec/features/cli/_args/path.md`
- Move: `spec/features/cli/task/_args/format.md` → `spec/features/cli/_args/format.md`
- Modify: `spec/features/cli/_args/README.md`
- Modify: `spec/features/cli/task/_args/README.md` (update link to point to global format.md)

- [ ] **Step 1: Create `path.md`**

```markdown
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
- `synchestra-spec.yaml` — spec repo
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
```

- [ ] **Step 2: Move `format.md` to global `_args/` and update**

Move `spec/features/cli/task/_args/format.md` to `spec/features/cli/_args/format.md`. Update the "Supported by" table to include `server status` and `server projects`:

```bash
git mv spec/features/cli/task/_args/format.md spec/features/cli/_args/format.md
```

Then update `spec/features/cli/_args/format.md` — add to the "Supported by" table:

```markdown
| [`server status`](../server/status/README.md) | `text`, `json`, `yaml` | `text` |
| [`server projects`](../server/projects/README.md) | `yaml`, `json`, `md`, `csv` | `yaml` |
```

- [ ] **Step 3: Update `spec/features/cli/task/_args/README.md`**

Change the `--format` link from `[format.md](format.md)` to `[format.md](../../_args/format.md)` and update the table row to link to the new global location.

- [ ] **Step 4: Update `spec/features/cli/_args/README.md`**

Add `--path` and `--format` to the arguments table:

```markdown
| [`--path`](path.md) | String | No | Working directory for context resolution |
| [`--format`](format.md) | String | No | Output format for read commands |
```

Add summary sections after the existing `### --project` summary:

```markdown
### `--path`

Overrides the current working directory for commands that resolve a Synchestra context (serve, server, mcp). The CLI traverses up from this path looking for `synchestra-server.yaml`, `synchestra-spec.yaml`, or `synchestra-state.yaml`. See [path.md](path.md).

### `--format`

Controls the output format of read commands. Supported formats vary by command. Promoted from task-group level since it is now shared by `task` and `server` subcommands. See [format.md](format.md).
```

- [ ] **Step 5: Commit**

```bash
git add spec/features/cli/_args/ spec/features/cli/task/_args/
git commit -m "spec: add --path global arg, promote --format to global level"
```

### Task 2: Update CLI README with new command groups

**Files:**
- Modify: `spec/features/cli/README.md`

- [ ] **Step 1: Add serve, server, and mcp to the Command Groups table**

Add rows to the existing table at the bottom of the Command Groups section:

```markdown
| [serve](serve/README.md) | Foreground dev server — HTTP, HTTPS, MCP |
| [server](server/README.md) | Background daemon management |
| [mcp](mcp/README.md) | stdio MCP server for AI agents |
```

Add summary sections after the existing `### task` summary:

```markdown
### `serve`

Starts a foreground Synchestra server exposing API over HTTP, HTTPS, and/or MCP. Designed for interactive development — logs stream to stdout, Ctrl+C to stop. See [serve/README.md](serve/README.md).

### `server`

Background daemon management — start, stop, restart, status, and project configuration. All settings come from `synchestra-server.yaml`. See [server/README.md](server/README.md).

### `mcp`

Starts a stdio-based MCP server for AI agent tools (Claude Code, Cursor, etc.). Designed to be launched as a subprocess. See [mcp/README.md](mcp/README.md).
```

- [ ] **Step 2: Commit**

```bash
git add spec/features/cli/README.md
git commit -m "spec: add serve, server, mcp to CLI command index"
```

---

## Chunk 2: `synchestra serve` Command Spec

### Task 3: Create serve command spec and arguments

**Files:**
- Create: `spec/features/cli/serve/README.md`
- Create: `spec/features/cli/serve/_args/README.md`
- Create: `spec/features/cli/serve/_args/http.md`
- Create: `spec/features/cli/serve/_args/https.md`
- Create: `spec/features/cli/serve/_args/mcp.md`
- Create: `spec/features/cli/serve/_args/tls-cert.md`
- Create: `spec/features/cli/serve/_args/tls-key.md`

- [ ] **Step 1: Create `serve/README.md`**

```markdown
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
```

- [ ] **Step 2: Create `serve/_args/README.md`**

```markdown
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
```

- [ ] **Step 3: Create `serve/_args/http.md`**

```markdown
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
```

- [ ] **Step 4: Create `serve/_args/https.md`**

```markdown
# --https

Starts an HTTPS listener on the Synchestra server.

| Detail | Value |
|---|---|
| Type | String (optional value: `host:port`) |
| Required | At least one of `--http`, `--https`, or `--mcp` is required |
| Default | `localhost:8443` when flag is present without a value |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Enables the HTTPS listener. Requires `--tls-cert` and `--tls-key` to provide the TLS certificate and private key.

- **Without value** (`--https`): listens on `localhost:8443`
- **With value** (`--https myhost:8443`): listens on the specified `host:port`

Can be combined with `--http` and/or `--mcp`.

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `https`.

## Examples

```bash
# HTTPS with TLS certificates
synchestra serve --https --tls-cert cert.pem --tls-key key.pem

# Custom address + HTTP
synchestra serve --http --https 0.0.0.0:443 --tls-cert cert.pem --tls-key key.pem
```

## Outstanding Questions

None at this time.
```

- [ ] **Step 5: Create `serve/_args/mcp.md`**

```markdown
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
```

- [ ] **Step 6: Create `serve/_args/tls-cert.md`**

```markdown
# --tls-cert

Path to the TLS certificate file for HTTPS.

| Detail | Value |
|---|---|
| Type | String (file path) |
| Required | Yes, when `--https` is used |
| Default | — |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Specifies the path to a PEM-encoded TLS certificate file. Required when `--https` is used. If `--https` is specified without `--tls-cert`, the command exits with code `2` (invalid arguments).

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `tls.cert`.

## Examples

```bash
synchestra serve --https --tls-cert /etc/ssl/certs/synchestra.pem --tls-key /etc/ssl/private/synchestra.key
```

## Outstanding Questions

None at this time.
```

- [ ] **Step 7: Create `serve/_args/tls-key.md`**

```markdown
# --tls-key

Path to the TLS private key file for HTTPS.

| Detail | Value |
|---|---|
| Type | String (file path) |
| Required | Yes, when `--https` is used |
| Default | — |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Specifies the path to a PEM-encoded TLS private key file. Required when `--https` is used. If `--https` is specified without `--tls-key`, the command exits with code `2` (invalid arguments).

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `tls.key`.

## Examples

```bash
synchestra serve --https --tls-cert cert.pem --tls-key key.pem
```

## Outstanding Questions

None at this time.
```

- [ ] **Step 8: Commit**

```bash
git add spec/features/cli/serve/
git commit -m "spec: add synchestra serve command specification"
```

---

## Chunk 3: `synchestra server` Command Group Spec

### Task 4: Create server command group and config spec

**Files:**
- Create: `spec/features/cli/server/README.md`
- Create: `spec/features/cli/server/synchestra-server.yaml.md`

- [ ] **Step 1: Create `server/README.md`**

```markdown
# Command Group: `synchestra server`

**Parent:** [CLI](../README.md)

Background daemon management for the Synchestra server. All configuration comes from `synchestra-server.yaml` — no protocol, port, or TLS arguments are accepted on the command line. Only [`--path`](../_args/path.md) is accepted to locate the config directory.

This group deviates from the `<resource> <action>` pattern used by commands like `task claim`. The lifecycle subcommands (`start`, `stop`, `restart`, `status`) manage a daemon process, not a domain resource. The `projects` sub-group nests project management under this namespace, creating three-level commands (e.g., `synchestra server projects add`), which is justified by the tight coupling between project management and server configuration.

The `start`, `stop`, `restart`, and `status` subcommands have no API equivalents — they are local-only operations. The daemon itself *is* the API server, so managing it over HTTP would be circular.

For foreground (interactive) operation, see [`synchestra serve`](../serve/README.md).

## Configuration

Server configuration is defined in [`synchestra-server.yaml`](synchestra-server.yaml.md). This file must exist in the resolved directory (via `--path` or CWD traversal).

## Commands

| Command | Description |
|---|---|
| [start](start/README.md) | Start the daemon |
| [stop](stop/README.md) | Stop the daemon |
| [restart](restart/README.md) | Restart the daemon |
| [status](status/README.md) | Show daemon status |
| [projects](projects/README.md) | Project management |

### `start`

Starts the Synchestra server as a background daemon. Reads all settings from `synchestra-server.yaml`, writes a PID file, and redirects logs to the configured log file. See [start/README.md](start/README.md).

### `stop`

Stops a running daemon by PID. Idempotent — stopping an already-stopped server exits with code `0`. See [stop/README.md](stop/README.md).

### `restart`

Stops then starts the daemon. See [restart/README.md](restart/README.md).

### `status`

Reports whether the daemon is running, its PID, uptime, and listening addresses. Supports `--format` for structured output. See [status/README.md](status/README.md).

### `projects`

Lists and manages projects in the server config. Does not require the server to be running — reads directly from `synchestra-server.yaml`. See [projects/README.md](projects/README.md).

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `server/synchestra-server.yaml.md`**

```markdown
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
```

- [ ] **Step 3: Commit**

```bash
git add spec/features/cli/server/README.md spec/features/cli/server/synchestra-server.yaml.md
git commit -m "spec: add synchestra server command group and config spec"
```

### Task 5: Create server lifecycle subcommand specs

**Files:**
- Create: `spec/features/cli/server/start/README.md`
- Create: `spec/features/cli/server/stop/README.md`
- Create: `spec/features/cli/server/restart/README.md`
- Create: `spec/features/cli/server/status/README.md`

- [ ] **Step 1: Create `server/start/README.md`**

```markdown
# Command: `synchestra server start`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server start [--path <dir>]
```

## Description

Starts the Synchestra server as a background daemon. All settings (listeners, TLS, projects) are read from [`synchestra-server.yaml`](../synchestra-server.yaml.md) — no protocol or port arguments are accepted.

The daemon writes its PID to the configured `pid_file` and redirects output to the configured `log_file`. If a daemon is already running (PID file exists and process is alive), the command exits with code `1` (conflict).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Daemon started successfully |
| `1` | Conflict — daemon already running |
| `2` | Invalid configuration |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find and parse `synchestra-server.yaml`; exit `3` if not found, exit `2` if invalid
3. Check PID file — if process is alive, exit `1`
4. Fork daemon process
5. Write PID file
6. Start configured listeners
7. Parent process exits `0`

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `server/stop/README.md`**

```markdown
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
```

- [ ] **Step 3: Create `server/restart/README.md`**

```markdown
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
```

- [ ] **Step 4: Create `server/status/README.md`**

```markdown
# Command: `synchestra server status`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server status [--path <dir>] [--format <format>]
```

## Description

Reports whether the Synchestra daemon is running, its PID, uptime, and listening addresses. Supports `--format` for structured output.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |
| [`--format`](../../_args/format.md) | No | Output format: `text` (default), `json`, `yaml` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Status reported (regardless of whether daemon is running) |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find `synchestra-server.yaml`; exit `3` if not found
3. Read PID file
4. Check if process is alive
5. Output status: running/stopped, PID, uptime, configured listeners, project count

## Outstanding Questions

None at this time.
```

- [ ] **Step 5: Commit**

```bash
git add spec/features/cli/server/start/ spec/features/cli/server/stop/ spec/features/cli/server/restart/ spec/features/cli/server/status/
git commit -m "spec: add server lifecycle subcommands (start, stop, restart, status)"
```

### Task 6: Create server projects subcommand specs

**Files:**
- Create: `spec/features/cli/server/projects/README.md`
- Create: `spec/features/cli/server/projects/add/README.md`
- Create: `spec/features/cli/server/projects/add/_args/README.md`
- Create: `spec/features/cli/server/projects/add/_args/spec.md`
- Create: `spec/features/cli/server/projects/add/_args/state.md`

- [ ] **Step 1: Create `server/projects/README.md`**

```markdown
# Command: `synchestra server projects`

**Parent:** [server](../README.md)

## Synopsis

```
synchestra server projects [--path <dir>] [--format <format>]
```

## Description

Lists projects configured in `synchestra-server.yaml`. Does not require the server to be running — reads directly from the config file.

Cross-linked with the API endpoint [`GET /api/v1/projects`](../../../../api/projects/README.md).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--path`](../../_args/path.md) | No | Working directory override |
| [`--format`](../../_args/format.md) | No | Output format: `yaml` (default), `json`, `md`, `csv` |

## Commands

| Command | Description |
|---|---|
| [add](add/README.md) | Add a project to the config |

### `add`

Adds a new project (spec + state repo pair) to `synchestra-server.yaml`. See [add/README.md](add/README.md).

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Projects listed |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find and parse `synchestra-server.yaml`; exit `3` if not found
3. Output the list of configured projects

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `server/projects/add/README.md`**

```markdown
# Command: `synchestra server projects add`

**Parent:** [projects](../README.md)

## Synopsis

```
synchestra server projects add --spec <path> --state <path> [--path <dir>]
```

## Description

Adds a new project to `synchestra-server.yaml` by appending a spec/state repo pair to the `projects` list.

Cross-linked with the API endpoint [`POST /api/v1/projects`](../../../../../api/projects/README.md).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--spec`](_args/spec.md) | Yes | Path to the spec repo |
| [`--state`](_args/state.md) | Yes | Path to the state repo |
| [`--path`](../../../_args/path.md) | No | Working directory override |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Project added |
| `1` | Conflict — project already exists in config |
| `2` | Invalid arguments |
| `3` | `synchestra-server.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve directory via `--path` or CWD traversal
2. Find and parse `synchestra-server.yaml`; exit `3` if not found
3. Validate that `--spec` points to a directory with `synchestra-spec.yaml`
4. Validate that `--state` points to a directory with `synchestra-state.yaml`
5. Check if the project already exists in the config; exit `1` if so
6. Append the new project entry to `synchestra-server.yaml`
7. Exit `0`

## Outstanding Questions

- Should this command validate that the spec and state repos cross-reference each other?
```

- [ ] **Step 3: Create `server/projects/add/_args/README.md`**

```markdown
# Projects Add Arguments

**Parent:** [projects add](../README.md)

Arguments specific to the `synchestra server projects add` command.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--spec`](spec.md) | String (directory path) | Yes | Path to the spec repo |
| [`--state`](state.md) | String (directory path) | Yes | Path to the state repo |

### `--spec`

Path to the project's spec repository. See [spec.md](spec.md).

### `--state`

Path to the project's state repository. See [state.md](state.md).

## Outstanding Questions

None at this time.
```

- [ ] **Step 4: Create `server/projects/add/_args/spec.md`**

```markdown
# --spec

Path to the project's spec repository.

| Detail | Value |
|---|---|
| Type | String (directory path) |
| Required | Yes |
| Default | — |

## Supported by

| Command |
|---|
| [`server projects add`](../README.md) |

## Description

Specifies the path to a spec repository containing `synchestra-spec.yaml`. The path is stored in `synchestra-server.yaml` and used by the server to locate project specifications.

Relative paths are resolved relative to the directory containing `synchestra-server.yaml`.

The corresponding field in [`synchestra-server.yaml`](../../../synchestra-server.yaml.md) is `projects[].spec`.

## Examples

```bash
synchestra server projects add --spec /home/user/projects/my-project --state /home/user/state/my-project

# Relative path
synchestra server projects add --spec ../specs/my-project --state ../state/my-project
```

## Outstanding Questions

None at this time.
```

- [ ] **Step 5: Create `server/projects/add/_args/state.md`**

```markdown
# --state

Path to the project's state repository.

| Detail | Value |
|---|---|
| Type | String (directory path) |
| Required | Yes |
| Default | — |

## Supported by

| Command |
|---|
| [`server projects add`](../README.md) |

## Description

Specifies the path to a state repository containing `synchestra-state.yaml`. The path is stored in `synchestra-server.yaml` and used by the server to manage project runtime state (tasks, claims, etc.).

Relative paths are resolved relative to the directory containing `synchestra-server.yaml`.

The corresponding field in [`synchestra-server.yaml`](../../../synchestra-server.yaml.md) is `projects[].state`.

## Examples

```bash
synchestra server projects add --spec /home/user/projects/my-project --state /home/user/state/my-project
```

## Outstanding Questions

None at this time.
```

- [ ] **Step 6: Commit**

```bash
git add spec/features/cli/server/projects/
git commit -m "spec: add server projects subcommands (list, add)"
```

---

## Chunk 4: `synchestra mcp` Command Spec

### Task 7: Create mcp command spec

**Files:**
- Create: `spec/features/cli/mcp/README.md`

- [ ] **Step 1: Create `mcp/README.md`**

```markdown
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
2. Find `synchestra-state.yaml`, `synchestra-spec.yaml`, or `synchestra-server.yaml`; exit `3` if not found
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
```

- [ ] **Step 2: Commit**

```bash
git add spec/features/cli/mcp/
git commit -m "spec: add synchestra mcp command specification"
```

---

## Chunk 5: Projects API Spec

### Task 8: Create projects API spec

**Files:**
- Create: `spec/api/projects/README.md`
- Create: `spec/api/projects/openapi.yaml`
- Modify: `spec/api/README.md`
- Modify: `spec/features/api/README.md`

- [ ] **Step 1: Create `spec/api/projects/README.md`**

```markdown
# Projects API

REST API endpoints for server project management. Maps to [`synchestra server projects`](../../features/cli/server/projects/README.md) CLI commands.

## Specification

Full OpenAPI 3.1 specification: [`openapi.yaml`](openapi.yaml)

## Endpoints

All endpoints are under `/api/v1/projects/`.

### Read Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `GET` | `/projects/list` | [`synchestra server projects`](../../features/cli/server/projects/README.md) | List configured projects |

### Mutation Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `POST` | `/projects/add` | [`synchestra server projects add`](../../features/cli/server/projects/add/README.md) | Add a project to the server |

## Error Codes

| HTTP Status | CLI Exit Code | Error Code | When |
|---|---|---|---|
| `200 OK` | 0 | — | Success |
| `201 Created` | 0 | — | Project added |
| `400 Bad Request` | 2 | `invalid_arguments` | Missing or invalid parameters |
| `409 Conflict` | 1 | `conflict` | Project already exists |
| `500 Internal Server Error` | 10+ | `internal_error` | Unexpected failure |

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `spec/api/projects/openapi.yaml`**

```yaml
openapi: 3.1.0
info:
  title: Synchestra Projects API
  description: |
    REST API for server project management in Synchestra.

    Endpoints map to `synchestra server projects` CLI commands.
    See the [CLI server projects spec](../../features/cli/server/projects/README.md) for full behavioral details.

    ## Authentication
    All requests require a Bearer token in the `Authorization` header.
  version: 0.1.0
  license:
    name: MIT

servers:
  - url: /api/v1
    description: API v1

tags:
  - name: projects-read
    description: Read-only project operations
  - name: projects-management
    description: Project management operations

security:
  - bearerAuth: []

paths:
  /projects/list:
    get:
      operationId: projectList
      summary: List projects
      description: |
        List all projects configured on this server.

        **CLI equivalent:** `synchestra server projects [--format <format>]`
        — see [spec](../../features/cli/server/projects/README.md)
      tags: [projects-read]
      responses:
        '200':
          description: List of projects
          content:
            application/json:
              schema:
                type: object
                required: [items]
                properties:
                  items:
                    type: array
                    items:
                      $ref: '#/components/schemas/Project'
        '500':
          $ref: '#/components/responses/InternalError'

  /projects/add:
    post:
      operationId: projectAdd
      summary: Add a project
      description: |
        Add a new project to the server configuration.

        **CLI equivalent:** `synchestra server projects add --spec <path> --state <path>`
        — see [spec](../../features/cli/server/projects/add/README.md)
      tags: [projects-management]
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
                  description: Path or URL to the spec repository
                  example: /home/user/projects/my-project
                state:
                  type: string
                  description: Path or URL to the state repository
                  example: /home/user/state/my-project
      responses:
        '201':
          description: Project added
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        '400':
          $ref: '#/components/responses/BadRequest'
        '409':
          $ref: '#/components/responses/Conflict'
        '500':
          $ref: '#/components/responses/InternalError'

components:
  schemas:
    Project:
      type: object
      required: [id, spec, state]
      properties:
        id:
          type: string
          description: Project identifier (derived from spec repo)
          example: my-project
        spec:
          type: string
          description: Path or URL to the spec repository
          example: /home/user/projects/my-project
        state:
          type: string
          description: Path or URL to the state repository
          example: /home/user/state/my-project

    Error:
      type: object
      required: [error, code]
      properties:
        error:
          type: string
          description: Human-readable error message
        code:
          type: string
          description: Machine-readable error code
          enum: [invalid_arguments, conflict, internal_error]
        request_id:
          type: string
          description: Unique request identifier

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      description: Bearer token authentication

  responses:
    BadRequest:
      description: Invalid arguments
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error: "Missing required field: spec"
            code: invalid_arguments
            request_id: req_abc123

    Conflict:
      description: Project already exists
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error: "Project 'my-project' already exists"
            code: conflict
            request_id: req_abc123

    InternalError:
      description: Unexpected failure
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error: Internal server error
            code: internal_error
            request_id: req_abc123
```

- [ ] **Step 3: Update `spec/api/README.md`**

Add `projects` to the Contents table. Note: the existing "Future resources" table lists `project/` (singular) — replace that row with the plural form since the actual directory is `projects/`:

```markdown
| [projects/](projects/README.md) | Server project management | In Progress |
```

Add a summary section:

```markdown
### projects

Server project management — list and add projects to a running server or server configuration. Endpoints map to `synchestra server projects` CLI commands. See [`projects/README.md`](projects/README.md).
```

- [ ] **Step 4: Update `spec/features/api/README.md`**

Add `projects` to the Specification table:

```markdown
| [Projects](../../api/projects/README.md) | [`spec/api/projects/openapi.yaml`](../../api/projects/openapi.yaml) | In Progress |
```

- [ ] **Step 5: Commit**

```bash
git add spec/api/projects/ spec/api/README.md spec/features/api/README.md
git commit -m "spec: add projects API specification (OpenAPI 3.1)"
```
