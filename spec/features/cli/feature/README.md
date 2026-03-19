# Command Group: `synchestra feature`

**Parent:** [CLI](../README.md)

Commands for querying features — listing, visualizing hierarchy, and tracing dependency and reference relationships.

All commands in this group are **read-only**. They pull the latest state from the spec repository but do not mutate anything.

## Feature IDs

A feature's ID is its path relative to the project's features directory, using `/` as the separator. Examples:

- `cli` — top-level feature
- `cli/task` — nested feature (child of `cli`)
- `micro-tasks` — top-level feature
- `cross-repo-sync` — top-level feature

Feature IDs match the directory structure under `spec/features/` (or the configured specifications directory).

## Dependency Discovery

Feature dependencies are declared in each feature's `README.md` under a structured `## Dependencies` section containing a bullet list of feature IDs:

```markdown
## Dependencies

- claim-and-push
- conflict-resolution
```

Features without a `## Dependencies` section (or with an empty one) are treated as independent — they have no dependencies.

The `deps` command reads this section from the target feature. The `refs` command scans all features' `## Dependencies` sections to find reverse references.

## Commands

| Command | Description | Skill |
|---|---|---|
| [list](list/README.md) | List all feature IDs, one per line | [synchestra-feature-list](../../../../skills/synchestra-feature-list/README.md) |
| [tree](tree/README.md) | Display feature hierarchy as an indented tree | [synchestra-feature-tree](../../../../skills/synchestra-feature-tree/README.md) |
| [deps](deps/README.md) | Show features that a given feature depends on | [synchestra-feature-deps](../../../../skills/synchestra-feature-deps/README.md) |
| [refs](refs/README.md) | Show features that reference a given feature | [synchestra-feature-refs](../../../../skills/synchestra-feature-refs/README.md) |

## Outstanding Questions

- Should features support metadata beyond the `## Dependencies` section (e.g., status, owner, tags) in a machine-readable format like YAML frontmatter?
- Should `deps` and `refs` support transitive dependency tracing (show the full chain, not just direct dependencies)?
