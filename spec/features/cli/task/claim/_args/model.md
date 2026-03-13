# --model

Identifies the model the agent is using to work on a task.

| Detail | Value |
|---|---|
| Type | String |
| Required | Yes |
| Default | — |

## Supported by

[`task claim`](../README.md)

## Description

Records which AI model the claiming agent is running. This information is stored in the task metadata and shown in status queries and task listings — useful for debugging, auditing, and understanding which model capabilities were applied to which tasks.

Common values: `haiku`, `sonnet`, `opus`, but any string is accepted.

## Examples

```bash
synchestra task claim --project synchestra --task implement-cli \
  --run 4821 --model sonnet

synchestra task claim --project synchestra --task complex-refactor \
  --run 9933 --model opus
```

## Outstanding Questions

None at this time.
