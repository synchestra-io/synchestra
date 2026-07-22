# --plan

Constrains SpecScore target resolution to Plans.

| Detail | Value |
|---|---|
| Type | String |
| Required | Conditional |
| Default | None |

## Supported by

`synchestra runner dispatch`

## Description

Accepts a committed Plan path, canonical ID, or unambiguous name. It is mutually exclusive with `--prompt`, `--task`, and the positional target.

## Examples

```bash
synchestra runner dispatch --plan PLAN-42
```

## Outstanding Questions

None at this time.
