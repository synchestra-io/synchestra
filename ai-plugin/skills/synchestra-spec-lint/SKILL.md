---
name: synchestra-spec-lint
description: Validates a Synchestra specification repository for structural convention violations. Use when checking spec tree integrity, verifying conventions after edits, or as a pre-commit gate before mutation commands.
---

# Skill: synchestra-spec-lint

Validate the spec tree for structural convention violations — missing README.md files, absent Outstanding Questions sections, stale index entries, and more.

**CLI reference:** [synchestra spec lint](../../spec/features/cli/spec/lint/README.md)

## When to use

- **After editing specs:** Verify that changes didn't break structural conventions
- **Before mutation commands:** Run lint before `feature new`, `feature status`, or other mutation commands to ensure the spec tree is in a valid state
- **CI/CD gating:** Fail a pipeline when spec conventions are violated
- **Auditing spec health:** Check the entire spec tree for drift or missing sections
- **Scoping checks:** Use `--rules` to run only specific checks (e.g., just `readme-exists`)

## Command

```bash
synchestra spec lint \
  [PATH] \
  [--rules <csv>] \
  [--ignore <csv>] \
  [--severity <level>] \
  [--format <format>]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `PATH` | No | Spec root directory to lint (default: `./spec`) |
| [`--rules`](../../spec/features/cli/spec/_args/rules.md) | No | Enable only specified rules (comma-separated: `readme-exists,oq-section`) |
| [`--ignore`](../../spec/features/cli/spec/_args/ignore.md) | No | Disable specified rules (comma-separated: `forward-refs,code-annotations`) |
| [`--severity`](../../spec/features/cli/spec/_args/severity.md) | No | Minimum severity: `error` (default), `warning`, `info` |
| `--format` | No | Output format: `text` (default), `json`, `yaml` |

## Linting Rules

| Rule | Severity | What it checks |
|---|---|---|
| `readme-exists` | error | Every spec directory has a `README.md` file |
| `oq-section` | error | Feature/plan READMEs have `## Outstanding Questions` section |
| `oq-not-empty` | warning | OQ section has content or explicitly states "None at this time." |
| `index-entries` | error | Child directory references in READMEs resolve to existing directories |
| `heading-levels` | warning | No heading level gaps (e.g., H2 → H4 is invalid) — *stub* |
| `feature-ref-syntax` | error | Feature references use valid path syntax — *stub* |
| `internal-links` | error | Internal markdown links resolve — *stub* |
| `forward-refs` | warning | No references to non-existent features — *stub* |
| `code-annotations` | warning | Go files have feature annotation comments — *stub* |

Rules marked *stub* are registered but not yet implemented (they pass without violations).

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Spec tree is valid (no violations at or above severity threshold) | Proceed with confidence |
| `1` | Violations found at or above severity threshold | Review the violations and fix them |
| `2` | Invalid arguments (unknown rule name, invalid severity, `--rules` and `--ignore` both specified) | Check parameter values |
| `10+` | Unexpected error (I/O failure, internal error) | Log the error and escalate |

## Examples

### Basic lint (errors only)

```bash
synchestra spec lint
# 0 violations found
```

### Include warnings

```bash
synchestra spec lint --severity warning
# features/sandbox/README.md:42 [warning] oq-not-empty: Outstanding Questions section appears empty
#
# 1 violations found (1 warning)
```

### Check specific rules only

```bash
synchestra spec lint --rules readme-exists,oq-section
# 0 violations found
```

### Ignore specific rules

```bash
synchestra spec lint --ignore code-annotations --severity warning
```

### JSON output for programmatic parsing

```bash
synchestra spec lint --format json --severity warning
# [
#   {
#     "file": "features/sandbox/README.md",
#     "line": 42,
#     "severity": "warning",
#     "rule": "oq-not-empty",
#     "message": "Outstanding Questions section appears empty"
#   }
# ]
```

### After creating a new feature

```bash
synchestra feature new cli/spec/search --title "spec search command"
synchestra spec lint  # verify spec tree is still clean
```

## Notes

- This is a **read-only** command — it never mutates the spec tree or repository.
- `--rules` and `--ignore` are mutually exclusive. Use one or the other.
- Default severity is `error` — warnings and info are hidden unless `--severity warning` or `--severity info` is passed.
- The integration test verifies the Synchestra spec repo itself produces zero errors.
- For querying spec content (rather than validating structure), use [`spec search`](../synchestra-spec-search/README.md).
