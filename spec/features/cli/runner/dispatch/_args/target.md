# <target>

Auto-resolves a committed SpecScore Plan or Task.

| Detail | Value |
|---|---|
| Type | String (positional) |
| Required | Conditional |
| Default | None |

## Supported by

`synchestra runner dispatch`

## Description

Mutually exclusive with `--prompt`, `--plan`, and `--task`. Resolution searches committed paths, IDs, exact normalized names, then normalized substring names across both resource kinds. Ambiguity is an error and never prompts.

## Examples

```bash
synchestra runner dispatch spec/plans/dependency-upgrade.md
```

## Outstanding Questions

None at this time.
