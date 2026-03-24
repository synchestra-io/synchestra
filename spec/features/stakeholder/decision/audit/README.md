# Feature: Stakeholder / Decision / Audit

**Status:** Conceptual

## Summary

Every [decision](../README.md) task accumulates a record of stakeholder responses in a `decisions.md` file within the task directory. The audit log captures who responded, what they chose, when, and the final outcome once the resolution policy is satisfied. It is the authoritative record of how and why a decision was made.

## Problem

Decisions today are ephemeral. An approval happens in a chat message, a code review comment, or a verbal agreement. When someone later asks "who approved this plan?" or "why did we pick JWT over OAuth?", the answer is scattered across tools or lost entirely. There is no single, queryable record of decision history within the project.

## Behavior

### File Location

The audit log lives in the decision task's directory:

```
synchestra/projects/{project}/tasks/{decision-task}/decisions.md
```

### Format

The audit log is a markdown file with a structured table for responses and an outcome summary:

```markdown
# Decision Log

## API Authentication Method

| Stakeholder | Response | Timestamp |
|---|---|---|
| alex@github | `jwt` | 2026-03-24T14:22:00Z |
| carol@github | `jwt` | 2026-03-24T15:01:00Z |

**Outcome:** `jwt` (policy satisfied: 2/2 min)
**Resolved:** 2026-03-24T15:01:00Z
```

### Response Recording

**Predefined option responses** are recorded by `key`:

```markdown
| alex@github | `jwt` | 2026-03-24T14:22:00Z |
```

**Custom responses** (when `allow_custom: true` and the stakeholder provides free text) link to a detail section:

```markdown
| Stakeholder | Response | Timestamp |
|---|---|---|
| alex@github | `jwt` | 2026-03-24T14:22:00Z |
| carol@github | custom: [see below](#carol-github) | 2026-03-24T15:01:00Z |

### carol-github

> What about mTLS? We already have cert infrastructure
> from the internal services. Would avoid token management entirely.
```

**Multi-select responses** (`pick-many`) list all selected keys:

```markdown
| alex@github | `unit-tests`, `integration-tests` | 2026-03-24T14:22:00Z |
```

### Outcome Row

The outcome row is written when the resolution policy is satisfied:

```markdown
**Outcome:** `jwt` (policy satisfied: 2/2 min)
**Resolved:** 2026-03-24T15:01:00Z
```

For unanimous decisions, the outcome is the common choice. For mixed decisions (custom responses, split votes), the outcome is descriptive:

```markdown
**Outcome:** mixed — see responses (policy satisfied: 2/2 min, custom response received)
**Resolved:** 2026-03-24T15:01:00Z
```

### Multiple Decisions Per Task

A single task may spawn multiple decisions over its lifetime — an agent gets unblocked, continues work, then hits another decision point. All decisions are recorded in the same `decisions.md` file as separate sections:

```markdown
# Decision Log

## API Authentication Method

| Stakeholder | Response | Timestamp |
|---|---|---|
| alex@github | `jwt` | 2026-03-24T14:22:00Z |

**Outcome:** `jwt` (policy satisfied: 1/1 any)
**Resolved:** 2026-03-24T14:22:00Z

## JWT Token Expiry Duration

| Stakeholder | Response | Timestamp |
|---|---|---|
| alex@github | `1h` | 2026-03-24T16:45:00Z |

**Outcome:** `1h` (policy satisfied: 1/1 any)
**Resolved:** 2026-03-24T16:45:00Z
```

### Design Choices

- **Markdown, not YAML** — the audit log is a human-readable record, not machine configuration. It should be easy to scan in a text editor or on GitHub.
- **Co-located with the task** — decisions belong to their task context. A centralized audit index across the project may be introduced later, but the source of truth is always the task directory.
- **Append-only** — responses are never edited or removed once recorded. The log is an immutable audit trail.

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria are not yet defined for this feature.

- Should the audit log include a machine-readable frontmatter section summarizing all outcomes for programmatic access?
- Should there be a project-wide audit index that aggregates decision outcomes across all tasks for reporting purposes?
- How should the audit log handle decision expiry — is "expired" an outcome, and what metadata accompanies it?
