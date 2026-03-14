# Command: `synchestra project set`

**Parent:** [project](../README.md)

## Synopsis

```
synchestra project set [--project <id>] [--spec-repo <ref>] [--state-repo <ref>] [--key=value...]
```

## Description

Updates project configuration. At least one valid argument is required; the command exits with code `2` if called with no arguments.

For `--state-repo`: if the new state repo has no `synchestra-state.yaml`, one is created pointing to the current spec repo. If it has a `synchestra-state.yaml` pointing to a different spec repo, the command fails with exit code `1`.

For `--spec-repo`: updates the `spec_repo` reference in `synchestra-state.yaml` and all `synchestra-target.yaml` files to point to the new spec repo.

Additional key-value settings (e.g., `--allow-proposals=true`) are written to `synchestra-spec.yaml` as configuration fields.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No (autodetected) | Project identifier |
| [`--spec-repo`](../_args/spec-repo.md) | No | New spec repository reference |
| [`--state-repo`](../_args/state-repo.md) | No | New state repository reference |
| `--key=value` | No | Additional config settings |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Project updated successfully |
| `1` | Conflict — state repo's `synchestra-state.yaml` points to a different spec repo |
| `2` | Invalid arguments (no arguments provided, or invalid format) |
| `3` | Project or repo not found |
| `10+` | Unexpected error |

## Behaviour

1. Resolve project via `--project` or autodetection from CWD
2. Pull latest state from the spec repo
3. Validate at least one setting is being changed; exit `2` if not
4. If `--state-repo` provided:
   a. Resolve the new state repo reference; clone if not on disk
   b. If `synchestra-state.yaml` exists and `spec_repo` points elsewhere, exit `1`
   c. If `synchestra-state.yaml` does not exist, create it with `spec_repo` pointing to the current spec repo
   d. Update `state_repo` in `synchestra-spec.yaml`
5. If `--spec-repo` provided:
   a. Resolve the new spec repo reference; clone if not on disk
   b. Move `synchestra-spec.yaml` to the new spec repo (or update in place)
   c. Update `spec_repo` in `synchestra-state.yaml` and all `synchestra-target.yaml` files
6. Apply any additional `--key=value` settings to `synchestra-spec.yaml`
7. Commit and push changes to all affected repos
8. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

- Should `set` support unsetting / removing config keys?
- Should changing `--spec-repo` remove `synchestra-spec.yaml` from the old spec repo?
