# `--profile`

Requests a provider-neutral execution capability and cost profile.

| Detail | Value |
|---|---|
| Type | Enum: `fast`, `balanced`, `large` |
| Required | No |
| Default | `balanced`, after project routing rules |

## Behavior

The scheduler records the requested profile and resolves it through versioned project/worker routing configuration. It never silently upgrades to a more expensive profile.

## Examples

```bash
synchestra runner dispatch --prompt "Fix formatting" --profile fast
```

## Outstanding Questions

None at this time.
