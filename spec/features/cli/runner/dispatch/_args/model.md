# --model

Requests an exact or adapter-specific model selector.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Scheduler-selected from profile |

## Supported by

`synchestra runner dispatch`

## Description

Values such as `sonnet` are passed unchanged as `model_selector`. The CLI records reject fallback and performs no local mapping.

## Examples

```bash
synchestra runner dispatch --prompt "Update dependencies" --model sonnet
```

## Outstanding Questions

None at this time.
