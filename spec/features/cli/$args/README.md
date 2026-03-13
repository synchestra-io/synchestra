# CLI Arguments

**Parent:** [CLI](../README.md)

Arguments available to all `synchestra` commands.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--project`](project.md) | String | No (autodetected) | Project identifier |

### `--project`

Identifies which Synchestra project to operate on. Optional when the CLI is running inside a project directory or subdirectory — the project is autodetected from `synchestra-project.yaml`. Not required by `serve` and other project-independent commands. See [project.md](project.md) for full details.

## Outstanding Questions

None at this time.
