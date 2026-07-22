# --agent

Requests an exact agent adapter.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Scheduler-selected |

## Supported by

`synchestra runner dispatch`

## Description

The CLI passes this selector unchanged. Eligibility validation occurs at the Hub boundary.

## Examples

```bash
synchestra runner dispatch --prompt "Update docs" --agent claude-code
```

## Outstanding Questions

None at this time.
