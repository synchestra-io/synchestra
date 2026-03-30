# Command: `synchestra feature deps`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-deps](../../../../../ai-plugin/skills/synchestra-feature-deps/README.md)

## Synopsis

```
synchestra feature deps <feature_id> [--project <project_id>] [--fields <fields>] [--transitive]
```

## Description

Shows the features that a given feature depends on. Dependencies are read from the `## Dependencies` section in the feature's `README.md`. Each dependency is output as a feature ID, one per line.

This is the spec → spec counterpart to [`synchestra code deps`](../../code/deps/README.md), which shows code → spec dependencies via [source references](https://github.com/synchestra-io/specscore/blob/main/spec/features/source-references/README.md).

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional argument, e.g., `cross-repo-sync`, `cli/task`) |
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../_args/fields.md) | No | Inline selected metadata next to each feature. Auto-switches output to YAML |
| [`--transitive`](../_args/transitive.md) | No | Follow the full dependency chain recursively instead of showing only direct dependencies |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success (including when the feature has no dependencies — outputs nothing) |
| `2` | Invalid arguments |
| `3` | Feature or project not found |

## Behaviour

1. Pull latest state from the spec repository
2. Locate the feature's `README.md` at `{features_dir}/{feature_id}/README.md`
3. Parse the `## Dependencies` section for bullet-listed feature IDs
4. Validate that each listed dependency exists as a feature in the project
5. If `--transitive`, recursively resolve each dependency's own dependencies, detecting and marking cycles
6. Output each dependency as a feature ID, one per line, sorted alphabetically

If the feature has no `## Dependencies` section or the section is empty, the command outputs nothing and exits with code `0`.

If a listed dependency does not exist as a feature in the project, the command outputs it with a `(not found)` suffix to stderr but does not fail — this allows forward references to planned features.

### Format behaviour

- Without `--fields`: plain text, one feature ID per line. With `--transitive`, indentation shows depth.
- With `--fields`: auto-switches to YAML. Each dependency becomes a YAML node with the requested fields. With `--transitive`, nesting uses `children` keys.
- `--format` overrides in either direction.

## Output

Plain text, one dependency feature ID per line:

```bash
synchestra feature deps cross-repo-sync --project synchestra
```

```
claim-and-push
conflict-resolution
```

### Transitive dependencies

```bash
synchestra feature deps cli/task --transitive
```

```
task-status-board
  conflict-resolution
state-store
```

### With fields (auto-switches to YAML)

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

### Feature with no dependencies

```bash
synchestra feature deps micro-tasks --project synchestra
# (no output — exit code 0)
```

## Outstanding Questions

None at this time.
