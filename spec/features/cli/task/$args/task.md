# --task

Identifies the task to operate on within a project.

| Detail | Value |
|---|---|
| Type | String |
| Required | Yes |
| Default | — |

## Supported by

All `synchestra task` subcommands except [`list`](../list/README.md).

## Description

A `/`-separated path that identifies a task within a project's task hierarchy. Top-level tasks are a single segment (e.g., `implement-cli`); subtasks use nested paths (e.g., `implement-cli/parse-arguments`).

The path corresponds to the task's directory within the project repo. Each segment must be a valid directory name — lowercase, kebab-case is conventional.

## Examples

```bash
# Top-level task
synchestra task claim --project synchestra --task implement-cli --run 42 --model sonnet

# Nested subtask
synchestra task start --project synchestra --task implement-cli/parse-arguments

# Deeply nested
synchestra task info --project synchestra --task implement-cli/parse-arguments/validate-flags
```

## Outstanding Questions

None at this time.
