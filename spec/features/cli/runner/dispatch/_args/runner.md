# --runner

Constrains scheduling to an exact registered runner.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Any eligible worker |

## Supported by

`synchestra runner dispatch`

## Description

The value is passed unchanged as `worker_constraints.runner_id`. The CLI does not inspect internal scheduler state.

## Examples

```bash
synchestra runner dispatch --prompt "Run validation" --runner personal-vm
```

## Outstanding Questions

None at this time.
