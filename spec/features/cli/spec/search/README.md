# Command: `synchestra spec search`

**Parent:** [spec](../README.md)

## Synopsis

```
synchestra spec search <query> [--feature <path>] [--section <name>] [--status <status>]
                                [--type <type>] [--context <n>] [--refs]
                                [--format <format>] [PATH]
```

## Description

Searches Synchestra specification documents for a query string with spec-aware scoping, metadata enrichment, and optional cross-reference context. Unlike plain `grep`, `spec search` understands the specification hierarchy — it knows which feature a match belongs to, what status that feature has, what section the match falls in, and how the containing feature relates to others in the dependency graph.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<query>` | Yes | Search term or quoted phrase. Quoted strings match exactly; unquoted terms match individually (all must appear in the same file) |
| [`--feature`](_args/feature.md) | No | Scope search to a feature subtree (e.g., `cli/task`) |
| [`--section`](_args/section.md) | No | Restrict search to named sections (e.g., `Outstanding Questions`) |
| [`--status`](_args/status.md) | No | Include only files belonging to features with the given status |
| [`--type`](_args/type.md) | No | Filter by resource type: `feature`, `plan`, `proposal` |
| [`--context`](_args/context.md) | No | Lines of context around each match (default: `2`) |
| [`--refs`](_args/refs.md) | No | Enrich results with dependency and reverse-dependency context |
| [`--format`](../../_args/format.md) | No | Output format: `text` (default), `yaml`, `json` |
| `PATH` | No | Root directory to search (default: `spec/`) |

## Behavior

### 1. Resolve search scope

Determine which directory tree to search:

- **Default:** the `spec/` directory of the current project.
- **`PATH` (positional):** overrides the default root.
- **`--feature <path>`:** narrows scope to `spec/features/<path>/` and all its descendants. If the feature path does not exist, exit with code 3.

`--feature` and `PATH` are mutually exclusive. If both are given, exit with code 2.

### 2. Discover and classify files

Find all `.md` files under the search scope. For each file, determine:

- **Resource type** — `feature` (under `spec/features/`), `plan` (under `spec/plans/`), `proposal` (under a `proposals/` directory within a feature), or `other`.
- **Feature path** — the feature this file belongs to (e.g., `cli/task/claim`), derived from the directory structure. Plans and proposals are associated with their referenced or parent feature.
- **Feature status** — extracted from the `**Status:**` line in the nearest feature README.

### 3. Apply pre-filters

Before text search, narrow the file set:

- **`--type`:** keep only files matching the specified resource type.
- **`--status`:** keep only files whose containing feature has the specified status.

Pre-filtering avoids searching files that will be discarded, improving performance on large spec trees.

### 4. Search within files

For each remaining file:

- If `--section <name>` is set, parse markdown headings to identify section boundaries, then search only within the named section(s). Section matching is case-insensitive and supports partial match (e.g., `--section oq` matches "Outstanding Questions").
- Search for `<query>`:
  - **Quoted phrase** (`"claiming protocol"`): match the exact string, case-insensitive.
  - **Unquoted terms** (`claiming protocol`): match files where all terms appear (AND semantics), each term matched case-insensitive. Individual term positions are independent.

### 5. Build results

For each match, collect:

| Field | Always | Description |
|---|---|---|
| `file` | ✓ | Path relative to project root |
| `line` | ✓ | Line number of the match |
| `context` | ✓ | Surrounding lines (controlled by `--context`) |
| `feature` | ✓ | Feature path the file belongs to (empty for non-feature files) |
| `status` | ✓ | Feature status (empty for non-feature files) |
| `type` | ✓ | Resource type (`feature`, `plan`, `proposal`, `other`) |
| `section` | ✓ | Heading of the section containing the match |
| `deps` | `--refs` | Direct dependencies of the containing feature |
| `reverse_deps` | `--refs` | Features that depend on the containing feature |
| `plans` | `--refs` | Plans referencing the containing feature |

### 6. Sort and output

Results are sorted by relevance:

1. **Heading match** — query appears in a markdown heading (strongest signal of topical relevance).
2. **Exact phrase match** — the full query string appears contiguously in the line.
3. **All-terms match** — all query terms present but not contiguous.

Within each relevance tier, results are ordered by file path (alphabetical) then line number (ascending).

When multiple matches fall in the same feature, they are grouped under that feature in YAML/JSON output to reduce redundancy.

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success (matches found) |
| `1` | No matches found |
| `2` | Invalid arguments (`--feature` and `PATH` both given, unknown `--type`, etc.) |
| `3` | Feature not found (the path given to `--feature` does not exist) |
| `10+` | Unexpected error (I/O failure, etc.) |

Exit code 1 for "no matches" follows `grep` convention and lets scripts branch on match presence without parsing output.

## Output Format

### Text (default)

Human-readable output with file location, metadata badge, and context lines. The matching line is marked with `>`.

```
spec/features/cli/task/claim/README.md:18  [feature: cli/task/claim | status: Conceptual]  § Claiming Protocol
  16: The board checks for concurrent claims using optimistic locking.
  17: If another agent already holds the claim, the request is rejected.
> 18: If no conflict, the claim is recorded and the agent becomes the owner.
  19: The agent must start work within the configured timeout.
  20:

