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
