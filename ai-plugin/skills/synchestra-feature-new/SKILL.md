---
name: synchestra-feature-new
description: Scaffolds a new feature directory with a README template. Use when creating new features, adding sub-features, or expanding the feature hierarchy.
---

# Skill: synchestra-feature-new

Scaffold a new feature directory with a README containing all required sections (Summary, Problem, Behavior, Acceptance Criteria, Outstanding Questions). The parent's Contents table and the feature index are updated automatically. Changes are local by default; use `--commit` or `--push` for git operations. Returns `feature info`-compatible output with section line ranges.

**CLI reference:** [synchestra feature new](../../spec/features/cli/feature/new/README.md)

## When to use

- You need to create a new feature in the spec hierarchy
- You are adding a sub-feature under an existing feature
- You want to scaffold a feature with the correct template and update all indexes atomically
- You need immediate section line ranges for follow-up editing (no separate `feature info` call needed)

## Command

```bash
synchestra feature new \
  --title <title> \
  [--slug <slug>] \
  [--parent <parent_id>] \
  [--status <status>] \
  [--description <description>] \
  [--depends-on <deps>] \
  [--project <project_id>] \
  [--format <format>] \
  [--commit] \
  [--push]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--title` | Yes | Human-readable feature title (e.g., `"Task Status Board"`) |
| `--slug` | No | Feature slug (directory name). Auto-generated from title if omitted. Must be lowercase, hyphen-separated |
| `--parent` | No | Parent feature ID for creating a sub-feature (e.g., `cli/task`). Cannot be combined with slashes in `--slug` |
| `--status` | No | Initial feature status: `draft` (default), `approved`, `implemented` |
| `--description` | No | Short description placed in the Summary section. Also appears in parent Contents and feature index |
| `--depends-on` | No | Comma-separated list of feature IDs this feature depends on (e.g., `state-store,cli`). Each must exist |
| `--project` | No | Project identifier. Autodetected from current directory if omitted |
| `--format` | No | Output format: `yaml` (default), `json`, `text` |
| `--commit` | No | Create a git commit with the changes |
| `--push` | No | Commit and push atomically. Implies `--commit` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Feature created successfully | Parse the output for section line ranges, then edit specific sections |
| `1` | Conflict — remote state changed during push | Re-pull and retry the creation |
| `2` | Invalid arguments (missing title, invalid slug, invalid status, nonexistent dependency) | Check parameter values and retry |
| `3` | Parent feature not found | Verify the parent feature ID exists before creating a sub-feature |
| `4` | Feature already exists at the target path | Use a different slug, or check the existing feature |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Create a top-level feature

```bash
synchestra feature new \
  --title "Task Status Board" \
  --description "A markdown table tracking task assignments and status." \
  --depends-on "state-store"
```

```yaml
path: task-status-board
status: draft
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

### Create a sub-feature under an existing parent

```bash
synchestra feature new \
  --title "Claim Protocol" \
  --parent cli/task \
  --description "Defines how agents claim tasks for exclusive work."
```

### Create a feature with explicit slug

```bash
synchestra feature new \
  --title "Outstanding Questions (OQ)" \
  --slug outstanding-questions \
  --status approved
```

### Create and commit in one step

```bash
synchestra feature new \
  --title "Sandbox Environment" \
  --description "Isolated execution environment for agent tasks." \
  --commit
```

### Create a nested feature using slash-slug (alternative to --parent)

```bash
synchestra feature new \
  --title "Container Image" \
  --slug sandbox/container-image \
  --description "Docker container image for sandbox environments."
```

## Notes

- **Slug auto-generation**: The title is lowercased, spaces/underscores become hyphens, non-alphanumeric characters are removed, consecutive hyphens collapse. Example: `"Outstanding Questions (OQ)"` → `outstanding-questions-oq`.
- **Mutual exclusion**: `--parent` and slashes in `--slug` cannot be used together — use one or the other for nesting.
- **Dependency validation**: All feature IDs in `--depends-on` must exist as directories under `spec/features/`. The command exits with code `2` if any ID is not found.
- **Status validation**: Only `draft`, `approved`, and `implemented` are accepted (case-insensitive). Default is `draft`.
- **Parent auto-update**: When creating a sub-feature, the parent's `## Contents` table is updated (or created if it doesn't exist yet).
- **Index auto-update**: When creating a top-level feature, the feature index (`spec/features/README.md`) is updated with a new row.
- **Output is `feature info`-compatible**: Use the `sections[].lines` ranges to immediately target specific sections for content population — no follow-up `feature info` call needed.
- **Local by default**: Without `--commit` or `--push`, changes are only made on disk. Use `--commit` to stage and commit, or `--push` for atomic commit-and-push.
- **Commit message**: `feat(spec): add feature {feature_id}` (not configurable).

## Outstanding Questions

None at this time.
