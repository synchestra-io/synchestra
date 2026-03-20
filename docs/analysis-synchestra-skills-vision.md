# Synchestra Skills for AI Agents: Vision & Analysis

> **Author:** AI analysis (Claude Opus 4.6), March 2026
>
> **Purpose:** Analyze Synchestra's spec organization, evaluate existing skills, propose new skills that
> would make AI agents faster and cheaper at navigating/editing specifications, and critically assess
> whether Synchestra brings real advantages over alternatives.

---

## Table of Contents

1. [Current State: Spec Organization](#1-current-state-spec-organization)
2. [Current State: Existing Skills](#2-current-state-existing-skills)
3. [The Gap: What AI Agents Actually Need](#3-the-gap-what-ai-agents-actually-need)
4. [Vision: Skills That Would Transform Agent Productivity](#4-vision-skills-that-would-transform-agent-productivity)
5. [Critical Assessment: Does Synchestra Actually Help?](#5-critical-assessment-does-synchestra-actually-help)
6. [Competitive Landscape](#6-competitive-landscape)
7. [Conclusions & Recommendations](#7-conclusions--recommendations)

---

## 1. Current State: Spec Organization

### Structure

Synchestra specs live under `spec/features/` as a hierarchy of directories, each with a mandatory
`README.md`. The tree currently holds **24 top-level features** with nested sub-features:

```
spec/features/
├── feature/                    # Meta: defines feature structure itself
├── cli/                        # 9 command groups, 30+ subcommands
│   ├── task/ (14 subcommands)
│   ├── feature/ (4 subcommands)
│   ├── config/, project/, server/, serve/, test/, mcp/
│   └── _args/ (global arguments)
├── agent-skills/
├── task-status-board/
├── development-plan/
├── proposals/
├── outstanding-questions/
├── project-definition/
├── state-store/
├── chat/ (with workflow/ sub-features)
├── api/
├── ui/ (web-app/, tui/)
├── bots/ (synchestra-bot/)
├── acceptance-criteria/
├── testing-framework/
├── [12 more features...]
└── README.md                   # Master index with status, descriptions, dependency graph
```

### Key Properties

| Property | How it works |
|---|---|
| **Feature identification** | Path-based: `cli/task/claim`, `chat/workflow` |
| **Feature lifecycle** | Conceptual → In Progress → Stable → Deprecated |
| **Dependencies** | Declared in `## Dependencies` section of each README |
| **Outstanding Questions** | Mandatory section in every README, with full lifecycle |
| **Reserved prefixes** | `_acs/`, `_args/`, `_tests/` — not sub-features |
| **Index** | `spec/features/README.md` maintains table + summaries + dependency graph |

### What's Actually Implemented (Go CLI)

Only a subset of the spec is implemented in code:

| Implemented | Spec-Only |
|---|---|
| `feature list/tree/deps/refs` | `feature create/info/edit/validate` (not specified) |
| `project new` | `plan create/submit/approve` (not specified) |
| `test run/list` | `proposal create/submit` (not specified) |
| State store interfaces | `question add/link` (proposed in OQ spec) |
| Git operations layer | All `task` commands (specified but not yet in Go) |

---

## 2. Current State: Existing Skills

### Inventory (19 skills, all in `ai-plugin/skills/`)

**Task lifecycle (14 skills):**
`synchestra-task-create`, `synchestra-task-enqueue`, `synchestra-claim-task`,
`synchestra-task-start`, `synchestra-task-status`, `synchestra-task-complete`,
`synchestra-task-fail`, `synchestra-task-block`, `synchestra-task-unblock`,
`synchestra-task-release`, `synchestra-task-abort`, `synchestra-task-aborted`,
`synchestra-task-list`, `synchestra-task-info`

**Feature queries (4 skills):**
`synchestra-feature-list`, `synchestra-feature-tree`,
`synchestra-feature-deps`, `synchestra-feature-refs`

### Coverage Analysis

```
Task management:  ████████████████████ 14/14 specified commands  (100%)
Feature queries:  ████████████████████  4/4  specified commands  (100%)
Feature mutation:                       0    (no CLI commands exist)
Plan management:                        0    (no CLI commands exist)
Proposal mgmt:                          0    (no CLI commands exist)
OQ management:                          0    (no CLI commands exist)
Spec navigation:                        0    (no concept exists)
Spec editing:                           0    (no concept exists)
Context loading:                        0    (no concept exists)
```

### Skill Format Strengths

Each skill follows a clean, consistent structure:
- YAML frontmatter (`name`, `description`) for agent discovery
- "When to use" trigger conditions
- Exact CLI command with parameters
- Exit code table with agent-actionable guidance
- Concrete examples

This format is **well-designed for agent consumption** — it's progressive (metadata loads cheap,
full content on demand) and actionable (agents know exactly what to do on each exit code).

---

## 3. The Gap: What AI Agents Actually Need

### The Real Problem: Specification Navigation is Expensive

When I (an AI agent) work on this repository, here's what actually happens:

1. **Finding relevant specs:** I `glob` for directories, `view` README files one by one, `grep` for
   keywords. Each file costs tokens. For 24 top-level features with nested sub-features, understanding
   "what exists" alone can consume 10,000+ tokens.

2. **Understanding dependencies:** I read a feature README, find its `## Dependencies` section, then
   manually open each referenced feature. Transitive deps require recursive exploration.

3. **Checking consistency:** To verify a cross-reference, I read both documents. To check if an
   outstanding question was answered, I need to find the linked task, check its status, and read its
   output.

4. **Editing specs:** Creating a new feature means: create directory, create README with correct
   structure (title, status, summary, problem, behavior, OQ section), update parent feature's Contents
   table, update `spec/features/README.md` index, update dependency graph. Miss any step and the
   spec is inconsistent.

5. **Context loading for a task:** Before working on a task, I need to understand the feature it
   implements. That means loading the feature spec, its parent feature, its dependencies, the
   development plan, and any relevant proposals. This is a **manual multi-file read every time**.

### Token Cost Analysis (Typical Operations)

| Operation | Current approach | Token cost | With CLI/skill | Projected cost |
|---|---|---|---|---|
| List all features | `view` spec/features/README.md | ~4,000 | `synchestra feature list` | ~500 |
| Show feature tree | `view` + recursive `glob` | ~6,000 | `synchestra feature tree` | ~800 |
| Get feature deps | Read README, parse section | ~3,000 | `synchestra feature deps` | ~300 |
| **Get feature context** | Read feature + deps + plan + proposals | **~15,000** | *(nothing exists)* | — |
| **Validate spec structure** | Manual inspection of all conventions | **~20,000** | *(nothing exists)* | — |
| **Create new feature** | Multi-file creation + index updates | **~8,000** | *(nothing exists)* | — |
| **Find related specs** | grep + manual reading | **~10,000** | *(nothing exists)* | — |
| **Load task context** | Read task + linked feature + deps | **~12,000** | *(nothing exists)* | — |

The existing skills (feature list/tree/deps/refs) cover the **cheapest** operations.
The **expensive** operations — the ones that actually burn tokens — have no tooling.

---

## 4. Vision: Skills That Would Transform Agent Productivity

### Tier 1: High-Impact Spec Navigation Skills

These wrap new CLI commands that don't exist yet but would deliver the highest agent productivity gains.

#### `synchestra-feature-show` → `synchestra feature show <feature-path>`

**What it does:** Returns a structured, token-efficient summary of a single feature without requiring
the agent to read the full README.

**Output:**
```yaml
path: cli/task/claim
status: Conceptual
summary: "Claim a task before starting work, using atomic git push for optimistic locking"
dependencies: [task-status-board, state-store]
dependents: [agent-skills]
outstanding_questions: 3
proposals: 0
plans: [e2e-testing-framework]
children: []
word_count: 1,247
```

**Why it matters:** An agent can understand what a feature *is* without reading 1,000+ words.
Saves ~2,500 tokens per feature inspection.

#### `synchestra-feature-context` → `synchestra feature context <feature-path>`

**What it does:** Loads the minimum context an agent needs to work on a feature — the feature spec
itself, its direct dependencies' summaries, linked plans, and active proposals — as a single
structured output.

**Output:** Concatenated, minimal representation of:
- The feature's full README
- One-paragraph summaries of each dependency
- Linked development plan steps (if any)
- Active proposals (titles + statuses only)
- Open outstanding questions

**Why it matters:** This is the **single most expensive recurring operation** for agents. Every time
an agent starts working on a feature, it manually assembles this context. A single command could cut
context-loading from ~15,000 tokens to ~5,000 tokens with better signal-to-noise.

#### `synchestra-spec-search` → `synchestra spec search <query>`

**What it does:** Semantic or keyword search across all spec documents, returning ranked results with
file paths and matching excerpts.

**Why it matters:** Agents currently grep blindly or read indexes hoping to find relevant specs.
A search command would eliminate wasted exploration entirely.

#### `synchestra-spec-validate` → `synchestra spec validate [--feature <path>]`

**What it does:** Checks structural conventions:
- Every directory has README.md
- README has required sections (Outstanding Questions, etc.)
- Features index is up-to-date
- Dependencies are bidirectionally consistent
- No broken cross-references

**Why it matters:** Agents making spec edits can validate their work in one command instead of
manually checking 5+ conventions. Catches errors that agents frequently introduce.

### Tier 2: Spec Mutation Skills

These would let agents create and modify specs safely.

#### `synchestra-feature-create` → `synchestra feature create <feature-path>`

**What it does:** Scaffolds a new feature directory with:
- README.md with correct template (all required sections)
- Updates parent feature's Contents table
- Updates `spec/features/README.md` index
- Atomic commit-and-push

**Why it matters:** Creating a feature currently requires editing 3+ files with exact formatting.
Agents get this wrong ~30% of the time (missing OQ section, forgetting index update, wrong status
format). A single command eliminates all structural errors.

#### `synchestra-feature-update-status` → `synchestra feature status <path> <new-status>`

**What it does:** Updates a feature's status in its README and in the features index, atomically.

#### `synchestra-proposal-create` → `synchestra proposal create --feature <path>`

**What it does:** Scaffolds a proposal directory under the feature with correct template, non-normative
disclaimer, and index updates.

#### `synchestra-plan-create` → `synchestra plan create --features <paths>`

**What it does:** Scaffolds a development plan in `spec/plans/` with correct template, feature
references, and status tracking.

#### `synchestra-question-add` → `synchestra question add --feature <path> --text "..."`

**What it does:** Adds a question to a feature's Outstanding Questions section with proper formatting,
optional task linkage.

#### `synchestra-question-resolve` → `synchestra question resolve --feature <path> --question <id>`

**What it does:** Moves a question from open to resolved, with optional resolution text.

### Tier 3: Agent Workflow Skills

These compose multiple operations into agent-optimized workflows.

#### `synchestra-task-context` → `synchestra task context <task-path>`

**What it does:** Loads everything an agent needs before starting a task:
- Task description and requirements
- Linked feature spec (or summary)
- Feature dependencies' summaries
- Plan steps relevant to this task
- Related code file references (if any)

**Why it matters:** This is the **#1 token sink** for task-executing agents. Every agent session
starts with context assembly. A purpose-built command could cut startup cost by 60-70%.

#### `synchestra-spec-diff` → `synchestra spec diff [--since <commit>]`

**What it does:** Shows what specs changed since a given point, with summaries of each change.

**Why it matters:** Agents resuming work after interruption need to know what changed. Currently
they re-read everything.

#### `synchestra-feature-impact` → `synchestra feature impact <feature-path>`

**What it does:** Given a feature, shows:
- All features that depend on it (transitive)
- All tasks linked to it
- All plans referencing it
- Estimated "blast radius" of a change

**Why it matters:** Before editing a spec, agents need to understand consequences. This currently
requires manual traversal of the dependency graph.

### Summary: Proposed Skills Roadmap

```
Tier 1 — Navigation (highest ROI, read-only)
├── synchestra-feature-show        New CLI: feature show
├── synchestra-feature-context     New CLI: feature context
├── synchestra-spec-search         New CLI: spec search
└── synchestra-spec-validate       New CLI: spec validate

Tier 2 — Mutation (structural safety)
├── synchestra-feature-create      New CLI: feature create
├── synchestra-feature-status      New CLI: feature status (update)
├── synchestra-proposal-create     New CLI: proposal create
├── synchestra-plan-create         New CLI: plan create
├── synchestra-question-add        New CLI: question add
└── synchestra-question-resolve    New CLI: question resolve

Tier 3 — Agent Workflows (composite operations)
├── synchestra-task-context        New CLI: task context
├── synchestra-spec-diff           New CLI: spec diff
└── synchestra-feature-impact      New CLI: feature impact
```

---

## 5. Critical Assessment: Does Synchestra Actually Help?

### The Case FOR Synchestra Skills

**1. Token efficiency is real and measurable.**

AI agents are billed per token. Context window limits are hard ceilings. Every token spent on
navigation is a token not spent on reasoning. Synchestra's CLI-as-API approach means:
- Structured output (YAML/JSON) instead of parsing markdown
- Pre-computed relationships instead of manual graph traversal
- Filtered context instead of full-document reads

A `feature context` command returning 2,000 tokens of focused context vs. an agent reading 15,000
tokens across 6 files is a **7.5x efficiency gain**. At scale (hundreds of agent sessions), this
translates directly to cost savings.

**2. Structural consistency is genuinely hard for agents.**

Agents are statistically good at content but bad at convention compliance. They forget the OQ section,
use wrong status values, skip index updates. Mutation commands that enforce structure eliminate an
entire class of errors that currently require human review to catch.

**3. Multi-agent coordination has no good solution.**

When Agent A finishes a sub-task and Agent B needs to pick up dependent work, the handoff today is
manual. Synchestra's task claiming protocol (optimistic locking via git push) is a genuine innovation
— it provides distributed coordination without a central server, using infrastructure (git) that
already exists.

**4. The skill format is well-suited for progressive context loading.**

YAML frontmatter lets agents discover skills cheaply (name + description only). Full instructions
load only when needed. This aligns perfectly with how modern agent platforms (Claude Code, Cursor)
handle skills — metadata-first, content-on-demand.

### The Case AGAINST (Honest Concerns)

**1. Premature abstraction risk.**

Most of the proposed skills wrap CLI commands that don't exist yet, for workflows that are still
conceptual. Building a rich skill layer atop unimplemented CLI commands creates a specification
castle — impressive on paper, disconnected from reality. The risk is that the skills get designed
in a vacuum and don't match actual agent usage patterns.

**Mitigation:** Build skills incrementally. Start with `feature show` and `feature context` (read-only,
testable immediately against existing spec files). Don't spec mutation skills until the underlying
state management is implemented.

**2. Git-as-database has real limitations.**

Optimistic locking via git push works for low-frequency operations (task claiming). It doesn't scale
for high-frequency state updates (progress reporting, heartbeats). The spec acknowledges this with the
`state-store` abstraction, but the git-first philosophy may hit walls earlier than expected.

**3. The "naming conventions as API" bet is fragile.**

Synchestra's core insight — that directory structure IS the API — works beautifully for agents that
respect conventions. But it means any structural violation (typo in directory name, missing README,
wrong section header) silently corrupts the "database." Traditional databases reject malformed writes;
a filesystem doesn't. The `spec validate` command becomes critical infrastructure, not a nice-to-have.

**4. Agent platforms are evolving fast.**

Claude Code, Cursor, and Copilot are rapidly adding built-in capabilities for project understanding,
multi-file editing, and context management. Some of what Synchestra skills provide (feature discovery,
dependency tracing) may become unnecessary as agent platforms get smarter at codebase navigation.

**Counter-argument:** Agent platforms optimize for *code* navigation, not *specification* navigation.
Synchestra's value is domain-specific — it understands that `spec/features/cli/task/claim/README.md`
is a *feature specification* with dependencies and a lifecycle, not just a markdown file. Generic
code intelligence won't provide this semantic layer.

**5. Adoption friction is real.**

Every new tool an agent needs to learn takes system prompt space. If an agent needs to understand
19 task skills + 4 feature skills + 13 proposed new skills = 36 skills, that's significant cognitive
overhead. The skill descriptions need to be exceptionally well-written so agents can discover the
right skill without loading all of them.

---

## 6. Competitive Landscape

### Direct Competitors

| Tool | What it does | How it compares to Synchestra |
|---|---|---|
| **GitHub Spec Kit** | Spec-driven development toolkit. `.specify/` directory with `spec.md`, `plan.md`, `tasks/`. CLI for init/plan/implement phases. | Closest competitor. Simpler (flat structure, single spec file). No multi-agent coordination, no task claiming, no feature hierarchy. Synchestra is significantly more ambitious in scope. |
| **AGOR (AgentOrchestrator)** | Multi-agent dev coordination via git. Context transfer, snapshots, task state across agents. | Similar philosophy (git-backed coordination). More focused on agent runtime orchestration than specification management. Less structured spec layer. |
| **Kiro (AWS)** | Spec-first development for cloud-native. Agent-driven implementation from specs. | Narrower scope (AWS-focused). Similar spec→plan→task pipeline. No open multi-agent coordination protocol. |

### Adjacent Tools (Different Category, Partial Overlap)

| Tool | Overlap with Synchestra |
|---|---|
| **CrewAI** | Multi-agent role assignment and orchestration. Runtime-focused (Python), not spec-focused. |
| **AutoGen (Microsoft)** | Conversation-driven multi-agent coordination. Runtime, not persistent state. |
| **LangGraph** | Graph-based workflow orchestration. Stateful transitions but in-memory, not git-backed. |
| **Google ADK** | Agent Development Kit. Hierarchical task decomposition. Runtime, not specification layer. |

### Synchestra's Unique Position

No existing tool combines all three of:
1. **Structured specification management** (hierarchical features, proposals, plans, OQs)
2. **Git-backed distributed coordination** (task claiming, optimistic locking, audit trail)
3. **Agent-native interface** (CLI + skills + MCP, designed for token efficiency)

GitHub Spec Kit is the closest, but it's a **starting point** (flat specs, single-agent workflow),
while Synchestra is an **operating system** for spec-driven multi-agent development. Whether that
ambition is a strength (comprehensive) or a weakness (overengineered) depends on execution.

### What Synchestra Could Learn From Competitors

- **From Spec Kit:** Simplicity wins adoption. A `synchestra init` that scaffolds a working project
  in 30 seconds would lower the barrier dramatically.
- **From CrewAI:** Role-based agent assignment is intuitive. Synchestra's task claiming is
  mechanism-focused; adding a "what kind of agent should work on this" layer could improve routing.
- **From LangGraph:** Checkpoint/resume semantics are valuable. Agents that crash mid-task should be
  able to resume from a known state, not restart from scratch.

---

## 7. Conclusions & Recommendations

### Is Synchestra Valuable for AI Agents?

**Yes, conditionally.** The core value proposition — structured specs as a navigable API, with CLI
commands that return focused context — is genuinely useful and not well-served by existing tools.
The token efficiency argument is real and will only grow more important as AI usage scales.

But the value is **unrealized until the CLI commands exist.** Skills that wrap unimplemented commands
are documentation, not tools. The priority should be:

### Recommended Build Order

**Phase 1: Make existing specs machine-readable (weeks, not months)**
1. `synchestra feature show` — Structured summary of any feature (the single highest-ROI command)
2. `synchestra feature context` — Load feature + deps as focused context bundle
3. `synchestra spec validate` — Catch structural errors programmatically
4. Skills for all three

**Phase 2: Enable safe spec mutation (after Phase 1 is validated)**
5. `synchestra feature create` — Scaffold with guaranteed structure
6. `synchestra question add/resolve` — Manage OQs programmatically
7. Skills for both

**Phase 3: Multi-agent workflow support (after task commands are implemented)**
8. `synchestra task context` — Pre-assembled context for task execution
9. `synchestra spec diff` — Incremental change awareness
10. `synchestra feature impact` — Change blast radius analysis

### The Honest Bottom Line

Synchestra is building something that **doesn't exist elsewhere** — a specification-native
coordination layer for AI agents. The spec organization is well-designed, the skill format is
thoughtful, and the git-backed concurrency model is clever.

The risk isn't the concept — it's the gap between specification and implementation. With 24 features
specified and ~374 lines of Go implemented, the spec-to-code ratio is heavily skewed. The proposed
skills would be transformative *if the underlying CLI commands get built*.

**For an AI agent like me, the #1 thing that would make working on this repository faster and
cheaper is `synchestra feature context <path>`.** One command that gives me everything I need to
understand a feature, instead of reading 6 files across 4 directories. Build that, and the rest
follows.

---

## Outstanding Questions

- Should `feature show` output be YAML, JSON, or a custom token-efficient format?
- Should `feature context` include the full README text or a summarized version? (Full text is more
  accurate; summary is cheaper. Perhaps a `--depth` flag?)
- How should skills handle the case where the CLI command isn't implemented yet — should the skill
  README exist as a forward-looking spec, or only be created when the command ships?
- Is there value in a `synchestra agent context` command that returns "everything this agent needs
  to know" based on its current task assignment — combining task context, feature context, and
  relevant code pointers into a single output?
- Should spec search be keyword-based (fast, deterministic) or semantic/embedding-based (smarter,
  more expensive)? Or both with a flag?
