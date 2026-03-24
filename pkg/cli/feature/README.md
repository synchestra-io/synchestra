# feature

Implements the `synchestra feature` command group — querying and scaffolding features in a spec repository.

## Commands

| Command | Type | Description |
|---|---|---|
| `info` | Query | Show feature metadata, section TOC with line ranges, and children |
| `list` | Query | List all feature IDs, one per line |
| `tree` | Query | Display feature hierarchy as an indented tree |
| `deps` | Query | Show dependencies of a feature |
| `refs` | Query | Show reverse references (features that depend on a given feature) |
| `new` | Mutation | Scaffold a new feature directory with a README template |

## Key Files

- `discover.go` — Spec repo discovery, feature walking, dependency parsing, and shared helpers (`featureExists`, `parseDependencies`)
- `info.go` — `feature info` command with YAML/JSON/text output and section parsing
- `list.go` — `feature list` command
- `tree.go` — `feature tree` command with hierarchy building
- `deps.go` — `feature deps` command
- `refs.go` — `feature refs` command
- `new.go` — `feature new` command: validation, slug generation, directory scaffolding, parent/index updates, git operations
- `slug.go` — Slug generation and validation algorithms
- `template.go` — README template generation and status validation

## Outstanding Questions

None at this time.
