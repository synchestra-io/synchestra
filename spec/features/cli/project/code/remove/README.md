# Command: `synchestra project code remove`

**Parent:** [code](../README.md)

## Synopsis

```
synchestra project code remove [--project <id>] --code-repo <ref> [--code-repo <ref>...]
```

## Description

Removes one or more code repositories from the project's `repos` list in `synchestra-spec.yaml`.

This command does **not** delete `synchestra-code.yaml` from the code repos — it only removes them from the project's configuration.

If a code repo is not in the project's `repos` list, it is skipped (not an error).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../../_args/project.md) | No (autodetected) | Project identifier |
| [`--code-repo`](../../_args/code-repo.md) | Yes (at least one) | Code repository reference (repeatable) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Code repo(s) removed successfully |
| `2` | Invalid arguments |
| `3` | Project not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve project via `--project` or autodetection from CWD
2. Pull latest state from the spec repo
3. For each `--code-repo`:
   a. Resolve reference to origin URL
   b. Remove matching entry from `repos` list in `synchestra-spec.yaml`
   c. Skip silently if not found in the list
4. Commit and push changes to spec repo
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

- Should the command warn if removing the last code repo (leaving the project with none)?
