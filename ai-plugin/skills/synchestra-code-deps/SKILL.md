---
name: synchestra-code-deps
description: Lists Synchestra resources (features, plans, docs) that source files depend on. Use when tracing code-to-spec relationships, auditing coverage, or understanding what specifications a file or set of files implements.
---

# Skill: synchestra-code-deps

Show what Synchestra resources source files depend on — scan code comments for `synchestra:` references and expanded `https://synchestra.io/` URLs.

**CLI reference:** [synchestra code deps](../../spec/features/cli/code/deps/README.md)

## When to use

- **Tracing code to specs:** Find out which features, plans, or docs a source file references
- **Auditing coverage:** Scan a directory or glob pattern to see which specs are referenced across the codebase
- **Test traceability:** Use `--path` with a test file pattern to see what specs test files cover
- **Filtering by type:** Use `--type=feature` to focus on feature references only
- **Understanding a file:** Before modifying a file, check what specs it implements

## Command

```bash
synchestra code deps \
  [--path <pattern>] \
  [--project <project_id>] \
  [--type <type>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--path` | No | Glob pattern to select files (e.g., `pkg/**/*.go`, `src/*/*_test.go`). Defaults to `**/*` (all files recursively) |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| `--type` | No | Filter to a resource type: `feature`, `plan`, or `doc` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success (including no references found — empty output) | Parse the output — results grouped by file |
| `2` | Invalid arguments (e.g., unknown `--type`, invalid glob) | Check parameter values |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Single file

```bash
synchestra code deps --path=pkg/cli/task/claim.go
# spec/features/cli/task/claim
# spec/features/state-sync/pull
# spec/plans/v2-migration
```

### Glob pattern — all Go files in a package

```bash
synchestra code deps --path="pkg/cli/task/*.go"
# pkg/cli/task/claim.go
#   spec/features/cli/task/claim
#   spec/features/state-sync/pull
#   spec/plans/v2-migration
#
# pkg/cli/task/update.go
#   spec/features/cli/task/update
#   spec/features/state-sync/pull
```

### Test files only

```bash
synchestra code deps --path="src/*/*_test.go"
# src/auth/login_test.go
#   spec/features/auth/login
#
# src/task/claim_test.go
#   spec/features/cli/task/claim
```

### Filter to features only

```bash
synchestra code deps --type=feature
# pkg/cli/task/claim.go
#   spec/features/cli/task/claim
#   spec/features/state-sync/pull
#
# pkg/cli/task/update.go
#   spec/features/cli/task/update
#   spec/features/state-sync/pull
```

### Before modifying a file

```bash
# 1. What specs does this file depend on?
synchestra code deps --path=pkg/cli/task/claim.go
# spec/features/cli/task/claim
# spec/features/state-sync/pull

# 2. Read the relevant spec before making changes
synchestra feature info cli/task/claim
```

## Notes

- This is a **read-only** command — it never mutates state.
- Scans source file comments for `synchestra:` short notation and `https://synchestra.io/` expanded URLs.
- Only references preceded by a recognized comment prefix (`//`, `#`, `--`, `/*`, `%`, `;`) are detected.
- When one file matches the glob: flat list, no file header. When multiple match: results grouped by file.
- Empty output means no Synchestra references were found in matched files.
- For the reverse — finding what code references a spec — use [`feature refs`](../synchestra-feature-refs/README.md).
- For spec-to-spec dependencies — use [`feature deps`](../synchestra-feature-deps/README.md).
