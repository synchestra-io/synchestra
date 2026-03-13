# --description

Task description included in the generated task `README.md`.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | — |

## Supported by

[`task create`](../README.md)

## Description

When provided, this text is written into the body of the task's `README.md` file. It should describe what the task involves, acceptance criteria, or any context an agent needs to understand the work.

If omitted, the task `README.md` is created with the title only.

## Examples

```bash
synchestra task create --project my-service --task fix-auth-bug \
  --title "Fix authentication bypass bug" \
  --description "Users can bypass auth by sending an empty token header. Fix the middleware to reject empty tokens."
```

## Outstanding Questions

- Should the description be read from stdin if not provided as a flag?
