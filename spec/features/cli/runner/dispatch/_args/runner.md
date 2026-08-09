# `--runner`

Constrains a dispatch to one registered worker.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Any authorized eligible worker |

## Behavior

When present, the scheduler considers only the named worker. When omitted, any authorized worker whose capabilities satisfy the dispatch may lease it.

## Examples

```bash
synchestra runner dispatch --prompt "Run the upgrade" --runner personal-vm
```

## Outstanding Questions

None at this time.
