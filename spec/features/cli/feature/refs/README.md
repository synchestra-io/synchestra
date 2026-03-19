# Command: `synchestra feature refs`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-refs](../../../../../skills/synchestra-feature-refs/README.md)

## Synopsis

```
synchestra feature refs <feature_id> [--project <project_id>]
```

## Description

Shows features that reference (depend on) a given feature. This is the inverse of `deps` — it scans all features' `## Dependencies` sections to find those that list the given feature ID.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional argument, e.g., `claim-and-push`, `cli/task`) |
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

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
5. Output each referencing feature as a feature ID, one per line, sorted alphabetically

If no features reference the target, the command outputs nothing and exits with code `0`.

## Output

Plain text, one referencing feature ID per line:

```bash
synchestra feature refs claim-and-push --project synchestra
```

```
conflict-resolution
cross-repo-sync
```

### Feature with no references

```bash
synchestra feature refs micro-tasks --project synchestra
# (no output — exit code 0)
```

## Outstanding Questions

- Should there be a `--recursive` flag to show transitive references (features that depend on features that depend on the target)?
