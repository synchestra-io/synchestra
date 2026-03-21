# --refs

Enriches search results with cross-reference context for each matching feature.

| Detail | Value |
|---|---|
| Type | Boolean (flag) |
| Required | No |
| Default | `false` |

## Syntax

```bash
synchestra spec search <query> --refs
```

## Behavior

When `--refs` is set, each search result is enriched with the following fields for its containing feature:

| Field | Description |
|---|---|
| `deps` | Direct dependencies of the containing feature |
| `reverse_deps` | Features that directly depend on the containing feature |
| `plans` | Plans that reference the containing feature |

This information is derived from the same parsing used by `feature deps` and `feature refs`. It is computed once per feature, not per match — if a feature has 5 matches, the refs are computed once and attached to all 5.

### Why this exists

An agent searching for "optimistic locking" may find a match in `cli/task/claim`. Knowing that `agent-skills` depends on `cli/task/claim` and that the `agent-skills-roadmap` plan references it gives the agent immediate blast-radius awareness without additional CLI calls.

### Performance note

`--refs` requires reading dependency metadata from all features in the spec tree (to compute reverse dependencies). On large spec trees this adds latency. For narrow queries where cross-reference context is unnecessary, omit `--refs`.

## Examples

```bash
# Search with full cross-reference context
synchestra spec search "claiming" --refs

# Combine with feature scoping (refs still computed for matched features)
synchestra spec search "timeout" --feature cli/task --refs --format yaml
```

## Outstanding Questions

- Should `--refs` support `--transitive` to include transitive (not just direct) dependencies and references?
