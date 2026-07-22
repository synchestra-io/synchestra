# <dispatch-id>

Identifies one durable dispatch.

| Detail | Value |
|---|---|
| Type | String (positional) |
| Required | Yes |
| Default | None |

## Supported by

`status`, `logs`, `retry`, and `cancel` under `synchestra runner dispatch`.

## Description

The ID returned by creation is routed to the corresponding public Hub caller endpoint. Invalid path-like IDs are rejected locally.

## Examples

```bash
synchestra runner dispatch status dsp_01HXYZ
```

## Outstanding Questions

None at this time.
