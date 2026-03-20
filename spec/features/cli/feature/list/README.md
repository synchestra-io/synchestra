# Command: `synchestra feature list`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-list](../../../../../ai-plugin/skills/synchestra-feature-list/README.md)

## Synopsis

```
synchestra feature list [--project <project_id>] [--fields <fields>]
```

## Description

Lists all features in a project as full feature IDs, one per line, sorted alphabetically. Each line is a feature ID — the path relative to the project's features directory using `/` as separator.

**`list` vs `tree`:** `list` outputs a flat, grep/pipe-friendly list with full paths (e.g., `cli/feature/deps`). Use it for machine processing, counting, filtering, or feeding IDs into other commands. For a visual hierarchy showing parent-child nesting, use [`feature tree`](../tree/README.md) instead.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../_args/fields.md) | No | Inline selected metadata next to each feature. Auto-switches output to YAML |

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

### Format behaviour

- Without `--fields`: plain text, one feature ID per line.
- With `--fields`: auto-switches to YAML. Each feature becomes a YAML node with the requested fields.
- `--format` overrides in either direction.

## Output

### Default (plain text)

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

### With fields (auto-switches to YAML)

```bash
synchestra feature list --fields=status,oq
```

```yaml
- path: agent-skills
  status: "In Progress"
  oq: 3
- path: cli
  status: "In Progress"
  oq: 3
- path: cli/feature
  status: "Conceptual"
  oq: 1
- path: cli/task
  status: "Conceptual"
  oq: 2
- path: task-status-board
  status: "Conceptual"
  oq: 4
```

## Outstanding Questions

None at this time.
