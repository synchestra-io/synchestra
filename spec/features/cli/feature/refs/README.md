# Command: `synchestra feature refs`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-refs](../../../../../ai-plugin/skills/synchestra-feature-refs/README.md)

## Synopsis

```
synchestra feature refs <feature_id> [--project <project_id>] [--fields <fields>] [--transitive]
```

## Description

Shows features that reference (depend on) a given feature. This is the inverse of `deps` — it scans all features' `## Dependencies` sections to find those that list the given feature ID.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional argument, e.g., `claim-and-push`, `cli/task`) |
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../_args/fields.md) | No | Inline selected metadata next to each feature. Auto-switches output to YAML |
| [`--transitive`](../_args/transitive.md) | No | Follow the full reference chain recursively — features that depend on features that depend on the target |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success (including when no features reference this one — outputs nothing) |
| `2` | Invalid arguments |
| `3` | Feature or project not found |

## Behaviour

1. Pull latest state from the spec repository
2. Verify the target feature exists at `{features_dir}/{feature_id}/README.md`
3. Scan all features' `README.md` files for a `## Dependencies` section
4. Collect features whose `## Dependencies` section lists the target feature ID
5. If `--transitive`, recursively find features that reference the collected features, detecting and marking cycles
6. Output each referencing feature as a feature ID, one per line, sorted alphabetically

If no features reference the target, the command outputs nothing and exits with code `0`.

### Format behaviour

- Without `--fields`: plain text, one feature ID per line. With `--transitive`, indentation shows depth.
- With `--fields`: auto-switches to YAML. Each reference becomes a YAML node with the requested fields. With `--transitive`, nesting uses `children` keys.
- `--format` overrides in either direction.

## Output

Plain text, one referencing feature ID per line:

```bash
synchestra feature refs claim-and-push --project synchestra
```

```
conflict-resolution
cross-repo-sync
```

### Transitive references

```bash
synchestra feature refs state-store --transitive
```

```
cli/task
  agent-skills
task-status-board
```

### With fields (auto-switches to YAML)

```bash
synchestra feature refs state-store --transitive --fields=status,oq
```

```yaml
- path: cli/task
  status: "Conceptual"
  oq: 2
  children:
    - path: agent-skills
      status: "In Progress"
      oq: 3
- path: task-status-board
  status: "Conceptual"
  oq: 4
```

### Feature with no references

```bash
synchestra feature refs micro-tasks --project synchestra
# (no output — exit code 0)
```

## Outstanding Questions

None at this time.
