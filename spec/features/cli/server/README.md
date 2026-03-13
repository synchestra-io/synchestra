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
