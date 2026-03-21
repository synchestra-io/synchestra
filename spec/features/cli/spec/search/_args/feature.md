# --feature

Scopes the search to a specific feature subtree within the specification tree.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | (none — searches all of `spec/`) |

## Syntax

```bash
synchestra spec search <query> --feature <feature-path>
```

The `<feature-path>` is a feature identifier relative to `spec/features/` (e.g., `cli/task`, `agent-skills`, `cli/spec/lint`). The search is restricted to `spec/features/<feature-path>/` and all its descendants.

## Behavior

- Resolves `<feature-path>` to the directory `spec/features/<feature-path>/`.
- If the directory does not exist, exits with code 3 (feature not found).
- Mutually exclusive with the `PATH` positional argument. If both are given, exits with code 2 (invalid arguments).
- When combined with `--type plan`, searches plans that reference the specified feature (plans live under `spec/plans/`, not under the feature directory). The command resolves cross-references to include relevant plans.

## Examples

```bash
# Search within the CLI task feature subtree
synchestra spec search "conflict" --feature cli/task

# Search within agent-skills and all sub-features
synchestra spec search "skill" --feature agent-skills

# Combine with section filter
synchestra spec search "version" --feature cli --section "Outstanding Questions"
```

## Outstanding Questions

- Should `--feature` accept glob patterns (e.g., `cli/*`) for multi-subtree search?
