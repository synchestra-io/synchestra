# Command: `synchestra server projects add`

**Parent:** [projects](../README.md)

## Synopsis

```
synchestra server projects add --spec <path> --state <path> [--path <dir>]
```

## Description

Adds a new project to `synchestra-server.yaml` by appending a spec/state repo pair to the `projects` list.

Cross-linked with the API endpoint [`POST /api/v1/projects/add`](../../../../../api/projects/README.md).

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
3. Validate that `--spec` points to a directory with `synchestra-project.yaml`
4. Validate that `--state` points to a directory with `synchestra-state.yaml`
5. Check if the project already exists in the config; exit `1` if so
6. Append the new project entry to `synchestra-server.yaml`
7. Exit `0`

## Outstanding Questions

- Should this command validate that the spec and state repos cross-reference each other?
