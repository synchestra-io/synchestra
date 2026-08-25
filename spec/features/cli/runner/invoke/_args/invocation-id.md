# `--invocation-id`

Caller-owned identity of the typed handler request.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | Yes |
| **Default** | — |

## Supported by

- [`runner invoke`](../README.md)

## Description

The identifier is supplied separately so Synchestra never parses identity from
the opaque payload. It is 1–128 bytes, starts with an ASCII letter or digit,
and thereafter permits ASCII letters, digits, `.`, `_`, `:`, and `-`.

For `wb.session.accept.v1`, this value is the WB handoff ID and derives the
durable dispatch idempotency key. The raw value is not embedded in that key.

## Examples

```bash
--invocation-id handoff-42
--invocation-id message:2026-08-25:17
```

## Open Questions

None at this time.
