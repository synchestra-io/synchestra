---
name: synchestra-feature-tree
description: Displays the feature hierarchy as an indented tree. Use when exploring project structure, navigating to a feature's ancestors/subtree, or generating an overview with metadata.
---

# Skill: synchestra-feature-tree

Display the feature hierarchy as an indented tree to understand project structure at a glance. Can focus on a specific feature showing ancestors, subtree, or both.

**CLI reference:** [synchestra feature tree](../../spec/features/cli/feature/tree/README.md)

## When to use

- **Understanding structure:** See how features are organized hierarchically
- **Navigating to a feature:** Focus on a feature to see its ancestors and/or subtree
- **Status overview:** Use `--fields=status` to see the state of each feature in the tree
- **Reporting:** Generate a human-readable overview of project capabilities

## Command

```bash
synchestra feature tree \
  [<feature_id>] \
  [--direction up|down] \
  [--project <project_id>] \
  [--fields <fields>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | No | Feature ID to focus on (e.g., `cli/task`). When omitted, shows the full project tree |
| `--direction` | No | `up` (ancestors only), `down` (subtree only). Default when `<feature_id>` given: both. Invalid without `<feature_id>` |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../../spec/features/cli/feature/_args/fields.md) | No | Inline metadata (e.g., `status,oq`). Auto-switches output to YAML |
| `--format` | No | Output format: `yaml`, `json`, `text`. Auto-selects `yaml` when `--fields` is set |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success | Read the indented tree output |
| `2` | Invalid arguments (e.g., `--direction` without `<feature_id>`) | Check parameter values |
| `3` | Feature or project not found | Verify the feature ID or project identifier |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Full project tree

```bash
synchestra feature tree --project synchestra
# agent-skills
# claim-and-push
# cli
# 	feature
# 	task
# conflict-resolution
# cross-repo-sync
```

### Feature in context (ancestors + subtree)

```bash
synchestra feature tree cli/task
# cli
# 	* task
# 		claim
# 		create
# 		list
# 		update
```

The `*` marks the target feature.

### Ancestors only

```bash
synchestra feature tree cli/task --direction=up
# cli
# 	* task
```

### Subtree only

```bash
synchestra feature tree cli/task --direction=down
# * task
# 	claim
# 	create
# 	list
# 	update
```

### Tree with status fields

```bash
synchestra feature tree cli/task --fields=status,oq
```

```yaml
- path: cli
  status: "In Progress"
  oq: 3
  children:
    - path: cli/task
      focus: true
      status: "Conceptual"
      oq: 2
      children:
        - path: cli/task/claim
          status: "Conceptual"
          oq: 0
```

## Notes

- This is a **read-only** command — it never mutates state.
- Indentation uses tab characters — one tab per nesting level.
- Nested features show only the leaf name; the full path is implied by context.
- `*` (text) / `focus: true` (YAML) marks the target feature when `<feature_id>` is given.
- For a flat list with full feature IDs, use `feature list` instead.
- Use `feature deps`/`feature refs` to trace dependency relationships (not hierarchy).
