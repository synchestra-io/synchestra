# `--commit`

Create a git commit with the scaffolded files.

| Field | Value |
|---|---|
| **Type** | Boolean (flag) |
| **Required** | No |
| **Default** | `false` |

## Supported by

- [`feature new`](../README.md)

## Description

When passed, all scaffolded and updated files are staged and committed in a single git commit. The commit message follows the format: `feat(spec): add feature {feature_id}`.

By default (without `--commit` or `--push`), changes are made to the working tree only — no git operations are performed. This allows the user to review and edit the generated files before committing.

If `--push` is also passed, `--commit` is implied and does not need to be specified separately.

## Examples

```bash
# Scaffold and commit locally
synchestra feature new --title "Task Status Board" --commit

# Scaffold only (no commit)
synchestra feature new --title "Task Status Board"
```

## Outstanding Questions

None at this time.
