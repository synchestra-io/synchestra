# Command Group: `synchestra spec`

**Parent:** [CLI](../README.md)

Commands for validating and searching Synchestra specification repositories. The `spec` command group provides tools to ensure specification tree integrity, validate structural conventions, and query spec content.

All `spec` commands are **read-only** — they scan specification files but do not mutate the specification or repository.

## Design Principles

Specification commands treat spec repositories as queryable data structures, not as opaque text. Commands validate against Synchestra's specification conventions (README.md files, Outstanding Questions sections, heading structure, feature references, etc.) to catch drift early and enable safe mutation commands (Phase 2).

## Commands

### Validate

| Command | Description | Skill |
|---|---|---|
| [lint](lint/README.md) | Validate spec tree for structural convention violations | [synchestra-spec-lint](../../../../ai-plugin/skills/synchestra-spec-lint/README.md) |

### Search

| Command | Description | Skill |
|---|---|---|
| search | Keyword search across spec documents (future) | (future) |

## Outstanding Questions

- Should `spec lint` support a `--fix` flag for auto-fixing certain violations (e.g., adding missing OQ sections)?
- Should there be a `--watch` mode for continuous linting during spec editing?
- Should `spec search` support semantic search in addition to keyword matching, or stick to keyword-only for phase 1?
