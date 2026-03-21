# --ignore

Disables specific linting rules for `synchestra spec lint`.

| Detail | Value |
|---|---|
| Type | String (CSV list) |
| Required | No |
| Default | No rules ignored (all enabled by default) |

## Syntax

```bash
synchestra spec lint --ignore RULE1,RULE2,RULE3
```

Comma-separated rule names (no spaces). Each listed rule is disabled; all other rules run normally.

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
# Ignore code annotations (common in shared/multi-lang repos)
synchestra spec lint --ignore code-annotations

# Ignore forward references and code annotations
synchestra spec lint --ignore forward-refs,code-annotations --severity warning

# Invalid rule name (exit code 2)
synchestra spec lint --ignore readme-exists,nonexistent-rule
# Error: unknown rule "nonexistent-rule"
```

## Relationship to `--rules`

`--rules` and `--ignore` are mutually exclusive:
- `--rules A,B,C` enables only those rules
- `--ignore A,B,C` disables those rules, enables all others
- If both are specified, exit code 2 with error message

## Outstanding Questions

None at this time.
