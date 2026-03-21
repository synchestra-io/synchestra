---
name: synchestra-feature-deps
description: Lists dependencies of a feature, optionally with transitive resolution and metadata fields. Use when checking prerequisites, planning work order, or analyzing dependency chains.
---

# Skill: synchestra-feature-deps

Show what a feature depends on — list the features it requires to function or be built.

**CLI reference:** [synchestra feature deps](../../spec/features/cli/feature/deps/README.md)

## When to use

- **Planning work order:** Find out what must be built before starting on a feature
- **Understanding prerequisites:** See what a feature requires before diving into its spec
- **Dependency analysis:** Map out the full dependency chain with `--transitive`
- **Status check:** Use `--fields=status` to see if dependencies are done

## Command

```bash
synchestra feature deps <feature_id> \
  [--project <project_id>] \
  [--fields <fields>] \
  [--transitive]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional, e.g., `cross-repo-sync`, `cli/task`) |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../../spec/features/cli/feature/_args/fields.md) | No | Inline metadata (e.g., `status,oq`). Auto-switches output to YAML |
| [`--transitive`](../../spec/features/cli/feature/_args/transitive.md) | No | Follow the full dependency chain recursively |
| `--format` | No | Output format: `yaml`, `json`, `text`. Auto-selects `yaml` when `--fields` is set |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success (including no dependencies — empty output) | Parse the output — one dependency per line |
| `2` | Invalid arguments | Check parameter values |
| `3` | Feature or project not found | Verify the feature ID and project |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Show direct dependencies

```bash
synchestra feature deps cross-repo-sync --project synchestra
# claim-and-push
# conflict-resolution
```

### Transitive dependencies

```bash
synchestra feature deps cli/task --transitive
# task-status-board
#   conflict-resolution
# state-store
```

### Dependencies with status

```bash
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

### Check before starting work

```bash
# 1. What does this feature need?
synchestra feature deps conflict-resolution --project synchestra
# claim-and-push

# 2. Is claim-and-push done? Check its tasks.
synchestra task list --project synchestra --status completed | grep claim-and-push
```

## Notes

- This is a **read-only** command — it never mutates state.
- Dependencies are declared in the feature's `README.md` under a `## Dependencies` section.
- Without `--fields`: plain text, one feature ID per line, sorted alphabetically.
- With `--fields`: output auto-switches to YAML with the requested metadata inline.
- `--transitive` resolves the full chain; cycles are detected and marked.
- Empty output means the feature has no dependencies (it's independent).
- For the reverse — finding what depends on a feature — use `feature refs`.
