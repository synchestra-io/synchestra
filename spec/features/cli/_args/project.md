# --project

Identifies which Synchestra project the command should operate on.

| Detail | Value |
|---|---|
| Type | String |
| Required | No (autodetected when inside a project directory) |
| Default | Autodetected from current working directory |

## Supported by

Most `synchestra` commands. Not required by `serve` and other commands that operate independently of a specific project.

## Description

Tells the CLI which project repo to read from or write to. The value matches the project's `project_id` as defined in `synchestra-project.yaml` or the project directory name under `synchestra/projects/` in a control repo.

### Autodetection

When `--project` is omitted, the CLI walks up from the current working directory looking for a `synchestra-project.yaml` file. If found, the project is inferred automatically. This means agents and humans running commands inside a project directory (or any subdirectory) can skip the flag entirely.

If the CLI cannot autodetect a project and `--project` is not provided, the command exits with code `2` (invalid arguments).

## Examples

```bash
# Explicit project
synchestra task list --project synchestra
synchestra task claim --project my-service --task fix-bug --run 42 --model sonnet

# Autodetected (running from within the project directory)
cd ~/projects/synchestra
synchestra task list
synchestra task claim --task fix-bug --run 42 --model sonnet
```

## Outstanding Questions

None at this time.
