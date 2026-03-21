# pkg/cli/spec

Go implementation of the `synchestra spec` command group.

## Commands

| Command | File | Description |
|---------|------|-------------|
| `spec lint` | [lint.go](lint.go) | Validate spec tree for structural convention violations |

## Architecture

The linter uses a pluggable checker architecture:

- **`linter.go`** — Orchestrator that manages checkers and rule filtering
- **`lint.go`** — CLI command wiring, output formatting, severity filtering
- **Checker files** — One file per rule (or group of related rules)

Each checker implements the `checker` interface:

```go
type checker interface {
    check(specRoot string) ([]Violation, error)
    name() string
    severity() string
}
```

## Rules

### Implemented

| Rule | File | Severity |
|------|------|----------|
| `readme-exists` | [readme_exists.go](readme_exists.go) | error |
| `oq-section` | [oq_section.go](oq_section.go) | error |
| `oq-not-empty` | [oq_section.go](oq_section.go) | warning |
| `index-entries` | [index_entries.go](index_entries.go) | error |

### Stub (Phase 2)

| Rule | File | Severity |
|------|------|----------|
| `heading-levels` | [checkers_extended.go](checkers_extended.go) | warning |
| `feature-ref-syntax` | [checkers_extended.go](checkers_extended.go) | error |
| `internal-links` | [checkers_extended.go](checkers_extended.go) | error |
| `forward-refs` | [checkers_extended.go](checkers_extended.go) | warning |
| `code-annotations` | [checkers_extended.go](checkers_extended.go) | warning |

## Outstanding Questions

None at this time.
