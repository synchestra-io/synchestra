# --project

Identifies which Synchestra project the command should operate on.

| Detail | Value |
|---|---|
| Type | String |
| Required | Yes |
| Default | — |

## Supported by

All `synchestra` commands.

## Description

Every Synchestra command requires a project identifier to know which project repo to read from or write to. The value matches the project's `project_id` as defined in `synchestra-project.yaml` or the project directory name under `synchestra/projects/` in a control repo.

## Examples

```bash
synchestra task list --project synchestra
synchestra task claim --project my-service --task fix-bug --run 42 --model sonnet
```

## Outstanding Questions

None at this time.
