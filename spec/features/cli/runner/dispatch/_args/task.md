# --task

Constrains SpecScore target resolution to Tasks.

| Detail | Value |
|---|---|
| Type | String |
| Required | Conditional |
| Default | None |

## Supported by

`synchestra runner dispatch`

## Description

Accepts a committed Task path, canonical ID, or unambiguous name. Numbered Plan tasks use `{plan-id}#task-{number}`. It is mutually exclusive with all other source forms.

## Examples

```bash
synchestra runner dispatch --task PLAN-42#task-2
```

## Outstanding Questions

None at this time.
