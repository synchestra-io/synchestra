# `@<payload-file>`

Path to the JSON file delivered as opaque bytes to the registered handler.

| Field | Value |
|---|---|
| **Type** | File path prefixed with `@` |
| **Required** | Yes |
| **Default** | — |

## Supported by

- [`runner invoke`](../README.md)

## Description

Exactly one positional argument is accepted and it must start with `@`.
Relative paths resolve from the caller's current directory. The file must
contain one syntactically valid, non-empty JSON value and be no larger than 1
MiB. Synchestra preserves its exact bytes and does not interpret JSON fields.

## Examples

```bash
synchestra runner invoke @handoff.json --runner personal-vm --handler wb.session.accept.v1 --invocation-id handoff-42
synchestra runner invoke @/tmp/message.json --runner personal-vm --handler wb.session.message.v1 --invocation-id message-17
```

## Open Questions

None at this time.
