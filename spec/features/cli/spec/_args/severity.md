# --severity

Controls the minimum severity level for violations reported by `synchestra spec lint`.

| Detail | Value |
|---|---|
| Type | String (enum) |
| Required | No |
| Default | `error` |

## Syntax

```bash
synchestra spec lint --severity LEVEL
```

Valid levels: `error`, `warning`, `info`. Case-insensitive.

## Severity Levels

| Level | Description | When to use |
|---|---|---|
| `error` | Must fix — spec tree will not be valid for mutation commands | Default. Use in CI/CD and pre-commit hooks. |
| `warning` | Should fix — spec tree is usable but has drift or suboptimal practices | Development and code review. |
| `info` | Advisory — informational diagnostics (reserved for future use) | Detailed analysis and debugging. |

When `--severity` is set to a level, all violations at that level and higher severity are reported. E.g., `--severity warning` reports both errors and warnings; `--severity info` reports errors, warnings, and infos.

## Examples

```bash
# Default: errors only
synchestra spec lint
# Output: 3 errors found

# Errors and warnings
synchestra spec lint --severity warning
# Output: 5 violations found (3 errors, 2 warnings)

# All diagnostics (errors, warnings, infos)
synchestra spec lint --severity info
# Output: 6 violations found (3 errors, 2 warnings, 1 info)

# Invalid level (exit code 2)
synchestra spec lint --severity critical
# Error: invalid severity level "critical"
```

## Interaction with `--rules` and `--ignore`

`--severity` is orthogonal to rule selection:
- `--rules` / `--ignore` determines *which* rules run
- `--severity` determines *which* rule violations are reported

Example:
```bash
# Run all rules but only report errors (not warnings)
synchestra spec lint --severity error

# Run only readme-exists and oq-section, report both errors and warnings
synchestra spec lint --rules readme-exists,oq-section --severity warning
```

## Outstanding Questions

None at this time.
