# Architecture Decision Records

This directory holds Architecture Decision Records (ADRs) — short, dated documents that capture significant architectural decisions for the Synchestra project: what was decided, why, and what alternatives were considered.

ADRs complement the [feature specs](../features/README.md). Where features describe *what the product does*, ADRs explain *why it was built that way* — context, tradeoffs, and rejected options that a future reader would otherwise have to reconstruct.

## When to write an ADR

Write an ADR when a decision:

- Is expensive to reverse (repository splits, protocol choices, major dependencies).
- Has non-obvious tradeoffs that future readers will re-litigate without context.
- Resolves a tension between two or more reasonable options.

Skip an ADR when the decision is trivially reversible or when the reasoning is fully captured in a feature spec.

## Format

Each ADR is a single markdown file named `NNNN-<slug>.md` with leading zeros (e.g., `0001-extract-ai-plugin.md`). Numbering is monotonic; never reused.

Every ADR follows this structure:

```markdown
# ADR-NNNN: <title>

**Status:** Proposed | Accepted | Superseded by ADR-NNNN
**Date:** YYYY-MM-DD

## Context
What situation prompted this decision? What forces are at play?

## Decision
What was chosen and why?

## Consequences
What becomes easier or harder as a result?

## Alternatives considered
What was rejected, and why?
```

Keep ADRs short. If one grows beyond ~2 pages, the decision probably belongs in a feature spec, not an ADR.

## Lifecycle

- **Proposed** — under discussion, not yet ratified.
- **Accepted** — in effect. The status line records the date of acceptance.
- **Superseded** — replaced by a later ADR. Keep the file; update the status line to point at the replacement. ADRs are never deleted; the history matters.

## Cross-repo decisions

Decisions that affect only the marketing/positioning narrative belong in [`synchestra-marketing`](https://github.com/synchestra-io/synchestra-marketing). Decisions that affect code, specs, or repository structure live here.

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-extract-ai-plugin.md) | Extract AI plugin to a dedicated repository | Accepted |
| [0002](0002-progressive-disclosure-skills.md) | Progressive-disclosure skill structure | Accepted |
| [0003](0003-skill-naming-plugin-namespace.md) | Skill directory names must not repeat the plugin namespace | Accepted |
| [0004](0004-layered-plugin-architecture.md) | Layered plugin architecture — CLI wrappers and methodology plugins | Accepted |
| [0005](0005-user-invocable-visibility.md) | Per-resource `user-invocable` visibility | Accepted |

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/decisions-index-specification*
