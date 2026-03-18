# Command: `synchestra project code add`

**Parent:** [code](../README.md)

## Synopsis

```
synchestra project code add [--project <id>] --code-repo <ref> [--code-repo <ref>...]
```

## Description

Adds one or more code repositories to the project. Each code repo is appended to the `repos` list in the spec repo's `synchestra-spec-repo.yaml`. If a repo is not already on disk, it is cloned. A `synchestra-code-repo.yaml` file is written to each newly added repo.

If a code repo is already in the project's `repos` list, it is skipped (not an error).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../../_args/project.md) | No (autodetected) | Project identifier |
| [`--code-repo`](../../_args/code-repo.md) | Yes (at least one) | Code repository reference (repeatable) |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Code repo(s) added successfully |
| `1` | Conflict — code repo already has a `synchestra-code-repo.yaml` for a different project |
| `2` | Invalid arguments |
| `3` | Project or repo not found — clone failed or project not resolved |
| `10+` | Unexpected error |

## Behaviour

1. Resolve project via `--project` or autodetection from CWD
2. Pull latest state from the spec repo
3. For each `--code-repo`:
   a. Resolve reference to `{repos_dir}/{hosting}/{org}/{repo}`
   b. Clone if not on disk; exit `3` on clone failure
   c. Validate it is a git repo
   d. If `synchestra-code-repo.yaml` exists and points to a different project, exit `1`
   e. Skip if already in the `repos` list
   f. Write `synchestra-code-repo.yaml` with `spec_repos` pointing to the spec repo
   g. Append origin URL to `repos` in `synchestra-spec-repo.yaml`
4. Commit and push changes to spec repo and each new code repo
5. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

None at this time.
