# Skill: synchestra-feature-refs

Show what depends on a feature — list the features that reference it as a dependency.

**CLI reference:** [synchestra feature refs](../../spec/features/cli/feature/refs/README.md)

## When to use

- **Impact analysis:** Before changing a feature, see what else depends on it
- **Priority assessment:** Features with many refs are high-impact — changes affect downstream features
- **Planning:** Understand the downstream consequences of delaying or modifying a feature

## Command

```bash
synchestra feature refs <feature_id> \
  [--project <project_id>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional, e.g., `claim-and-push`, `cli/task`) |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success (including no references — empty output) | Parse the output — one referencing feature per line |
| `2` | Invalid arguments | Check parameter values |
| `3` | Feature or project not found | Verify the feature ID and project |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Show what depends on a feature

```bash
synchestra feature refs claim-and-push --project synchestra
# conflict-resolution
# cross-repo-sync
```

### Feature with no references

```bash
synchestra feature refs micro-tasks --project synchestra
# (no output — nothing depends on this feature)
```

### Assess impact before making changes

```bash
# 1. What depends on claim-and-push?
synchestra feature refs claim-and-push --project synchestra
# conflict-resolution
# cross-repo-sync

# 2. That's two downstream features — changes to claim-and-push will affect them.
```

## Notes

- This is a **read-only** command — it never mutates state.
- This is the inverse of `feature deps`. If `A` lists `B` in its dependencies, then `feature refs B` will include `A`.
- Output is plain text, one feature ID per line, sorted alphabetically.
- Empty output means no other feature depends on this one.
- For finding what a feature depends on (rather than what depends on it), use `feature deps`.