spec/features/agent-skills/README.md:29  [feature: agent-skills | status: In Progress]  § Design Principles
  27: ### Skills wrap the CLI
  28:
> 29: Skills are not an alternative to the CLI — they wrap it. The skill provides the agent with:
  30: - **When to use it** — trigger conditions
  31: - **What to run** — the exact CLI command with parameter descriptions

2 matches in 2 features
```

### YAML

Structured output for agent consumption:

```yaml
query: "claim"
scope: spec/
filters:
  type: null
  status: null
  section: null
matches:
  - file: spec/features/cli/task/claim/README.md
    line: 18
    feature: cli/task/claim
    status: Conceptual
    type: feature
    section: Claiming Protocol
    context:
      - "16: The board checks for concurrent claims using optimistic locking."
      - "17: If another agent already holds the claim, the request is rejected."
      - "18: If no conflict, the claim is recorded and the agent becomes the owner."
      - "19: The agent must start work within the configured timeout."
  - file: spec/features/agent-skills/README.md
    line: 29
    feature: agent-skills
    status: In Progress
    type: feature
    section: Design Principles
    context:
      - "27: ### Skills wrap the CLI"
      - "28:"
      - "29: Skills are not an alternative to the CLI — they wrap it."
      - "30: - **When to use it** — trigger conditions"
      - "31: - **What to run** — the exact CLI command with parameter descriptions"
summary:
  total_matches: 2
  features_matched: 2
```

### YAML with `--refs`

When `--refs` is specified, each match includes cross-reference context:

```yaml
matches:
  - file: spec/features/cli/task/claim/README.md
    line: 18
    feature: cli/task/claim
    status: Conceptual
    type: feature
    section: Claiming Protocol
    context:
      - "18: If no conflict, the claim is recorded and the agent becomes the owner."
    refs:
      deps:
        - task-status-board
        - state-store
      reverse_deps:
        - agent-skills
      plans:
        - agent-skills-roadmap
```

## Examples

```bash
# Basic keyword search across all specs
synchestra spec search "optimistic locking"

# Search within a specific feature subtree
synchestra spec search "exit code" --feature cli/task

# Search only Outstanding Questions sections
synchestra spec search "versioning" --section "Outstanding Questions"

# Find all in-progress features mentioning "token"
synchestra spec search token --status "In Progress" --type feature

# Search plans only, with dependency context
synchestra spec search "phase 2" --type plan --refs

# YAML output for agent consumption
synchestra spec search "claiming" --format yaml

# Increase context to ±5 lines
synchestra spec search "conflict resolution" --context 5

# JSON output piped to jq for filtering
synchestra spec search "mutation" --format json | jq '.matches[] | select(.status == "Conceptual")'

# Search a non-default spec directory
synchestra spec search "README" /path/to/other/spec
```

## Design Rationale

### Why not just `grep`?

Raw text search tools find matches but lose specification context. An agent grepping for "claiming" gets file paths and lines but must separately:

1. Determine which feature the file belongs to
2. Look up that feature's status
3. Identify the section context (is this an OQ? A design decision?)
4. Chase dependencies to understand the blast radius

`spec search` collapses these steps into a single call, reducing agent round-trips from 4–5 to 1.

### Exit code 1 for no matches

Following `grep` convention, exit code 1 signals "no matches" rather than Synchestra's typical "conflict" meaning. For a search command, knowing "zero results" at the exit-code level is more useful than parsing output, and "conflict" has no meaning for a read-only search.

### AND semantics for unquoted terms

Unquoted multi-word queries use AND semantics (all terms must appear in the same file) rather than OR. This matches the intuition of "search for X and Y" and avoids flooding results with single-term matches. Agents that want OR behavior can make multiple `spec search` calls.

### Section name partial matching

`--section oq` matching "Outstanding Questions" reduces typing and avoids forcing agents to remember exact section names. The match is case-insensitive and checks whether the query is a substring of the heading text.

## Acceptance Criteria

- [ ] Searches all `.md` files under `spec/` by default
- [ ] Supports quoted exact-phrase and unquoted AND-terms search
- [ ] `--feature` scopes search to a feature subtree
- [ ] `--section` restricts search to named markdown sections (case-insensitive partial match)
- [ ] `--status` filters results by containing feature's status
- [ ] `--type` filters by resource type (`feature`, `plan`, `proposal`)
- [ ] `--context` controls surrounding lines (default: 2)
- [ ] `--refs` enriches results with deps, reverse deps, and plans
- [ ] Results are sorted by relevance (heading > exact phrase > all-terms)
- [ ] YAML and JSON output group matches by feature
- [ ] Text output shows metadata badge and section context
- [ ] Exit code 1 when no matches found, 0 when matches found
- [ ] `--feature` and `PATH` are mutually exclusive (exit code 2 if both)
- [ ] Feature path validation returns exit code 3 for non-existent paths
- [ ] Performance: completes in under 2 seconds on the Synchestra spec repo

## Outstanding Questions

- Should `spec search` support regular expressions in addition to literal strings (e.g., `--regex` flag)?
- Should results be deduplicated when the same line matches multiple terms?
- Should `--section` accept multiple section names (comma-separated or repeated flag)?
- How should matches in non-feature files (e.g., top-level `spec/README.md`) be presented when `--refs` is used?
- Should there be a `--limit <n>` flag to cap result count, or is the expectation that spec trees are small enough to return all matches?
