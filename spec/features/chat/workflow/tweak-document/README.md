# Feature: Workflow — Tweak Document

**Status:** Conceptual

## Summary

The "Tweak Document" workflow handles quick fixes — typos, formatting, clarifications, small wording changes, and minor code fixes — with minimal ceremony. It is the lightest-weight [workflow](../README.md) and is designed for changes that do not require brainstorming or exploration.

## Problem

Small fixes (a typo in a feature spec, a formatting issue, a one-line code fix) do not warrant the overhead of a full proposal or issue workflow. But making these fixes still requires understanding the project's file structure, editing conventions, and submission process. A lightweight workflow removes this friction and encourages incremental quality improvements.

## Behavior

### Trigger

A user clicks "Tweak" (or similar action) on any document or code file in the web UI.

### Anchor types

- `feature` — a feature spec document
- `feature-section` — a section of a feature spec
- `proposal` — a proposal document
- `spec-document` — any document in the spec repo
- `code-file` — a file in a code repository (if allowed by project config)

### Scope: spec documents vs code files

The tweak workflow can target both spec repo documents and code repo files, but with different rules:

| Target | Allowed by default | Submit behavior |
|---|---|---|
| Spec document | yes | Varies by role (see below) |
| Code file | no (requires `allow-code-changes: true`) | Always PR with CI |

**Code change constraints:**
- Code tweaks are limited to a **single code repository.** If the change requires modifications across multiple repos, the system suggests escalating to the "Create Proposal" workflow — multi-repo changes need a development plan and task status board.
- All code changes go through a **PR with CI validation**, regardless of the user's role.

### Steps

#### 1. Understand

The AI asks what the user wants to change. For simple requests ("fix the typo in the third paragraph"), this step may be near-instant — the AI confirms the fix and moves on.

**Goal:** the AI knows exactly what to change.

#### 2. Apply

The AI produces the edit. For spec documents, this is a direct modification. For code files, this creates a branch and applies the change.

**Produces:** the modified document or code file.

#### 3. Verify

Runs project-configured checks against the modified content:
- For spec documents: spell check, linting, link validation
- For code files: the target repo's CI pipeline via PR

Failures are surfaced to the user with actionable output. The user can iterate on the change until checks pass.

**Goal:** the change meets project quality standards.

#### 4. Submit

Behavior depends on the target type and user role:

| Target | Role | Submit behavior |
|---|---|---|
| Spec document | Maintainer | Direct commit to the default branch |
| Spec document | Trusted contributor | Creates a PR with the change |
| Spec document | External contributor | Creates a draft PR, requests review |
| Code file | Any role | Creates a PR (CI must pass) |

### Workflow definition

```yaml
name: tweak-document
title: Tweak
description: Quick fix — typo, formatting, or small change
anchor-types: [feature, feature-section, proposal, spec-document, code-file]
produces: [commit, pull-request]
roles: ["*"]
context:
  load:
    - anchor-document
retention: dispose
ui:
  icon: pencil
  variant: subtle
  sort-order: 50
steps:
  - name: understand
    prompt: gather-tweak-details
    description: Understand what to change
  - name: apply
    prompt: apply-tweak
    description: Make the change
  - name: verify
    prompt: verify-tweak
    description: Run checks against the change
  - name: submit
    prompt: submit-tweak
    description: Commit or create a PR
    produces: [commit, pull-request]
```

### Project configuration

```yaml
workflows:
  tweak-document:
    allow-code-changes: true         # default: false
    code-repos: [main-app]           # which code repos allow tweaks (single repo only)
    escalate-on-multi-repo: true     # suggest proposal workflow if multi-repo change detected
```

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Chat](../../README.md) | This workflow runs on the chat layer. Tweak chats are typically short-lived. |
| [Workflow](../README.md) | This is one of the built-in workflows. It has the simplest step sequence. |
| [Feature](../../../feature/README.md) | Can tweak feature spec documents. |
| [Proposals](../../../proposals/README.md) | Can tweak proposal documents. May escalate to "Create Proposal" for larger changes. |

## Outstanding Questions

- Should maintainers be able to direct-commit code changes (bypassing PR), or should code always require a PR regardless of role?
- Should there be a size/complexity heuristic that automatically suggests escalating to a proposal workflow if the "tweak" grows too large?
- Should the tweak workflow support batch changes (e.g., "fix all typos in this feature spec"), or is that a separate workflow?
