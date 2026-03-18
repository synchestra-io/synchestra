# CLI Arguments

**Parent:** [CLI](../README.md)

Arguments available to all `synchestra` commands.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--project`](project.md) | String | No (autodetected) | Project identifier |
| [`--path`](path.md) | String | No | Working directory for context resolution |
| [`--format`](format.md) | String | No | Output format for read commands |

### `--project`

Identifies which Synchestra project to operate on. Optional when the CLI is running inside a project directory or subdirectory — the project is autodetected from `synchestra-spec-repo.yaml`. Not required by `serve` and other project-independent commands. See [project.md](project.md) for full details.

### `--path`

Overrides the current working directory for commands that resolve a Synchestra context (serve, server, mcp). The CLI traverses up from this path looking for `synchestra-server.yaml`, `synchestra-spec-repo.yaml`, or `synchestra-state-repo.yaml`. See [path.md](path.md).

### `--format`

Controls the output format of read commands. Supported formats vary by command. Promoted from task-group level since it is now shared by `task` and `server` subcommands. See [format.md](format.md).

## Outstanding Questions

None at this time.
