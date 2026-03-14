# Feature: Proposals

**Status:** Conceptual

## Summary

Proposals let humans and agents request changes to an existing feature without changing the current specification immediately. A proposal lives under the feature it targets, can link to a single external tracker issue, and remains explicitly non-normative until its content is incorporated into the feature's main `README.md`.

## Problem

Feature ideas, change requests, and refinements need a place to live before they become part of the canonical specification. Today, those requests would either be written directly into the feature spec too early or live outside Synchestra in chat threads and issue trackers.

That creates two problems:

- The current feature spec becomes polluted with ideas that are not yet accepted
- Change requests lose context when they are tracked only in external systems such as GitHub Issues

Synchestra needs a first-class way to record proposed changes while keeping the current system description clean and reliable.

## Proposed Behavior

### Proposal location

Each feature may contain a `proposals/` subtree:

```text
spec/features/{feature}/
  README.md
  proposals/
    README.md
    {proposal}/
      README.md
```

`{proposal}` is a slug scoped to the feature (for example `support-passkeys`).

### Proposals are not part of the current specification

This rule is the core of the feature:

- A proposal is a change request, not a description of current behavior
- Proposal content must be ignored by default when understanding the current system state
- Context generation, summarization, indexing, task planning, and feature overviews should exclude proposals unless the caller explicitly asks for them

This disclaimer must appear in three places:

1. At the top of every proposal `README.md`
2. At the top of the feature's `proposals/README.md`
3. In the `Proposals` section of the parent feature `README.md`

Recommended wording:

> Proposals are change requests and are not part of the current specification unless and until their content is incorporated into this feature's main README.

### Proposal document structure

Each proposal lives in its own directory with a `README.md`. The proposal README should contain:

- Title
- Proposal status
- Submitter
- Created date
- Submitted date (if applicable)
- External tracker link (if present)
- Summary of the requested change
- Detailed proposal body
- Relationship to the current feature spec

A simple markdown metadata table near the top is sufficient for both humans and parsers:

```markdown
# Proposal: Support passkeys

> Proposals are change requests and are not part of the current specification unless and until their content is incorporated into this feature's main README.

| Field | Value |
|---|---|
| Status | `submitted` |
| Submitter | `@alex` |
| Created | `2026-03-12` |
| Submitted | `2026-03-13` |
| Tracker | [GitHub issue #42](https://github.com/org/repo/issues/42) |

## Summary

Add passkey-based authentication as an alternative to password login.

## Proposed Change

...

## Relationship to Current Spec

This proposal is not implemented in the current feature spec.

## Outstanding Questions

None at this time.
```

### Proposal statuses

Proposals use these statuses:

| Status | Meaning |
|---|---|
| `draft` | Proposal is being written or revised and is not yet under active review |
| `submitted` | Proposal is ready for review |
| `approved` | Proposal direction is accepted, but the feature spec may not yet include the change |
| `rejected` | Proposal was reviewed and not accepted in its current form |
| `implemented` | Proposal has been incorporated into the feature's main spec and should be treated as historical record rather than pending change |

### Status transitions

```text
draft → submitted → approved → implemented
   ↑         ↓           ↓
   └─────────┘           └──────────────┐
     (withdrawn)                        │
                                        │
submitted → rejected → draft → submitted│
                                        │
submitted ─────────────────────────────→ implemented
```

Notes:

- `submitted -> draft` is allowed and represents a withdrawn proposal
- `rejected -> draft` is allowed so the proposal can be revised and resubmitted
- `submitted -> implemented` is allowed for fast-tracked changes, but only when the parent feature spec is updated in the same change set

### Implemented is a guarded terminal state

`implemented` is only valid if the parent feature's main `README.md` has been updated to incorporate the proposal's accepted behavior.

If a proposal is marked `implemented` but the feature README still omits material parts of the proposal, the transition must be rejected. In other words:

- `approved` means the direction is accepted
- `implemented` means the accepted change is now reflected in the normative feature spec

This keeps proposal history and current feature behavior aligned.

### Feature proposals page

If a feature has a `proposals/` directory, it must contain its own `README.md`.

