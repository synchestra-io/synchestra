# `--deadline`

Optional immutable deadline for starting or completing the handler invocation.

| Field | Value |
|---|---|
| **Type** | RFC3339 timestamp |
| **Required** | No |
| **Default** | — |

## Supported by

- [`runner invoke`](../README.md)

## Description

The CLI accepts RFC3339 or RFC3339Nano input and records the same instant in
canonical UTC. A supplied deadline becomes part of the immutable dispatch
intent, so an idempotent replay must supply the same instant. The durable
dispatch record, rather than the caller, supplies canonical creation time.

## Examples

```bash
--deadline 2026-08-25T18:00:00Z
--deadline 2026-08-25T19:00:00+01:00
```

## Open Questions

None at this time.
