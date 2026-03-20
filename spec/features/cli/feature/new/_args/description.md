# `--description`

Short description placed in the Summary section of the generated README.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | No |
| **Default** | — |

## Supported by

- [`feature new`](../README.md)

## Description

When provided, the description text is placed in the `## Summary` section of the generated feature README. When omitted, a placeholder (`TODO: Brief summary of the feature.`) is used instead.

The description should be 1–3 sentences summarizing the feature's purpose. It also appears in the feature index row and the parent's Contents summary when those are updated.

## Examples

```bash
synchestra feature new --title "Task Status Board" \
  --description "A markdown table that tracks task assignments and status, using git commits as an optimistic locking mechanism."
```

## Outstanding Questions

None at this time.
