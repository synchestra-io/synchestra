# --prompt

Supplies ad-hoc repository-scoped work without creating a Task.

| Detail | Value |
|---|---|
| Type | String |
| Required | Conditional |
| Default | None |

## Supported by

`synchestra runner dispatch`

## Description

Mutually exclusive with every SpecScore target form. Empty prompts are rejected. The exact prompt is stored in the v1 `ad_hoc` source and is not rewritten with model or fallback instructions.

## Examples

```bash
synchestra runner dispatch --prompt "Update dependencies"
```

## Outstanding Questions

None at this time.
