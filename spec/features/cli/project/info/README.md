# Command: `synchestra project info`

**Parent:** [project](../README.md)

## Synopsis

```
synchestra project info [--project <id>]
```

## Description

Displays the contents of the spec repo's `synchestra-spec.yaml` for the specified or autodetected project. This is a read-only command — it pulls the latest spec repo state before displaying.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No (autodetected) | Project identifier |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Project not found — no `synchestra-spec.yaml` at resolved location |
| `10+` | Unexpected error |

## Behaviour

1. Resolve project via `--project` or autodetection from CWD
2. Pull latest state from the spec repo
3. Read and output `synchestra-spec.yaml` contents
4. Exit `0`

## Outstanding Questions

- Should `info` also show derived information (e.g., resolved local paths for each repo, clone status)?
