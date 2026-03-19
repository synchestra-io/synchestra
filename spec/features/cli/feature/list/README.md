# Command: `synchestra feature list`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-list](../../../../../skills/synchestra-feature-list/README.md)

## Synopsis

```
synchestra feature list [--project <project_id>]
```

## Description

Lists all features in a project as full feature IDs, one per line, sorted alphabetically. Each line is a feature ID — the path relative to the project's features directory using `/` as separator.

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
4. Output the full feature ID (relative path) for each feature, one per line, sorted alphabetically

## Output

Plain text, one feature ID per line:

```
agent-skills
claim-and-push
cli
cli/feature
cli/task
conflict-resolution
cross-repo-sync
micro-tasks
model-selection
outstanding-questions
task-status-board
```

Nested features appear with their full path. Both the parent (`cli`) and the child (`cli/task`) are listed as separate features.

## Outstanding Questions

None at this time.
