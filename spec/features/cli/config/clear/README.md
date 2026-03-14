# Command: `synchestra config clear`

**Parent:** [config](../README.md)

## Synopsis

```
synchestra config clear --repos-dir
```

## Description

Removes one or more config values from `~/.synchestra.yaml`, reverting them to their defaults. Only fields that support default values can be cleared. At least one flag is required; the command exits with code `2` if called with no arguments.

The flags are boolean-style — pass the flag name without a value to clear that field.

If clearing results in an empty config, the file is left in place with no fields (not deleted).

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--repos-dir`](../_args/repos-dir.md) | No | Clear `repos_dir`, reverting to default (`~/synchestra/repos`) |

At least one parameter must be provided.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Config value(s) cleared successfully |
| `2` | Invalid arguments (no arguments provided, or field does not support defaults) |
| `3` | `~/.synchestra.yaml` not found |
| `10+` | Unexpected error |

## Behaviour

1. Validate at least one flag is provided; exit `2` if not
2. Read `~/.synchestra.yaml`; exit `3` if not found
3. Remove the specified fields from the config
4. Write `~/.synchestra.yaml`
5. Exit `0`

## Outstanding Questions

- Should `clear` succeed silently if the field is already absent, or exit with a specific code?
