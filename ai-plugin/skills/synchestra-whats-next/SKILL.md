---
name: synchestra-whats-next
description: Generates or updates the WHATS-NEXT.md prioritization report for plans. Use when a plan or task is completed, or when the user wants to see what to work on next.
---

# Skill: synchestra-whats-next

Generate or update `spec/plans/WHATS-NEXT.md` — an AI-generated prioritization report that surfaces completed work, in-progress plans, and recommended next targets.

**CLI reference:** [development-plan feature spec](../../spec/features/development-plan/README.md)

## When to use

- After completing a plan or task (when `planning.whats_next` is `incremental` or `full`)
- When the user asks "what should we work on next?"
- When the user explicitly invokes this skill

## Modes

### Incremental (default)

1. Read the existing `spec/plans/WHATS-NEXT.md`
2. Determine what changed since the last generation (completed plans/tasks, status changes)
3. Update only the affected sections
4. Commit the updated file

### Full (--full or first-time generation)

1. Scan all features via `synchestra feature list --fields=status`
2. Scan all plans in `spec/plans/` — read each README.md for status, effort, impact, dependencies
3. If a state store is available, check task progress for approved plans
4. Generate the complete report from scratch
5. Commit the file

## Report structure

Write `spec/plans/WHATS-NEXT.md` with this structure:

```markdown
# What's Next

**Generated:** YYYY-MM-DD
**Mode:** incremental | full

## Completed Since Last Update

- [plan-slug](plan-slug/) — completed YYYY-MM-DD

## In Progress

- [plan-slug](plan-slug/) — N/M steps done, blockers (if any)

## Recommended Next

1. **[plan-slug](plan-slug/)** — Impact: X, Effort: Y. One-sentence reasoning.
2. ...

### Reasoning

2-5 sentences explaining the prioritization: dependency unlocks, ROI, momentum, competing priorities.

## Outstanding Questions

(any ambiguities surfaced during analysis)
```

## Prioritization logic

Rank candidates by combining these signals (in priority order):

1. **Explicit ROI metadata** — `**Effort:**` and `**Impact:**` fields in plan headers. Higher impact / lower effort = higher priority.
2. **Dependency unlocks** — Plans newly unblocked by recent completions get a priority boost.
3. **Momentum** — Prefer advancing roadmaps that are already in progress over starting new ones.
4. **Feature importance** — Plans targeting features closer to "stable" status get a boost.
5. **AI inference** — When ROI metadata is absent, infer effort from step count/dependency depth and impact from feature importance/downstream dependents.

## Process

1. Check config: read `synchestra-spec-repo.yaml` for `planning.whats_next` setting.
2. If invoked automatically and config is `disabled`, skip silently.
3. Determine mode (incremental or full).
4. Gather data (features, plans, tasks).
5. Generate report following the structure above.
6. Write to `spec/plans/WHATS-NEXT.md`.
7. Commit: `git commit -m "chore: update WHATS-NEXT.md (mode)"`

## Notes

- This skill generates content that costs tokens. Only invoke automatically when the config enables it.
- The report is committed to git, providing a history of how priorities evolved.
- When no plans exist or all are in draft, generate a minimal report noting this.
