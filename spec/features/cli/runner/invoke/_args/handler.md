# `--handler`

Code-registered fixed handler invoked by the selected runner.

| Field | Value |
|---|---|
| **Type** | Enum: `wb.session.accept.v1`, `wb.session.message.v1` |
| **Required** | Yes |
| **Default** | — |

## Supported by

- [`runner invoke`](../README.md)

## Description

The value selects a closed handler registry entry and its scheduler capability.
It is not an executable, subcommand, shell fragment, or argv template. Unknown
values fail before the Hub is contacted.

## Examples

```bash
--handler wb.session.accept.v1
--handler wb.session.message.v1
```

## Open Questions

None at this time.
