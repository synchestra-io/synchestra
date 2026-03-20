# `--title`

Human-readable title for the new feature.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | Yes |
| **Default** | — |

## Supported by

- [`feature new`](../README.md)

## Description

The feature title appears as the `# Feature: {title}` heading in the generated README. It is also used to auto-generate the feature slug (directory name) when `--slug` is not provided.

The title should be concise and descriptive — typically 2–5 words that identify the feature's purpose.

## Examples

```bash
synchestra feature new --title "Task Status Board"
synchestra feature new --title "Cross-Repo Sync"
synchestra feature new --title "CLI" --parent "synchestra"
```

## Outstanding Questions

None at this time.
