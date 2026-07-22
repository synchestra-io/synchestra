# --cursor

Selects the first unread log position.

| Detail | Value |
|---|---|
| Type | Non-negative integer |
| Required | No |
| Default | `0` |

## Supported by

`synchestra runner dispatch logs`

## Description

The value is sent as the public logs endpoint's `cursor` query parameter. The response returns `next_cursor` for the next request.

## Examples

```bash
synchestra runner dispatch logs dsp_01HXYZ --cursor 24
```

## Outstanding Questions

None at this time.
