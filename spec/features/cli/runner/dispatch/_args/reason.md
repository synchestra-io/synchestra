# --reason

Adds an optional audit reason to a caller mutation.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Empty |

## Supported by

`synchestra runner dispatch retry` and `synchestra runner dispatch cancel`.

## Description

The text is sent unchanged in the versioned retry/cancel request; it does not affect local state.

## Examples

```bash
synchestra runner dispatch cancel dsp_01HXYZ --reason "superseded"
```

## Outstanding Questions

None at this time.
