# What's Next Report & Hierarchical Plans

## Summary

Three additions to the Synchestra planning system:

1. **Hierarchical plans** — plans can nest like features, with roadmap-level parents containing child plans
2. **Optional ROI metadata** — `Effort` and `Impact` fields on plans, AI-suggested during authoring, human-controlled
3. **`WHATS-NEXT.md`** — an AI-generated prioritization report updated incrementally on plan/task completion

### Key decisions

- **Nesting limit:** 2 levels (roadmap → plan), consistent with the 2-level step limit inside plans
- **Roadmaps have no implementation steps** — they define ordering between child plans
- **ROI metadata is optional** — AI infers from plan structure when absent
- **Report generation is opt-in** via `planning.whats-next` config to avoid surprise token spend
- **Incremental by default** — previous report + delta, with `--full` override for from-scratch regeneration

## Motivation

The plans README currently lists all plans in a flat table with no guidance on what to work on next. As the number of plans grows, three problems emerge:

- **No priority signal.** All plans appear equally important. A human scanning the table must mentally evaluate dependencies, feature importance, and effort to decide where to focus.
- **No structural grouping.** Related plans (e.g., `chat-infrastructure` and `chat-workflow-engine` both serving the Chat roadmap) are flat siblings with no visible relationship.
- **No completion-driven recommendations.** When a plan completes and unblocks downstream work, nothing surfaces what just became actionable.

## Hierarchical Plans

### Directory structure

Plans can nest to mirror the feature tree convention:

```text
spec/plans/
  README.md                          ← index
  chat-feature/
    README.md                        ← roadmap plan (parent)
    chat-infrastructure/
      README.md                      ← child plan
    chat-workflow-engine/
      README.md                      ← child plan
  e2e-testing-framework/
    README.md                        ← standalone plan (no children)
```

### Rules

- A **roadmap** (parent plan) defines ordering and dependencies between child plans. It does not have implementation steps — its Steps section lists child plans with their relationships, not implementation work.
- A **child plan** has steps, task mappings, and acceptance criteria — same format as today's plans.
- A **standalone plan** (no children) works exactly as today. No changes required.
- Nesting is limited to **2 levels**: roadmap → child plan. Deeper nesting belongs in task decomposition.

### Roadmap document structure

```markdown
# Plan: Chat Feature Roadmap

**Status:** draft
**Features:**
  - [chat](../../features/chat/README.md)
  - [chat/workflow](../../features/chat/workflow/README.md)
**Source type:** feature
**Source:** [Chat feature spec](../../features/chat/)
**Author:** @alex
**Created:** 2026-03-24
**Effort:** XL
**Impact:** critical

## Context

High-level roadmap for the Chat feature. Decomposes into sequential
phases, each with its own child plan.

## Acceptance criteria

- All child plans completed
- Chat feature status moves to Stable

## Child Plans

| Order | Plan | Status | Effort | Impact |
|-------|------|--------|--------|--------|
| 1 | [chat-infrastructure](chat-infrastructure/) | draft | L | high |
| 2 | [chat-workflow-engine](chat-workflow-engine/) | draft | M | high |
```

### Roadmap status derivation

A roadmap's status is derived from its children:

- `draft` — at least one child is `draft`
- `in_review` — all children are `in_review` or `approved`
- `approved` — all children are `approved`
- `in_progress` — at least one child plan has linked tasks in progress
- `superseded` — explicitly set when the roadmap is replaced

### Feature linking

The existing bidirectional linking convention scales to hierarchy without changes:

- **Feature → Plan/Roadmap:** A feature's `## Plans` table can reference either a roadmap or a child plan. The path disambiguates:
  ```markdown
  | [chat-feature](../../plans/chat-feature/) | draft | @alex | — |
  | [chat-infrastructure](../../plans/chat-feature/chat-infrastructure/) | draft | @alex | — |
  ```
- **Plan/Roadmap → Feature:** The `**Features:**` header field lists affected features. A roadmap lists broad features; a child plan lists specific features it implements.
- A feature appearing in both a roadmap and its child plan is valid — the roadmap covers it broadly, the child plan implements a slice.
- A feature linked only to a roadmap (no child plan yet) signals "planned but not decomposed."

### Plans index table

The `spec/plans/README.md` table gains indentation to show hierarchy:

```markdown
| Plan | Status | Progress | Features | Effort | Impact | Author | Approved |
|---|---|---|---|---|---|---|---|
| [chat-feature](chat-feature/) | draft | — | chat, chat/workflow | XL | critical | @alex | — |
| &ensp;[chat-infrastructure](chat-feature/chat-infrastructure/) | draft | — | chat | L | high | @alex | — |
| &ensp;[chat-workflow-engine](chat-feature/chat-workflow-engine/) | draft | — | chat/workflow | M | high | @alex | — |
| [e2e-testing-framework](e2e-testing-framework/) | draft | — | testing-framework | — | — | @alex | — |
```

## Optional ROI Metadata

### Header fields

Two optional fields added to the plan document header:

```markdown
**Effort:** S | M | L | XL
**Impact:** low | medium | high | critical
```

### Rules

- Both fields are **optional**. When absent, the AI infers effort from step count, dependency depth, and acceptance criteria complexity. It infers impact from feature importance and downstream dependents.
- During plan authoring (brainstorming/writing-plans flow), the AI **suggests** values. The user accepts, declines, or overwrites.
- For roadmaps, effort/impact describe the aggregate. Child plans carry independent estimates.
- The AI report may flag plans where estimates seem inconsistent with actual structure (e.g., a plan marked `S` with 12 steps).

### Scale definitions

| Effort | Rough meaning |
|--------|---------------|
| S | A few hours of focused work, 1-3 steps |
| M | A few days, 3-6 steps, limited dependencies |
| L | A week or more, 5-10 steps, cross-cutting |
| XL | Multi-week, many steps, multiple child plans or deep dependencies |

| Impact | Rough meaning |
|--------|---------------|
| low | Nice-to-have, no users blocked |
| medium | Improves existing capability, some users benefit |
| high | Enables important new capability, many users benefit |
| critical | Unblocks core functionality or other critical work |

## Generated `WHATS-NEXT.md` Report

### Location

`spec/plans/WHATS-NEXT.md`

### Structure

```markdown
# What's Next

**Generated:** 2026-03-24
**Mode:** incremental | full

## Completed Since Last Update

- [chat-infrastructure](chat-feature/chat-infrastructure/) — completed 2026-03-20

## In Progress

- [hero-scene](hero-scene/) — 2/4 steps done, no blockers

## Recommended Next

1. **[chat-workflow-engine](chat-feature/chat-workflow-engine/)** — Impact: high,
   Effort: M. Unblocked by chat-infrastructure completion. Advances the
   highest-impact roadmap.
2. **[agent-skills-roadmap](agent-skills-roadmap/)** — Impact: medium, Effort: L.
   No blockers, independent of current momentum.

### Reasoning

Brief AI explanation of prioritization — dependency unlocks, ROI ratio,
momentum, competing priorities.

## Outstanding Questions

(ambiguities the AI surfaced during analysis)
```

### Configuration

In `synchestra-spec-repo.yaml`:

```yaml
planning:
  whats-next: disabled          # disabled | incremental | full
```

- **`disabled`** (default) — no automatic generation, no token spend
- **`incremental`** — regenerated on plan/task completion events using previous report + delta
- **`full`** — regenerated from scratch on each completion event

The explicit command `synchestra plans whats-next` works regardless of config setting. Pass `--full` to force full regeneration.

### Update mechanism

- **Trigger:** plan or task completion events (via `synchestra task-complete` / plan status transitions)
- **Incremental mode:** reads previous `WHATS-NEXT.md` + the completion delta. Regenerates only affected sections. Minimizes token usage.
- **Full mode:** scans all features, plans, and task statuses. Used for initial generation or to correct incremental drift.
- The file is **committed to git** after each update, providing a history of how priorities evolved over time.

### Prioritization inputs (in order)

1. Explicit ROI metadata (effort/impact) when present
2. Dependency graph — what's newly unblocked by recent completions
3. Momentum — preference for advancing roadmaps already in progress
4. Feature status — features closer to "stable" get a boost
5. AI inference from plan complexity when ROI metadata is absent

## Outstanding Questions

- Should `synchestra plans whats-next` support a `--dry-run` flag that prints the report without committing?
- Should the report include a "Blocked" section for plans that are blocked on external dependencies?
