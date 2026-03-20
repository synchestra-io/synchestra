# Command Group: `synchestra code`

**Parent:** [CLI](../README.md)

Commands for querying source code relationships to Synchestra resources. Where `synchestra feature` operates on the specification graph (feature → feature dependencies), `synchestra code` operates on the code → specification graph (source files → features, plans, docs they reference).

The primary data source is [source references](../../source-references/README.md) — `synchestra:` annotations and their expanded `https://synchestra.io/` URLs embedded in source file comments. All `code` commands scan source files for these references using the [comment-prefix detection rule](../../source-references/README.md#detection-strategy).

All `code` commands are **read-only** — they scan the working tree and optionally pull the spec repository for validation, but do not mutate anything.

## Relationship to `synchestra feature`

The `code` and `feature` command groups are complementary views of the same dependency data:

| Direction | Command | Question answered |
|---|---|---|
| Code → Spec | [`code deps`](deps/README.md) | "What specs does this code depend on?" |
| Spec → Spec | [`feature deps`](../feature/deps/README.md) | "What features does this feature depend on?" |
| Spec → Code | [`feature refs`](../feature/refs/README.md) | "What code references this feature?" |

`code deps` and `feature refs` are inverse operations over the same source-reference data.

## Commands

### Query

| Command | Description |
|---|---|
| [deps](deps/README.md) | Show Synchestra resources that source files depend on |

## Outstanding Questions

- Should there be a `code list` command that lists all source files containing Synchestra references?
- Should there be a `code refs` command (reverse of `code deps`) — given a source file, show what other source files reference the same resources?
