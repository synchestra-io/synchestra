# `--runner`

Exact registered runner that must execute the handler invocation.

| Field | Value |
|---|---|
| **Type** | String |
| **Required** | Yes |
| **Default** | — |

## Supported by

- [`runner invoke`](../README.md)

## Description

Unlike ordinary dispatch, typed invocation does not permit general scheduler
matching. The value becomes the dispatch's hard `runner_id` constraint and is
checked with the handler-specific required capability.

## Examples

```bash
synchestra runner invoke @handoff.json --runner personal-vm --handler wb.session.accept.v1 --invocation-id handoff-42
```

## Open Questions

None at this time.
