---
name: synchestra-spec-search
description: Searches Synchestra specification documents with spec-aware scoping, metadata filtering, and cross-reference context. Use when finding spec content by keyword, searching within specific sections or feature subtrees, or understanding how a match connects to the broader feature graph.
---

# Skill: synchestra-spec-search

Search specification documents for keywords with spec-aware scoping — results include feature metadata, section context, and optional dependency information.

**CLI reference:** [synchestra spec search](../../spec/features/cli/spec/search/README.md)

## When to use

- **Finding relevant specs:** Search for a concept or term across all spec documents instead of manually browsing the feature tree
- **Scoping to a subtree:** Use `--feature` to search within a specific feature and its children (e.g., `--feature cli/task` to search all task-related specs)
- **Searching specific sections:** Use `--section` to restrict search to Outstanding Questions, Acceptance Criteria, or other sections (e.g., `--section oq` to find open questions mentioning a term)
- **Filtering by status:** Use `--status` to find matches only in features at a specific lifecycle stage (e.g., `--status "In Progress"` to search active features)
- **Understanding context:** Use `--refs` to see how matching features relate to the dependency graph without separate `feature deps`/`feature refs` calls
- **Narrowing by resource type:** Use `--type` to search only features, plans, or proposals

## Command

```bash
synchestra spec search <query> \
  [--feature <path>] \
  [--section <name>] \
  [--status <status>] \
  [--type <type>] \
  [--context <n>] \
  [--refs] \
  [--format <format>] \
  [PATH]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<query>` | Yes | Search term or quoted phrase. Quoted strings match exactly; unquoted terms use AND semantics (all must appear in the same file) |
| [`--feature`](../../spec/features/cli/spec/search/_args/feature.md) | No | Scope to a feature subtree (e.g., `cli/task`). Mutually exclusive with `PATH` |
| [`--section`](../../spec/features/cli/spec/search/_args/section.md) | No | Restrict to named sections. Case-insensitive partial match (e.g., `oq` matches "Outstanding Questions") |
| [`--status`](../../spec/features/cli/spec/search/_args/status.md) | No | Include only features with this status (e.g., `Conceptual`, `In Progress`, `Implemented`) |
| [`--type`](../../spec/features/cli/spec/search/_args/type.md) | No | Filter by resource type: `feature`, `plan`, `proposal` |
| [`--context`](../../spec/features/cli/spec/search/_args/context.md) | No | Lines of context around each match (default: `2`) |
| [`--refs`](../../spec/features/cli/spec/search/_args/refs.md) | No | Enrich results with dependency, reverse-dependency, and plan context |
| `--format` | No | Output format: `text` (default), `yaml`, `json` |
| `PATH` | No | Root directory to search (default: `spec/`). Mutually exclusive with `--feature` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Matches found | Parse the results |
| `1` | No matches found | Broaden search — try fewer filters, wider scope, or different terms |
| `2` | Invalid arguments (`--feature` and `PATH` both given, unknown `--type`, etc.) | Check parameter values |
| `3` | Feature not found (the `--feature` path does not exist) | Verify the feature path with `feature list` |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Basic keyword search

```bash
synchestra spec search "optimistic locking"
# spec/features/task-status-board/README.md:42  [feature: task-status-board | status: In Progress]  § Claiming Protocol
#   40: The board uses git as the persistence layer.
#   41: Concurrent claims are handled with an optimistic locking strategy:
# > 42: each agent reads the current state, applies its change, and pushes.
#   43: If another agent pushed first, the push is rejected and the agent retries.
#   44:
#
# 1 match in 1 feature
```

### Search within a feature subtree

```bash
synchestra spec search "exit code" --feature cli/task
# spec/features/cli/task/claim/README.md:31  [feature: cli/task/claim | status: Conceptual]  § Exit Codes
# > 31: | `1` | Claim conflict — another agent claimed the task first |
#
# spec/features/cli/task/new/README.md:45  [feature: cli/task/new | status: Implemented]  § Exit Codes
# > 45: | `0` | Task created successfully |
#
# 2 matches in 2 features
```

### Search Outstanding Questions only

```bash
synchestra spec search "versioning" --section oq
# spec/features/agent-skills/README.md:93  [feature: agent-skills | status: In Progress]  § Outstanding Questions
# > 93: - How are skills versioned? Does the CLI version imply the skill version, or are they independent?
#
# 1 match in 1 feature
```

### Filter by status and type

```bash
synchestra spec search "token" --status "In Progress" --type feature --format yaml
# query: "token"
# scope: spec/
# filters:
#   type: feature
#   status: In Progress
# matches:
#   - file: spec/features/agent-skills/README.md
#     line: 20
#     feature: agent-skills
#     status: In Progress
#     type: feature
#     section: Context
#     context:
#       - "19: - **Token cost is 5–10× higher than necessary.** ..."
#       - "20: A full feature README averages ~3,000 tokens."
# summary:
#   total_matches: 1
#   features_matched: 1
```

### Search with cross-reference context

```bash
synchestra spec search "claiming" --refs --format yaml
# matches:
#   - file: spec/features/cli/task/claim/README.md
#     line: 18
#     feature: cli/task/claim
#     status: Conceptual
#     section: Claiming Protocol
#     context:
#       - "18: If no conflict, the claim is recorded and the agent becomes the owner."
#     refs:
#       deps:
#         - task-status-board
#         - state-store
#       reverse_deps:
#         - agent-skills
#       plans:
#         - agent-skills-roadmap
```

### Search plans for roadmap context

```bash
synchestra spec search "phase 2" --type plan
```

## Notes

- This is a **read-only** command — it never mutates the spec tree or repository.
- `--feature` and `PATH` are mutually exclusive. Use `--feature` for spec-aware subtree scoping; use `PATH` for arbitrary directory scoping.
- Unquoted multi-word queries use AND semantics — all terms must appear in the same file. For OR behavior, make separate calls.
- `--section` uses partial, case-insensitive matching: `--section oq` matches "Outstanding Questions", `--section accept` matches "Acceptance Criteria".
- `--refs` adds latency on large spec trees because it computes reverse dependencies. Omit it for quick keyword lookups.
- Results are sorted by relevance: heading matches first, then exact phrases, then all-terms matches.
- For validating spec structure (rather than searching content), use [`spec lint`](../synchestra-spec-lint/README.md).
- For navigating the feature hierarchy, use [`feature tree`](../synchestra-feature-tree/README.md) or [`feature info`](../synchestra-feature-info/README.md).
