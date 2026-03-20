# Command: `synchestra feature info`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-info](../../../../../ai-plugin/skills/synchestra-feature-info/README.md)

## Synopsis

```
synchestra feature info <feature_id> [--project <project_id>] [--format <format>]
```

## Description

Returns structured metadata and a section table-of-contents with line ranges for a feature's `README.md`, enabling agents to surgically read only the sections they need.

The default output format is YAML (agent-first). Use `--format text` for human-readable tables.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional argument, e.g., `cli/task/claim`) |
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--format`](../../_args/format.md) | No | Output format. Supported values: `yaml` (default), `json`, `text` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Feature or project not found |

## Behaviour

1. Pull latest state from the spec repository
2. Locate the feature's `README.md` at `{features_dir}/{feature_id}/README.md`
3. Extract metadata: `path`, `status` (from the Status line in the README), `deps` (from `## Dependencies` section), `refs` (by scanning all features' Dependencies sections)
4. Discover child sub-features: scan for immediate child directories containing `README.md`, check if each child is listed in the parent's `## Contents` table, set `in_readme` accordingly
5. Find linked plans by scanning `spec/plans/` for plans referencing this feature
6. Parse README headings to build section TOC with line ranges — include nested sub-headings as `children`
7. For sections that contain lists (Outstanding Questions, Dependencies, Acceptance Criteria), include `items` count
8. Output as YAML by default

## Output

Default format is YAML:

```bash
synchestra feature info cli/task/claim --project synchestra
```

```yaml
path: cli/task/claim
status: "Conceptual"
deps: [task-status-board, state-store]
refs: [agent-skills]
children:
  - path: task/claim/substep
    in_readme: true
  - path: task/claim/other
    in_readme: false
plans: [e2e-testing-framework]
sections:
  - title: Summary
    lines: 3-5
  - title: Problem
    lines: 7-14
  - title: Behavior
    lines: 16-52
    children:
      - title: Claiming Protocol
        lines: 18-35
      - title: Conflict Handling
        lines: 37-52
  - title: Dependencies
    lines: 54-58
    items: 2
  - title: Acceptance Criteria
    lines: 60-78
    items: 5
  - title: Outstanding Questions
    lines: 80-91
    items: 3
```

### `children` as consistency check

The `children` field lists sub-feature directories discovered on disk with `in_readme` indicating whether each child is listed in the parent's `## Contents` table. `in_readme: false` signals the spec tree is out of sync.

## Design Rationale

Metadata and sections are merged into one command because by the time an agent calls `info`, it has already identified the feature (via `list`/`tree`) and wants both the overview and the roadmap for selective reading.

## Outstanding Questions

- Should `feature info` support `--sections-only` (skip metadata) or `--meta-only` (skip sections) flags?
- How deep should `sections` nesting go? Only `h2` + `h3`? Or all heading levels?
- Should sections include the heading's markdown level (e.g., `level: 2`) for agents that want to understand document structure?
