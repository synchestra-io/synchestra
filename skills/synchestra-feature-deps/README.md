# Skill: synchestra-feature-deps

Show what a feature depends on — list the features it requires to function or be built.

**CLI reference:** [synchestra feature deps](../../spec/features/cli/feature/deps/README.md)

## When to use

- **Planning work order:** Find out what must be built before starting on a feature
- **Understanding prerequisites:** See what a feature requires before diving into its spec
- **Dependency analysis:** Map out the dependency chain for a feature

## Command

```bash
synchestra feature deps <feature_id> \
  [--project <project_id>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | Yes | Feature ID to query (positional, e.g., `cross-repo-sync`, `cli/task`) |
| [`--project`](../../spec/features/cli/_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success (including no dependencies — empty output) | Parse the output — one dependency per line |
| `2` | Invalid arguments | Check parameter values |
| `3` | Feature or project not found | Verify the feature ID and project |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Show dependencies

```bash
synchestra feature deps cross-repo-sync --project synchestra
# claim-and-push
# conflict-resolution
```

### Feature with no dependencies

```bash
synchestra feature deps micro-tasks --project synchestra
# (no output — feature is independent)
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
- Output is plain text, one feature ID per line, sorted alphabetically.
- Empty output means the feature has no dependencies (it's independent).
- For the reverse — finding what depends on a feature — use `feature refs`.
