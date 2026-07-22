# `--prompt`

Supplies an ad-hoc work description without requiring a pre-existing Synchestra Task.

| Detail | Value |
|---|---|
| Type | String |
| Required | Conditional; mutually exclusive with `<target>` |
| Default | — |

## Behavior

The CLI records the prompt together with repository identity and the immutable base revision resolved from the current Git checkout. Empty prompts are rejected.

## Examples

```bash
synchestra runner dispatch --prompt "Update dal-go dependencies to latest" --model sonnet
```

## Outstanding Questions

None at this time.
