# --profile

Requests a provider-neutral execution profile.

| Detail | Value |
|---|---|
| Type | Enum: `fast`, `balanced`, `large` |
| Required | No |
| Default | `balanced` |

## Supported by

`synchestra runner dispatch`

## Description

The requested profile is recorded unchanged. Mapping the profile to a model belongs to the Hub scheduler.

## Examples

```bash
synchestra runner dispatch --prompt "Fix formatting" --profile fast
```

## Outstanding Questions

None at this time.
