# Skill: synchestra-feature-tree

Display the feature hierarchy as an indented tree to understand project structure at a glance.

**CLI reference:** [synchestra feature tree](../../spec/features/cli/feature/tree/README.md)

## When to use

- **Understanding structure:** See how features are organized hierarchically
- **Navigating a project:** Quickly identify parent-child relationships between features
- **Reporting:** Generate a human-readable overview of project capabilities

## Command

```bash
synchestra feature tree \
  [--project <project_id>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Read the indented tree output |
| `2` | Invalid arguments | Check parameter values |
| `3` | Project not found | Verify the project identifier or current directory |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Show feature tree

```bash
synchestra feature tree --project synchestra
# agent-skills
# claim-and-push
# cli
# 	feature
# 	task
# conflict-resolution
# cross-repo-sync
# micro-tasks
# model-selection
# outstanding-questions
# task-status-board
```

### Tree with autodetected project

```bash
cd ~/projects/synchestra
synchestra feature tree
```

## Notes

- This is a **read-only** command — it never mutates state.
- Indentation uses tab characters — one tab per nesting level.
- Nested features show only the leaf name (e.g., `task` under `cli`, not `cli/task`). The full path is implied by the tree context.
- For a flat list with full feature IDs, use `feature list` instead.
- Use `feature deps`/`feature refs` to trace dependency relationships that aren't visible in the hierarchy.
