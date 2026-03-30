# Command: `synchestra feature new`

**Parent:** [feature](../README.md)

## Synopsis

```
synchestra feature new --title <title> [--slug <slug>] [--parent <parent_id>] [--status <status>] [--description <description>] [--depends-on <deps>] [--project <project_id>] [--format <format>] [--commit] [--push]
```

## Description

Scaffolds a new feature directory with a README template containing all required sections. By default, changes are made locally without committing. Use `--commit` to create a git commit, or `--push` to commit and push atomically.

The feature slug (directory name) is auto-generated from the title by lowercasing, replacing spaces with hyphens, and removing non-URL-safe characters. Use `--slug` to override the generated slug.

For sub-features, use `--parent` to nest the new feature under an existing parent, or include slashes in `--slug` (e.g., `--slug cli/task/claim`). When `--parent` is used, the generated or provided slug is appended to the parent path. The parent feature must already exist; if it does not, the command fails with exit code `3`.

When creating a sub-feature, the parent's `## Contents` section is updated to include the new child. When creating a top-level feature, the feature index (`spec/features/README.md`) is updated with a new row.

The generated README includes all required sections per the [feature spec](https://github.com/synchestra-io/specscore/blob/main/spec/features/feature/README.md): title with status, Summary, Problem, Behavior, Acceptance Criteria, and Outstanding Questions. If `--description` is provided, it is placed in the Summary section. If `--depends-on` is provided, a Dependencies section is included.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--title`](_args/title.md) | Yes | Human-readable feature title |
| [`--slug`](_args/slug.md) | No | Feature slug (directory name). Auto-generated from title if omitted |
| [`--parent`](_args/parent.md) | No | Parent feature ID for creating a sub-feature |
| [`--status`](_args/status.md) | No | Initial feature status. Default: `Conceptual` |
| [`--description`](_args/description.md) | No | Short description placed in the Summary section |
| [`--depends-on`](_args/depends-on.md) | No | Comma-separated list of feature IDs this feature depends on |
| [`--project`](../../_args/project.md) | No | Project identifier. Autodetected from current directory if omitted |
| [`--format`](../../_args/format.md) | No | Output format: `yaml` (default), `json`, `text` |
| [`--commit`](_args/commit.md) | No | Create a git commit with the changes |
| [`--push`](_args/push.md) | No | Commit and push atomically. Implies `--commit` |

## Output

On success, `feature new` returns the same structured output as [`feature info`](../info/README.md) for the newly created feature. This gives the caller immediate access to section line ranges for surgical editing — no follow-up `feature info` call needed.

Default format is YAML (agent-first). Use `--format text` for human-readable output.

### Example

```bash
synchestra feature new --title "Task Status Board" \
  --description "A markdown table tracking task assignments and status." \
  --depends-on "state-store"
```

```yaml
path: task-status-board
status: "Conceptual"
deps:
  - state-store
refs: []
children: []
plans: []
sections:
  - title: Summary
    lines: 5-5
  - title: Problem
    lines: 7-7
  - title: Behavior
    lines: 9-9
  - title: Dependencies
    lines: 11-13
    items: 1
  - title: Acceptance Criteria
    lines: 15-15
  - title: Outstanding Questions
    lines: 17-17
    items: 0
```

Agents can then use the `lines` ranges to target specific sections for content population (e.g., write the Problem statement at line 7, expand Behavior starting at line 9).

## Exit Codes

| Exit code | Meaning |
|---|---|
| `0` | Feature created successfully |
| `1` | Conflict — remote state changed during push (only with `--push`) |
| `2` | Invalid arguments (missing title, invalid slug, invalid status) |
| `3` | Parent feature not found |
| `4` | Feature already exists at the target path |
| `10+` | Unexpected errors |

## Behaviour

1. Validate arguments: `--title` is required; slug (if provided) must be lowercase, hyphen-separated, and URL-safe per the [feature structural rules](https://github.com/synchestra-io/specscore/blob/main/spec/features/feature/README.md)
2. Generate slug from title if `--slug` is not provided:
   - Lowercase the title
   - Replace spaces and underscores with hyphens
   - Remove characters that are not alphanumeric or hyphens
   - Collapse consecutive hyphens into one
   - Trim leading and trailing hyphens
3. Resolve the full feature path:
   - If `--parent` is provided: `{features_dir}/{parent_path}/{slug}`
   - If `--slug` contains slashes: treat as a full nested path `{features_dir}/{slug}`
   - Otherwise: `{features_dir}/{slug}` (top-level feature)
4. If creating a sub-feature: validate that the parent feature directory exists and contains a `README.md`. Exit `3` if not found.
5. Verify that no feature already exists at the target path. Exit `4` if the directory exists.
6. Create the feature directory and `README.md` from the template (see [Generated README Template](#generated-readme-template))
7. If creating a sub-feature: update the parent's `## Contents` section with a new index row and a brief summary line for the child
8. If creating a top-level feature: update the feature index (`spec/features/README.md`) with a new row in the index table and a brief summary in the Feature Summaries section
9. If `--commit` or `--push`: stage all changed files and create a single git commit with message `feat(spec): add feature {feature_id}`
10. If `--push`: push to remote. On conflict: pull, re-validate preconditions (parent still exists, target path still free), retry or exit `1`

## Generated README Template

```markdown
# Feature: {title}

**Status:** {status}

## Summary

{description or "TODO: Brief summary of the feature."}

## Problem

TODO: What problem does this feature solve?

## Behavior

TODO: How does this feature work?

## Dependencies

- {dep1}
- {dep2}

## Acceptance Criteria

TODO: Define acceptance criteria.

## Outstanding Questions

None at this time.
```

The `## Dependencies` section is only included when `--depends-on` is provided. The `## Contents` section is not generated initially — it is added automatically when the first sub-feature is created under this feature.

## Slug Generation

The slug algorithm converts a human-readable title into a valid feature directory name:

| Input title | Generated slug |
|---|---|
| `Task Status Board` | `task-status-board` |
| `CLI` | `cli` |
| `Cross-Repo Sync` | `cross-repo-sync` |
| `Outstanding Questions (OQ)` | `outstanding-questions-oq` |
| `  Extra   Spaces  ` | `extra-spaces` |

The `--slug` flag overrides this entirely, but the provided slug is still validated against the same rules (lowercase, hyphen-separated, URL-safe, no underscores or special characters).

## Outstanding Questions

- Should `--dry-run` be supported to preview the scaffolded files without writing them?
- Should the command validate that feature IDs in `--depends-on` actually exist, or accept any string?
- Should the generated template include optional sections like `## Interaction with Other Features` or `## Configuration` based on additional flags?
- When updating the parent's `## Contents` section: if the section does not exist yet, should the command create it? (Likely yes, to support the first sub-feature case.)
- Should the commit message format be configurable, or is `feat(spec): add feature {feature_id}` sufficient?
