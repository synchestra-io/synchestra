# --transitive

Follow dependency/reference chains to their full depth instead of just direct relationships.

| Detail | Value |
|---|---|
| Type | Boolean flag |
| Required | No |
| Default | `false` (show only direct dependencies/references) |

## Supported by

| Command | Behaviour |
|---|---|
| `feature deps` (`../deps/README.md`) | Shows the full transitive dependency chain — dependencies of dependencies, recursively |
| `feature refs` (`../refs/README.md`) | Shows the full transitive reference chain — features that reference features that reference the target, recursively |

Not supported by `feature list`, `feature tree`, `feature info` — these don't operate on relationship chains.

## Description

Without `--transitive`, only direct (first-level) dependencies or references are shown.
With `--transitive`, the full chain is followed recursively until all leaf nodes are reached.

Circular dependencies are detected and reported — the cycle is broken, not infinite.

Output uses nested YAML `children` to show the dependency tree structure when combined with `--fields`.
Without `--fields`, transitive output uses indentation (like `feature tree`) to show depth.

## Examples

```bash
# Direct deps only (default)
synchestra feature deps cli/task
task-status-board
state-store

# Transitive deps — follow the full chain
synchestra feature deps cli/task --transitive
task-status-board
  conflict-resolution
state-store

# Transitive with fields — YAML output
synchestra feature deps cli/task --transitive --fields=status
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

### Cycle detection

```bash
# If A depends on B and B depends on A
synchestra feature deps feature-a --transitive
feature-b
  feature-a (cycle)
```

## Outstanding Questions

- Should `--depth <n>` be supported to limit transitive resolution to N levels?
- Should cycle detection output the full cycle path or just mark the node?
