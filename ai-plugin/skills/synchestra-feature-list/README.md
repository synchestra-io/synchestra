---
name: synchestra-feature-list
description: Lists all features in a project. Use when listing features, exploring feature structure, or checking what features exist.
---

# Skill: synchestra-feature-list

List all features in a project to get an overview of what capabilities exist and how they're organized.

**CLI reference:** [synchestra feature list](../../spec/features/cli/feature/list/README.md)

## When to use

- **Surveying a project:** Get a complete list of all features in the project
- **Finding a feature ID:** Look up the exact ID before using `deps` or `refs`
- **Checking feature coverage:** See what areas are already defined vs. missing

## Command

```bash
synchestra feature list \
  [--project <project_id>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Parse the output — one feature ID per line |
| `2` | Invalid arguments | Check parameter values |
| `3` | Project not found | Verify the project identifier or current directory |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### List all features

```bash
synchestra feature list --project synchestra
# agent-skills
# claim-and-push
# cli
# cli/feature
# cli/task
# conflict-resolution
# cross-repo-sync
# micro-tasks
# model-selection
# outstanding-questions
# task-status-board
```

### List features with autodetected project

```bash
cd ~/projects/synchestra
synchestra feature list
```

### Pipe to other tools

```bash
# Count features
synchestra feature list --project synchestra | wc -l

# Find features matching a pattern
synchestra feature list --project synchestra | grep cli
```

## Notes

- This is a **read-only** command — it never mutates state.
- Output is plain text, one feature ID per line, sorted alphabetically. Easy to parse with standard Unix tools.
- Both parent features (`cli`) and their children (`cli/task`) appear as separate entries.
- Use `feature tree` for a hierarchical view, or `feature deps`/`feature refs` to trace relationships.
