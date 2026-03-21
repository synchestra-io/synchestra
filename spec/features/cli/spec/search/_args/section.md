# --section

Restricts the search to text within named markdown sections.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | (none — searches entire file) |

## Syntax

```bash
synchestra spec search <query> --section <section-name>
```

## Behavior

Section matching is **case-insensitive** and supports **partial match** — the provided name is checked as a substring of each markdown heading. For example:

| `--section` value | Matches headings |
|---|---|
| `Outstanding Questions` | `## Outstanding Questions` |
| `oq` | `## Outstanding Questions` |
| `accept` | `## Acceptance Criteria` |
| `behavior` | `## Behavior`, `## Behaviour`, `### Claiming Behavior` |

A section spans from its heading to the next heading of equal or higher level (or end of file). All content within matching sections is searched; content outside matching sections is skipped.

If no sections in a file match the `--section` value, that file produces no results (it is silently skipped, not an error).

## Examples

```bash
# Search only Outstanding Questions sections
synchestra spec search "versioning" --section "Outstanding Questions"

# Short form — partial match
synchestra spec search "versioning" --section oq

# Search within Acceptance Criteria
synchestra spec search "atomic" --section "Acceptance Criteria"

# Combine with feature scoping
synchestra spec search "timeout" --feature cli/task --section behavior
```

## Outstanding Questions

- Should `--section` accept multiple names (comma-separated or repeated flag) to search across several sections at once?
