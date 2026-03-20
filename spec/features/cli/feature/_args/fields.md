# --fields

Inline selected metadata next to each feature in the output of list/tree/deps/refs commands.

| Detail | Value |
|---|---|
| Type | Comma-separated list of field names |
| Required | No |
| Default | (none — bare output without metadata) |

## Supported by

| Command | Link |
|---|---|
| [`feature list`](../list/README.md) | |
| [`feature tree`](../tree/README.md) | |
| [`feature deps`](../deps/README.md) | |
| [`feature refs`](../refs/README.md) | |

## Available fields

| Field | Source |
|---|---|
| `status` | Feature status from README frontmatter |
| `oq` | Count of Outstanding Questions in the feature README |
| `deps` | List of dependency paths |
| `refs` | List of reference paths |
| `children` | List of child feature paths |
| `plans` | List of associated plan paths |
| `proposals` | List of associated proposal paths |

## Description

Enriches feature command output with inline metadata for each feature. Field values are computed the same way as in `feature info` — status from README, oq count from the Outstanding Questions section, etc.

Multiple fields are comma-separated: `--fields=status,oq,plans`.

### Format behaviour

- **Without `--fields`:** compact text (default for tree/deps/refs/list).
- **With `--fields`:** output auto-switches to YAML since the result is structured data.
- **`--format text|yaml|json`** overrides in either direction.

## Examples

```bash
# Tree with statuses and OQ counts — auto-switches to YAML
synchestra feature tree --fields=status,oq
```

```yaml
- path: cli
  status: "In Progress"
  oq: 3
  children:
    - path: cli/task
      status: "Conceptual"
      oq: 2
      children:
        - path: cli/task/claim
          status: "Conceptual"
          oq: 0
        - path: cli/task/release
          status: "Conceptual"
          oq: 1
```

```bash
# Deps with status
synchestra feature deps cli/task --fields=status --transitive
```

```yaml
- path: task-status-board
  status: "In Progress"
  children:
    - path: conflict-resolution
      status: "Conceptual"
- path: state-store
  status: "Conceptual"
```

## Outstanding Questions

- Should `--fields=all` be supported as a shorthand for all available fields?
- Should custom/computed fields be supported in the future (e.g., `sync` to check if children are in sync with README)?
