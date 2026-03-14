# Command: `synchestra config show`

**Parent:** [config](../README.md)

## Synopsis

```
synchestra config show
```

## Description

Displays the effective global configuration. Reads `~/.synchestra.yaml` and outputs all fields with their current values. Fields that are absent or empty in the file are populated with their default values in the output, so the consumer always sees the complete configuration without needing to know the default logic.

If `~/.synchestra.yaml` does not exist, the output contains all fields set to their defaults.

Output format is YAML, matching the file format.

## Parameters

None.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `10+` | Unexpected error |

## Behaviour

1. Read `~/.synchestra.yaml` if it exists; otherwise start with an empty config
2. For each field that supports a default value, fill in the default if absent or empty
3. Output the complete configuration as YAML to stdout
4. Exit `0`

## Outstanding Questions

None at this time.
