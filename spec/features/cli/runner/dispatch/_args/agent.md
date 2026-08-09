# `--agent`

Overrides the worker's default agent adapter.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Worker or project default |

## Behavior

The scheduler accepts the override only when an eligible worker advertises the named agent. Unsupported overrides fail before a lease is issued.

## Examples

```bash
synchestra runner dispatch --prompt "Update dependencies" --agent claude-code
```

## Outstanding Questions

None at this time.
