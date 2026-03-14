# Command: `synchestra config set`

**Parent:** [config](../README.md)

## Synopsis

```
synchestra config set --repos-dir <path>
```

## Description

Sets one or more config values in `~/.synchestra.yaml`. Creates the file if it does not exist. At least one flag is required; the command exits with code `2` if called with no arguments. Empty values are not allowed — the command exits with code `2` if a flag is passed with an empty string.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--repos-dir`](../_args/repos-dir.md) | No | Root directory for cloned repositories |

At least one parameter must be provided.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Config updated successfully |
| `2` | Invalid arguments (no arguments provided, or empty value) |
| `10+` | Unexpected error |

## Behaviour

1. Validate at least one flag is provided; exit `2` if not
2. Validate no flag has an empty value; exit `2` if so
3. Read `~/.synchestra.yaml` if it exists; otherwise start with an empty config
4. Update the specified fields
5. Write `~/.synchestra.yaml`
6. Exit `0`

## Outstanding Questions

None at this time.
