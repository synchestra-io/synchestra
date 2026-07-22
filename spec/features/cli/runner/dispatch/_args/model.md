# `--model`

Requests an exact or adapter-specific model selector.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Resolved from `--profile` or project routing rules |

## Behavior

The selector takes precedence over profile-to-model mapping while preserving the requested profile in the dispatch record. Aliases such as `sonnet` are interpreted by the selected agent adapter. An unavailable exact selector fails unless an explicit fallback policy exists.

## Examples

```bash
synchestra runner dispatch --prompt "Update dal-go dependencies to latest" --model sonnet
```

## Outstanding Questions

None at this time.
