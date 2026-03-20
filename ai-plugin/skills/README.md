# Skills

Synchestra skills are focused, self-contained instructions that teach AI agents how to perform specific Synchestra operations. Each skill wraps a single CLI command with clear trigger conditions, parameters, and exit code handling.

See the [agent-skills feature spec](../../spec/features/agent-skills/README.md) for design principles and the full skill format.

## How Skills Transform Agent Workflows

### An LSP for Specifications

What [LSP](https://microsoft.github.io/language-server-protocol/) did for code navigation, Synchestra's `feature` commands do for specification navigation. LSP gives IDEs structured access to code — symbols, definitions, references, diagnostics. Synchestra's CLI gives AI agents (and humans) structured access to specifications — features, dependencies, sections, lifecycle status.

| LSP concept | Synchestra equivalent |
|---|---|
| `textDocument/documentSymbol` | `feature info` (section TOC with line ranges) |
| `textDocument/definition` | `feature deps` (go to dependency) |
| `textDocument/references` | `feature refs` (find all referrers) |
| `workspace/symbol` | `feature list` / `feature tree` |
| Hover | `feature info` metadata (status, oq, children) |
| Call hierarchy | `--transitive` flag (follow chains) |
| Inlay hints | `--fields` flag (inline metadata) |
| Type hierarchy | `feature tree --direction up\|down` |

The difference: LSP serves IDEs via a persistent server and binary protocol. Synchestra serves AI agents via a stateless CLI and YAML output. But the semantic layer is the same — structured navigation over a domain-specific document tree.

### The Problem: Specification Navigation is Expensive

When AI agents work on spec repos, they glob, view, and grep files one by one. Each file costs tokens. Understanding "what exists" across 24+ features can consume 10,000+ tokens before the agent does any real work.

Concrete pain points:

- **Feature discovery** — reading the feature index, then individual READMEs, just to answer "what depends on what?"
- **Dependency traversal** — following `depends-on` links requires recursive file reads, each costing ~500–3,000 tokens
- **Feature creation** — editing 3+ files (README, parent index, feature index), with agents missing steps ~30% of the time
- **Context budget pressure** — agents that spend tokens on navigation have fewer tokens left for reasoning

### How Skills Solve This

- **Token efficiency** — `feature info` returns ~500 tokens of structured metadata vs. reading a 3,000-token README. `feature deps --transitive` resolves full dependency chains in one call instead of recursive reads.
- **Structural safety** — mutation commands (future: `feature new`, `question add`) enforce spec conventions, eliminating missing OQ sections, forgotten index updates, and wrong status values.
- **Progressive discovery** — YAML frontmatter lets agents discover skills by name and description only. Full instructions load on demand.
- **Composable enrichment** — `--fields` and `--transitive` flags let agents request exactly the metadata they need without loading unnecessary content.

### Token Cost Comparison

| Operation | Without skills | With skills | Savings |
|---|---|---|---|
| List all features | ~4,000 tokens (read index) | ~500 tokens | 87% |
| Feature metadata + sections | ~3,000 (read full README) | ~500 (`feature info`) | 83% |
| Transitive deps | ~9,000 (recursive reads) | ~300 (`deps --transitive`) | 97% |
| Full feature context | ~15,000 (multi-file reads) | ~2,000 (`info` + `deps --fields`) | 87% |

## Skill Design Principles

Core principles (see the [agent-skills spec](../../spec/features/agent-skills/README.md) for full details):

- **One skill per CLI command** — no multi-purpose skills. Small, testable, easy to reason about.
- **Skills wrap the CLI, not replace it** — each skill tells the agent *when* to use a command, *what* to run, and *what happens next* (exit codes + follow-up actions).
- **Agent-first output** — YAML by default for structured parsing; `--format text` for human consumption.
- **Composable flags over monolithic context** — `--fields`, `--transitive`, and `--direction` let agents request exactly what they need.
- **Progressive context loading** — metadata first, content on demand. Agents start with cheap overviews and drill down only when needed.

## Available Skills

### Task Management

| Skill | Description | CLI Command |
|---|---|---|
| [synchestra-task-new](synchestra-task-new/README.md) | Create a new task | [task new](../../spec/features/cli/task/new/README.md) |
| [synchestra-task-enqueue](synchestra-task-enqueue/README.md) | Move a task from planning to queued | [task enqueue](../../spec/features/cli/task/enqueue/README.md) |
| [synchestra-claim-task](synchestra-claim-task/README.md) | Claim a task before starting work on it | [task claim](../../spec/features/cli/task/claim/README.md) |
| [synchestra-task-start](synchestra-task-start/README.md) | Begin work on a claimed task | [task start](../../spec/features/cli/task/start/README.md) |
| [synchestra-task-status](synchestra-task-status/README.md) | Query or update task status | [task status](../../spec/features/cli/task/status/README.md) |
| [synchestra-task-complete](synchestra-task-complete/README.md) | Mark a task as completed | [task complete](../../spec/features/cli/task/complete/README.md) |
| [synchestra-task-fail](synchestra-task-fail/README.md) | Mark a task as failed with reason | [task fail](../../spec/features/cli/task/fail/README.md) |
| [synchestra-task-block](synchestra-task-block/README.md) | Mark a task as blocked | [task block](../../spec/features/cli/task/block/README.md) |
| [synchestra-task-unblock](synchestra-task-unblock/README.md) | Resume a blocked task | [task unblock](../../spec/features/cli/task/unblock/README.md) |
| [synchestra-task-release](synchestra-task-release/README.md) | Release a claimed task back to queued | [task release](../../spec/features/cli/task/release/README.md) |
| [synchestra-task-abort](synchestra-task-abort/README.md) | Request abortion of a task | [task abort](../../spec/features/cli/task/abort/README.md) |
| [synchestra-task-aborted](synchestra-task-aborted/README.md) | Report a task has been aborted | [task aborted](../../spec/features/cli/task/aborted/README.md) |
| [synchestra-task-list](synchestra-task-list/README.md) | List tasks with filtering | [task list](../../spec/features/cli/task/list/README.md) |
| [synchestra-task-info](synchestra-task-info/README.md) | Show full task details and context | [task info](../../spec/features/cli/task/info/README.md) |

### Feature Navigation

| Skill | Description | CLI Command |
|---|---|---|
| [synchestra-feature-info](synchestra-feature-info/README.md) | Show feature metadata, section TOC, and children | [feature info](../../spec/features/cli/feature/info/README.md) |
| [synchestra-feature-list](synchestra-feature-list/README.md) | List all features with optional metadata fields | [feature list](../../spec/features/cli/feature/list/README.md) |
| [synchestra-feature-tree](synchestra-feature-tree/README.md) | Display feature hierarchy with focus/direction support | [feature tree](../../spec/features/cli/feature/tree/README.md) |
| [synchestra-feature-deps](synchestra-feature-deps/README.md) | Show dependencies with optional transitive resolution | [feature deps](../../spec/features/cli/feature/deps/README.md) |
| [synchestra-feature-refs](synchestra-feature-refs/README.md) | Show reverse dependencies with optional transitive resolution | [feature refs](../../spec/features/cli/feature/refs/README.md) |

### Code Navigation

| Skill | Description | CLI Command |
|---|---|---|
| [synchestra-code-deps](synchestra-code-deps/README.md) | Show Synchestra resources that source files depend on | [code deps](../../spec/features/cli/code/deps/README.md) |

## Roadmap

**Implemented:** all task lifecycle commands (create through abort), feature list, feature tree, feature deps, feature refs.

**Next up:** feature info, `--fields` flag for selective metadata, `--transitive` for dependency resolution, spec validate, feature new.

See the [Agent Skills Roadmap](../../spec/plans/agent-skills-roadmap/README.md) for the phased plan and competitive analysis.

## Competitive Context

No existing tool combines structured specification management, git-backed multi-agent coordination, and an agent-native CLI interface. GitHub Spec Kit is the closest analog — but it's simpler, flat (no hierarchy), and doesn't support multi-agent workflows. Synchestra's skill layer gives agents a token-efficient, convention-enforcing interface that scales with spec complexity.

## Skill File Format

Every `README.md` inside a skill directory **MUST** begin with a YAML frontmatter header containing `name` and `description` fields. This is required by the [Claude Code skills format](https://code.claude.com/docs/en/skills.md).

```yaml
---
name: synchestra-feature-list
description: Lists all features in a project. Use when listing features, exploring feature structure, or checking what features exist.
---
```

- **`name`** — the skill identifier (must match the directory name).
- **`description`** — a concise, action-oriented sentence describing what the skill does and when to invoke it. Include trigger phrases like "Use when…" so agents can match user intent to the right skill.

The rest of the file follows the standard skill body format (heading, context, parameters, exit codes, etc.).

## Outstanding Questions

- Should Synchestra expose a proper [LSP server](https://microsoft.github.io/language-server-protocol/) for specification files? The CLI already provides the semantic layer (feature info → documentSymbol, deps → definition, refs → references). An LSP adapter would give humans live IDE integration: hover for feature metadata, autocomplete feature IDs in dependency sections, red squiggles for broken cross-references, rename refactoring across all specs. The Go packages powering the CLI could be reused — the incremental cost is the protocol adapter. But the primary audience today is agents (served by CLI), and an LSP would primarily benefit humans editing specs in IDEs. *(Tracked: dedicated [LSP feature spec](../../spec/features/lsp/README.md) created.)*
