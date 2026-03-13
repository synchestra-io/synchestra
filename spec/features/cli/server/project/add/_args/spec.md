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
| [`server project add`](../README.md) |

## Description

Specifies the path to a spec repository containing `synchestra-project.yaml`. The path is stored in `synchestra-server.yaml` and used by the server to locate project specifications.

Relative paths are resolved relative to the directory containing `synchestra-server.yaml`.

The corresponding field in [`synchestra-server.yaml`](../../../synchestra-server.yaml.md) is `projects[].spec`.

## Examples

```bash
synchestra server project add --spec /home/user/projects/my-project --state /home/user/state/my-project

# Relative path
synchestra server project add --spec ../specs/my-project --state ../state/my-project
```

## Outstanding Questions

None at this time.
