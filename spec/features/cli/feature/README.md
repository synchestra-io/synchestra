# Command Group: `synchestra feature`

**Parent:** [CLI](../README.md)

Commands for querying and managing features — listing, visualizing hierarchy, tracing dependency and reference relationships, and scaffolding new features. The query commands form an [LSP-like](https://microsoft.github.io/language-server-protocol/) semantic layer for specifications: `info` maps to document symbols, `deps`/`refs` map to go-to-definition/find-references, `tree` maps to type hierarchy, and `--fields`/`--transitive` enrich the output like inlay hints and call hierarchy. See the [skills README](../../../../ai-plugin/skills/README.md#an-lsp-for-specifications) for the full analogy.

Query commands (`info`, `list`, `tree`, `deps`, `refs`) are **read-only** — they pull the latest state from the spec repository but do not mutate anything. Mutation commands (`new`) scaffold files and optionally commit and push.

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

### Query

| Command | Description | Skill |
|---|---|---|
| [info](info/README.md) | Show feature metadata and section TOC with line ranges | [synchestra-feature-info](../../../../ai-plugin/skills/synchestra-feature-info/README.md) |
| [list](list/README.md) | Flat list of all feature IDs — grep/pipe-friendly, full paths | [synchestra-feature-list](../../../../ai-plugin/skills/synchestra-feature-list/README.md) |
| [tree](tree/README.md) | Indented hierarchy showing parent-child nesting; supports focus on a single feature with direction | [synchestra-feature-tree](../../../../ai-plugin/skills/synchestra-feature-tree/README.md) |
| [deps](deps/README.md) | Show features that a given feature depends on | [synchestra-feature-deps](../../../../ai-plugin/skills/synchestra-feature-deps/README.md) |
| [refs](refs/README.md) | Show features that reference a given feature | [synchestra-feature-refs](../../../../ai-plugin/skills/synchestra-feature-refs/README.md) |

### Mutation

| Command | Description | Skill |
|---|---|---|
| [new](new/README.md) | Scaffold a new feature directory with README template | — |

## Shared Arguments

Arguments available across multiple `feature` subcommands. See [`_args/`](_args/README.md) for details.

| Argument | Supported by | Description |
|---|---|---|
| [`--fields`](_args/fields.md) | list, tree, deps, refs | Inline selected metadata fields next to each feature |
| [`--transitive`](_args/transitive.md) | deps, refs | Follow dependency/reference chains to full depth |

## Outstanding Questions

- Should features support metadata beyond the `## Dependencies` section (e.g., status, owner, tags) in a machine-readable format like YAML frontmatter? *(Partially addressed: `feature info` extracts metadata from README structure; YAML frontmatter is not yet required but remains an option.)*
- Should Synchestra expose a proper LSP server for specification files, reusing the same Go packages that power these commands? This would give IDE users live diagnostics, hover info, autocomplete for feature IDs, and rename refactoring. See [LSP feature](../lsp/README.md) for the full proposal and [skills README](../../../../ai-plugin/skills/README.md#an-lsp-for-specifications) for the LSP analogy. *(Tracked: dedicated [LSP feature spec](../lsp/README.md) created.)*
