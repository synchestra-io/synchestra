# Command: `synchestra code deps`

**Parent:** [code](../README.md)

## Synopsis

```
synchestra code deps [<path>...] [--project <project_id>] [--type <type>]
```

## Description

Shows the Synchestra resources (features, plans, docs) that source files depend on. Scans source files for [source references](../../../source-references/README.md) — `synchestra:` annotations and expanded `https://synchestra.io/` URLs in comments — and lists the referenced resources.

This is the code → specification counterpart to [`synchestra feature deps`](../../feature/deps/README.md), which shows spec → spec dependencies. Together they provide full traceability from code to specifications and between specifications.

This is a read-only command. It scans the working tree and does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<path>...` | No | File(s) or directory(s) to scan. Defaults to the current working directory. When a directory is given, all files are scanned recursively |
| [`--project`](../../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| `--type` | No | Filter results to a specific resource type: `feature`, `plan`, or `doc`. Without this flag, all types are shown |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success (including when no references are found — outputs nothing) |
| `2` | Invalid arguments (e.g., unknown `--type` value) |
| `3` | Path not found |

## Behaviour

1. Resolve the target path(s) — default to current working directory if none given
2. Scan all files under the target path(s) for source references using the [comment-prefix detection rule](../../../source-references/README.md#detection-strategy)
3. Parse each detected reference to extract `{type}` and `{path}` (and `{org}/{repo}` for cross-repo references)
4. If `--type` is specified, filter to matching resource type
5. Deduplicate and sort results alphabetically by type, then by path
6. Output each referenced resource, one per line

If no source references are found, the command outputs nothing and exits with code `0`.

### Grouping by file

When scanning multiple files, results are grouped by source file with the file path as a header. When scanning a single file, the file path header is omitted.

### Cross-repo references

Cross-repo references (`@{host}/{org}/{repo}`) are included in the output with the `@{host}/{org}/{repo}` suffix. They are not validated against the remote repository by default (validation would require network access).

## Output

### Single file

```bash
synchestra code deps pkg/cli/task/claim.go
```

```
spec/features/cli/task/claim
spec/features/state-sync/pull
spec/plans/v2-migration
```

### Directory scan

```bash
synchestra code deps pkg/cli/task/
```

```
pkg/cli/task/claim.go
  spec/features/cli/task/claim
  spec/features/state-sync/pull
  spec/plans/v2-migration

pkg/cli/task/update.go
  spec/features/cli/task/update
  spec/features/state-sync/pull
```

### Filtered by type

```bash
synchestra code deps --type=feature
```

```
pkg/cli/task/claim.go
  spec/features/cli/task/claim
  spec/features/state-sync/pull

pkg/cli/task/update.go
  spec/features/cli/task/update
  spec/features/state-sync/pull
```

### Cross-repo references

```bash
synchestra code deps pkg/integration/orchestrator.go
```

```
spec/features/agent-skills@github.com/acme/orchestrator
spec/features/cli/task/claim
```

### No references found

```bash
synchestra code deps pkg/util/strings.go
# (no output — exit code 0)
```

## Outstanding Questions

None at this time.
