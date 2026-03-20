# `--slug`

Override the auto-generated feature slug (directory name).

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | No |
| **Default** | Auto-generated from `--title` |

## Supported by

- [`feature new`](../README.md)

## Description

The slug determines the feature's directory name and ID. When omitted, it is auto-generated from the title (lowercase, hyphens for spaces, non-URL-safe characters removed).

When provided, the slug must comply with feature naming rules: lowercase, hyphen-separated, URL-safe. No underscores, spaces, or special characters.

If the slug contains slashes (`/`), it is treated as a full nested path relative to the features directory. This is an alternative to using `--parent` for creating sub-features. When both `--parent` and a slash-containing `--slug` are provided, the command exits with code `2` (ambiguous nesting).

## Examples

```bash
# Auto-generated slug: "task-status-board"
synchestra feature new --title "Task Status Board"

# Explicit slug override
synchestra feature new --title "Task Status Board" --slug "status-board"

# Slash in slug creates a nested feature (equivalent to --parent cli/task)
synchestra feature new --title "Claim" --slug "cli/task/claim"
```

## Outstanding Questions

None at this time.
