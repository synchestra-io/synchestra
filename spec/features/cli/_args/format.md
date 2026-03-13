# --format

Controls the output format of read commands.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | `yaml` for `list`, `text` for `info` |

## Supported by

| Command | Allowed values | Default |
|---|---|---|
| [`list`](../task/list/README.md) | `yaml`, `json`, `md`, `csv` | `yaml` |
| [`info`](../task/info/README.md) | `text`, `json`, `yaml` | `text` |
| [`server status`](../server/status/README.md) | `text`, `json`, `yaml` | `text` |
| [`server project list`](../server/project/list/README.md) | `yaml`, `json`, `md`, `csv` | `yaml` |

## Description

Determines how command output is structured. Useful for both human readability and programmatic parsing.

- **`yaml`** — Structured, readable by both humans and machines. Default for `list`.
- **`json`** — Machine-readable, useful for piping to other tools.
- **`text`** — Human-readable plain text. Default for `info`.
- **`md`** — Markdown table format, suitable for embedding in READMEs. Supported by `list` and `server project list`.
- **`csv`** — Flat comma-separated values with a header row. Supported by `list` and `server project list`.

## Examples

```bash
# Default YAML listing
synchestra task list --project synchestra

# JSON for programmatic use
synchestra task list --project synchestra --format json

# Markdown table for embedding
synchestra task list --project synchestra --format md

# Task info as YAML
synchestra task info --project synchestra --task fix-bug --format yaml
```

## Outstanding Questions

None at this time.