The `proposals/README.md` file:

- States clearly that proposals are non-normative
- Lists all proposals for the feature
- Links to each proposal directory
- Shows proposal status, submitter, tracker link, and created date

Example:

```markdown
# Proposals: Authentication

> Proposals are change requests and are not part of the current specification unless and until their content is incorporated into the feature's main README.

## Index

| Proposal | Status | Submitter | Tracker | Created |
|---|---|---|---|---|
| [support-passkeys](support-passkeys/README.md) | `submitted` | `@alex` | [#42](https://github.com/org/repo/issues/42) | `2026-03-12` |
| [magic-link-login](magic-link-login/README.md) | `draft` | `@sam` | — | `2026-03-15` |

## Outstanding Questions

None at this time.
```

### Feature README proposals section

Each feature README may include a `Proposals` section when proposals exist for that feature.

That section:

- Repeats the non-normative disclaimer
- Shows a compact table with columns `Proposal`, `Submitter`, and `Status`
- Displays only the last `N` proposals
- Orders the displayed rows by creation date ascending

This means the system first selects the newest `N` proposals by `created_at`, then renders that slice from oldest to newest.

Example:

```markdown
## Proposals

> Proposals are change requests and are not part of the current specification unless and until their content is incorporated into this feature's main README.

| Proposal | Submitter | Status |
|---|---|---|
| [support-passkeys](proposals/support-passkeys/README.md) | `@alex` | `submitted` |
| [magic-link-login](proposals/magic-link-login/README.md) | `@sam` | `draft` |
| [biometric-fallback](proposals/biometric-fallback/README.md) | `@jo` | `approved` |
```

### Configurable feature-page limit

The number of proposals shown in the feature README is configurable per project.

| Setting | Description | Default |
|---|---|---|
| `proposals.feature_page.limit` | Maximum number of proposals shown in a feature README's `Proposals` section | `3` |

If the setting is absent, the default is `3`.

### External tracker linkage

Each proposal may link to zero or one external tracker issue.

For the MVP:

- The only supported tracker type is a GitHub issue
- The relationship is one proposal to one GitHub issue
- The proposal stores the issue URL and display label

The model should remain extensible so future tracker kinds such as GitLab issues or Jira tickets can be added without redesigning proposal storage.

### Synchestra UI behavior

Synchestra UI can create proposals from a feature page.

The create flow should:

1. Create the proposal directory and `README.md`
2. Ensure the feature's `proposals/README.md` exists and is updated
3. Update the parent feature's `Proposals` section
4. Optionally create and link a GitHub issue

If the user requests automatic GitHub issue creation and that tracker operation fails, the UI must surface the error explicitly rather than silently dropping the link.

For the MVP, proposal creation is defined for the UI flow first. The storage model and document structure should remain compatible with future CLI and API support.

## Interaction with current-state understanding

Proposal content must be excluded from default "what is the system today?" workflows, including:

- Feature summaries
- Generated implementation context
- Current behavior explanations
- State-of-the-world dashboards
- Automatic task planning based on accepted spec only

Proposal content is included only when:

- The user explicitly asks about proposals
- A workflow is operating on proposal review or proposal implementation
- A proposal is being incorporated into the main feature spec

## Interaction with Development Plans

An approved proposal is a trigger for [development plan](../development-plan/README.md) creation. When a plan is created from a proposal:

- The plan's **Source type** is `change-request` and its **Source** field links to the proposal.
- The proposal gains a **Plan** field linking forward to the plan.

This creates a bidirectional, traceable chain: **proposal → plan → tasks**.

```markdown
# Proposal: Deprecate v1 endpoints

| Field | Value |
|---|---|
| Status | `approved` |
| Plan | [migrate-to-v2](../../../plans/migrate-to-v2/) |
```

## Additional Rules

- The parent feature's compact `Proposals` table includes proposals regardless of status, including `draft` and `rejected`, subject to the configured row limit.
- For this specification layer, incorporation into the parent feature `README.md` is the gate for `implemented`. Runtime delivery may be tracked elsewhere, but it is not required to keep the proposal history consistent with the spec.

## Outstanding Questions

None at this time.
