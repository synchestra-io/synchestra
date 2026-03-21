# --status

Filters search results to only include files belonging to features with the specified status.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | (none — all statuses included) |

## Syntax

```bash
synchestra spec search <query> --status <status>
```

## Behavior

Before searching file contents, `spec search` reads the `**Status:**` line from each feature's README to determine the feature's status. Only files belonging to features whose status matches `<status>` (case-insensitive) are searched.

Valid status values follow the feature lifecycle: `Conceptual`, `In Progress`, `Implemented`, `Stable`, `Deprecated`.

Files not associated with a feature (e.g., top-level `spec/README.md`) are excluded when `--status` is set, since they have no status to match against.

## Examples

```bash
# Find in-progress features mentioning "token"
synchestra spec search token --status "In Progress"

# Find conceptual features with open questions about scaling
synchestra spec search "scaling" --status Conceptual --section oq

# Combine with type filter
synchestra spec search "mutation" --status "In Progress" --type feature
```

## Outstanding Questions

- Should `--status` accept multiple values (comma-separated) to include features in any of the listed statuses?
