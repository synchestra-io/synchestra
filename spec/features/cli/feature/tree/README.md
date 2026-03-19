# Command: `synchestra feature tree`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-tree](../../../../../skills/synchestra-feature-tree/README.md)

## Synopsis

```
synchestra feature tree [--project <project_id>]
```

## Description

Displays the feature hierarchy as an indented tree. Top-level features are printed at the root, and nested features are indented with tabs to show parent-child relationships.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Project not found |

## Behaviour

1. Pull latest state from the spec repository
2. Walk the project's features directory recursively
3. Each directory containing a `README.md` is a feature
4. Output the feature hierarchy with tab indentation for each nesting level, sorted alphabetically at each level

## Output

Indented tree using tabs, one feature per line:

```
agent-skills
claim-and-push
cli
	feature
	task
conflict-resolution
cross-repo-sync
micro-tasks
model-selection
outstanding-questions
task-status-board
```

Each nesting level adds one tab character. Only the leaf name is shown for nested features — the full path is implied by the indentation context.

## Outstanding Questions

- Should the tree include feature status (e.g., `cli [In Progress]`) alongside each name?
- Should there be a `--depth` flag to limit tree depth?
