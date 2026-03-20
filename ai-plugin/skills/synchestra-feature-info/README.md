---
name: synchestra-feature-info
description: Shows metadata, section table-of-contents, and children for a single feature. Use when you need to understand a feature's structure before reading its full spec.
---

# Skill: synchestra-feature-info

Get a compact overview of a feature — metadata, section TOC with line numbers, and children consistency check — without reading the full README.

**CLI reference:** [synchestra feature info](../../spec/features/cli/feature/info/README.md)

## When to use

- **Before reading a spec:** Decide which sections to read based on the section TOC
- **Quick metadata check:** See status, parent, outstanding questions count, dependencies count
- **Children audit:** The `in_readme` field shows if a child directory is listed in the README's Contents table
- **Triage:** Quickly assess a feature's state without loading the full document

## Command

```bash
synchestra feature info <feature_id> \
  [--project <project_id>] \
  [--format text|yaml|json]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional, e.g., `cli/feature`, `task-status-board`) |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--format`](../../spec/features/cli/_args/format.md) | No | Output format: `yaml` (default), `json`, `text` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Parse the YAML/JSON output |
| `2` | Invalid arguments | Check parameter values |
| `3` | Feature or project not found | Verify the feature ID and project |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Get feature info

```bash
synchestra feature info cli/feature
```

```yaml
path: cli/feature
title: "Feature Management Commands"
status: "In Progress"
parent: cli
children:
  - path: cli/feature/info
    in_readme: true
  - path: cli/feature/list
    in_readme: true
  - path: cli/feature/tree
    in_readme: true
  - path: cli/feature/deps
    in_readme: true
  - path: cli/feature/refs
    in_readme: true
file: spec/features/cli/feature/README.md
lines: 89
sections:
  - title: "Synopsis"
    lines: [7, 12]
  - title: "Commands"
    lines: [14, 32]
    items: 5
  - title: "Shared Arguments"
    lines: [34, 48]
    items: 2
  - title: "Outstanding Questions"
    lines: [85, 89]
    items: 1
```

### Decide what to read

```bash
# 1. Get overview
synchestra feature info task-status-board

# 2. Based on section TOC, read only the relevant section
# e.g., lines 45-78 cover "Claiming Protocol"
```

## Notes

- This is a **read-only** command — it never mutates state.
- Default output is YAML (agent-first). Use `--format text` for human-readable output.
- The `sections` list is a table of contents with line ranges — use it to read specific sections instead of the whole file.
- `items` appears only on sections containing countable lists (Outstanding Questions, Dependencies, Acceptance Criteria).
- `children.in_readme: false` means a child directory exists on disk but isn't listed in the README's Contents table — a consistency issue.
- ~500 tokens total — much cheaper than reading the full README.
