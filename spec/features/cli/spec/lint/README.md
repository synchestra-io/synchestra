# Command: `synchestra spec lint`

**Parent:** [spec](../README.md)

Validates a Synchestra specification repository for structural convention violations. Checks that all directories have README.md files, required sections (Outstanding Questions) are present, heading levels are consistent, feature references are valid, internal links resolve, and specification indices are up-to-date.

## Behavior

`spec lint` scans the specification tree and reports all violations in a single pass (does not fail-fast). Violations are categorized by severity: **error** (must fix), **warning** (should fix), **info** (advisory). The command exits with code 0 when no violations are found, code 1 when violations are found, and code 10+ for unexpected errors.

By default, only **error**-level violations are reported. Use `--severity warning` to include warnings, or `--severity info` to include all diagnostic information.

### Linting Rules

#### Error-Level Rules

| Rule | Scope | Check |
|---|---|---|
| `readme-exists` | All spec directories | Every spec subdirectory must have a `README.md` file (exception: `.github/` directory itself; subdirectories like `.github/workflows/` must still have README.md) |
| `oq-section` | Feature/plan README files | Every feature and plan README must have a `## Outstanding Questions` section |
| `feature-ref-syntax` | All markdown files | Feature references must use valid path syntax: `spec/features/cli/task`, `../cli/task`, or `[link](../cli/task/README.md)` |
| `internal-links` | All markdown files | Links to README.md files and anchor references must resolve to existing files and headings |
| `index-entries` | Feature README files | If a README includes a child index/contents section, every actual child directory must be listed, and no non-existent directories should be referenced |

#### Warning-Level Rules

| Rule | Scope | Check |
|---|---|---|
| `oq-not-empty` | Feature/plan README files | Outstanding Questions section must either list questions or explicitly state "None at this time." (not blank) |
| `heading-levels` | All markdown files | Heading structure must not skip levels (e.g., H2 → H4 is invalid; must be H2 → H3 → H4) |
| `forward-refs` | Feature/plan README files | No references to features/plans that don't exist yet (e.g., `sandbox/vm` if that directory is not present) |
| `code-annotations` | Go source files | Integration points should include `// Features implemented:` and `// Features depended on:` annotations (per [source-references](../../source-references/README.md) convention) |

#### Info-Level Rules

| Rule | Scope | Check |
|---|---|---|
| (reserved for future diagnostics) | | |

### Defaults and Customization

- **Default scope:** `./spec` (scans entire specification tree)
- **Default severity threshold:** `error` (warnings and info not reported)
- **Default rule set:** all rules enabled

Use flags to customize:
- `--rules`: Enable only specified rules (comma-separated: `readme-exists,oq-section`)
- `--ignore`: Disable specified rules (comma-separated: `forward-refs,code-annotations`)
- `--severity`: Report violations of this level or higher (`error`, `warning`, `info`; default: `error`)

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Specification is valid (no violations at or above severity threshold) |
| `1` | Violations found at or above severity threshold |
| `2` | Invalid arguments (unknown rule, invalid severity level, etc.) |
| `10+` | Unexpected error (I/O, internal panic, etc.) |

### Output Format

By default, `spec lint` produces human-readable output:

```
spec/features/cli/task/README.md:1 [error] readme-exists: README.md not found in directory
spec/features/agent-skills/README.md:50 [warning] oq-section: OQ section present but appears empty
spec/features/cli/README.md:12 [error] heading-levels: Invalid heading jump from H2 to H4
spec/features/cli/README.md:45 [error] feature-ref-syntax: Invalid feature reference "task/invalid/path"
spec/features/state-sync/README.md:20 [error] internal-links: Broken link to "../missing-feature/README.md"
spec/features/sandbox/README.md:15 [warning] forward-refs: References feature "sandbox/vm" which does not exist

6 violations found (5 errors, 1 warning)
```

Use `--format json` for machine-parseable output:

```json
[
  {
    "file": "spec/features/cli/task/README.md",
    "line": 1,
    "severity": "error",
    "rule": "readme-exists",
    "message": "README.md not found in directory"
  },
  {
    "file": "spec/features/agent-skills/README.md",
    "line": 50,
    "severity": "warning",
    "rule": "oq-section",
    "message": "OQ section present but appears empty"
  }
]
```

### Examples

```bash
# Lint ./spec with default settings (errors only)
$ synchestra spec lint
0 violations found
$ echo $?
0

# Lint with warnings included
$ synchestra spec lint --severity warning
5 violations found (3 errors, 2 warnings)
$ echo $?
1

# Check only specific rules
$ synchestra spec lint --rules readme-exists,oq-section
2 violations found (2 errors)

# Ignore code-annotations (common in shared repos)
$ synchestra spec lint --ignore code-annotations

# Output as JSON for programmatic parsing
$ synchestra spec lint --format json | jq '.[] | select(.severity == "error")'

# Lint a non-default spec directory
$ synchestra spec lint /path/to/custom/spec
```

## Acceptance Criteria

- [x] Command signature and behavior defined
- [ ] All error and warning rules implemented
- [ ] Reuses parsing from `feature info`/`deps`/`refs` where possible
- [ ] Reports all violations in single pass (no fail-fast)
- [ ] Running against Synchestra spec repo produces zero errors (or documented exceptions listed)
- [ ] Comprehensive unit tests for each rule with fixture specs
- [ ] Integration test passes
- [ ] Wrapped as `synchestra-spec-lint` agent skill

## Outstanding Questions

- Should certain directories (e.g., `spec/proposals/`, `spec/archived/`) be excluded or have relaxed rules?
- Should `spec lint` validate that feature README files have specific required sections beyond OQ (e.g., "Summary", "Design Principles")?
- Should code annotations checking be scoped to specific directories (e.g., only `pkg/cli/`) or all `.go` files?
