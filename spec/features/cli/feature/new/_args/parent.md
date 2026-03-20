# `--parent`

Parent feature ID for creating a sub-feature.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | No |
| **Default** | — |

## Supported by

- [`feature new`](../README.md)

## Description

When provided, the new feature is created as a child of the specified parent feature. The parent feature must already exist (its directory must contain a `README.md`); otherwise the command exits with code `3`.

The parent's `## Contents` section is automatically updated to include the new child feature in its index table.

If both `--parent` and a slash-containing `--slug` are provided, the command exits with code `2` (ambiguous nesting — use one or the other).

## Examples

```bash
# Create "claim" as a sub-feature of "cli/task"
synchestra feature new --title "Claim" --parent "cli/task"

# Equivalent using slash in slug (no --parent needed)
synchestra feature new --title "Claim" --slug "cli/task/claim"
```

## Outstanding Questions

None at this time.
