# --rules

Selectively enables specific linting rules for `synchestra spec lint`.

| Detail | Value |
|---|---|
| Type | String (CSV list) |
| Required | No |
| Default | All rules enabled |

## Syntax

```bash
synchestra spec lint --rules RULE1,RULE2,RULE3
```

Comma-separated rule names (no spaces). Each rule becomes an enabled filter; all other rules are skipped.

## Supported Rules

| Rule | Level | Applies to |
|---|---|---|
| `readme-exists` | error | All spec directories |
| `oq-section` | error | Feature/plan READMEs |
| `feature-ref-syntax` | error | All markdown files |
| `internal-links` | error | All markdown files |
| `index-entries` | error | Feature READMEs |
| `oq-not-empty` | warning | Feature/plan READMEs |
| `heading-levels` | warning | All markdown files |
| `forward-refs` | warning | Feature/plan READMEs |
| `code-annotations` | warning | Go source files |

## Examples

```bash
# Check only README and OQ section rules
synchestra spec lint --rules readme-exists,oq-section

# Check README, OQ, and feature reference syntax
synchestra spec lint --rules readme-exists,oq-section,feature-ref-syntax

# Invalid rule name (exit code 2)
synchestra spec lint --rules readme-exists,nonexistent-rule
# Error: unknown rule "nonexistent-rule"
```

## Relationship to `--ignore`

`--rules` and `--ignore` are mutually exclusive:
- `--rules A,B,C` enables only those rules
- `--ignore A,B,C` disables those rules, enables all others
- If both are specified, exit code 2 with error message

## Outstanding Questions

None at this time.
