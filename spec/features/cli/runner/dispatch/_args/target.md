# `<target>`

Identifies a SpecScore Plan or Task to dispatch.

| Detail | Value |
|---|---|
| Type | String (positional) |
| Required | Conditional; mutually exclusive with `--prompt` |
| Default | — |

## Behavior

The CLI resolves a path first, then an explicit resource ID, then an unambiguous active-resource name. Ambiguous or missing targets fail without creating a dispatch. The resolved repository and source revision are recorded immutably.

## Examples

```bash
synchestra runner dispatch spec/plans/upgrade-dependencies.md
synchestra runner dispatch TASK-1024 --profile balanced
```

## Outstanding Questions

None at this time.
