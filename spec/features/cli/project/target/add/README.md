# Command: `synchestra project target add`

**Parent:** [target](../README.md)

## Synopsis

```
synchestra project target add [--project <id>] --target-repo <ref> [--target-repo <ref>...]
```

## Description

Adds one or more target (code) repositories to the project. Each target repo is appended to the `repos` list in the spec repo's `synchestra-spec.yaml`. If a repo is not already on disk, it is cloned. A `synchestra-target.yaml` file is written to each newly added repo.

If a target repo is already in the project's `repos` list, it is skipped (not an error).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../../_args/project.md) | No (autodetected) | Project identifier |
| [`--target-repo`](../../_args/target-repo.md) | Yes (at least one) | Target repository reference (repeatable) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Target repo(s) added successfully |
| `1` | Conflict — target repo already has a `synchestra-target.yaml` for a different project |
| `2` | Invalid arguments |
| `3` | Project or repo not found — clone failed or project not resolved |
| `10+` | Unexpected error |

## Behaviour

1. Resolve project via `--project` or autodetection from CWD
2. Pull latest state from the spec repo
3. For each `--target-repo`:
   a. Resolve reference to `{repos_dir}/{hosting}/{org}/{repo}`
   b. Clone if not on disk; exit `3` on clone failure
   c. Validate it is a git repo
   d. If `synchestra-target.yaml` exists and points to a different project, exit `1`
   e. Skip if already in the `repos` list
   f. Write `synchestra-target.yaml` with `spec_repo` pointing to the spec repo
   g. Append origin URL to `repos` in `synchestra-spec.yaml`
4. Commit and push changes to spec repo and each new target repo
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
