# Feature: Workflow — Create Feature

**Status:** Conceptual

## Summary

The "Create Feature" workflow guides a user through defining a new [feature](../../../feature/README.md) specification from scratch. The user starts from the features index (or any context where "New feature" is available), and the system creates a new feature document in a dedicated branch, iterates on the spec within the chat, and submits it for review.

## Problem

Creating a new feature specification requires understanding the feature README structure, metadata conventions, and how features relate to proposals, plans, and tasks. A guided workflow lowers this barrier and ensures that new features follow project conventions from the start.

## Behavior

### Trigger

A user clicks "New Feature" on the features index page or a feature directory in the web UI.

### Anchor types

- `feature-index` — the features index page (`spec/features/`)
- `feature` — creating a sub-feature under an existing feature

### Anchor and branch creation

When the chat starts:

1. The AI asks "What new feature are we going to create?"
2. Based on the user's description, the system creates a new branch (e.g., `feature/{feature-slug}`)
3. A feature directory and initial `README.md` are created on that branch following the [feature README structure](../../../feature/README.md#feature-readme-structure)
4. The chat anchors to the new document on the new branch

The chat metadata includes the branch reference:

```yaml
anchor: spec/features/{feature-slug}/README.md
branch: feature/{feature-slug}
workflow: create-feature
status: active
```

### Steps

#### 1. Understand

The AI asks what the new feature should do, who it serves, and what problem it solves. If creating a sub-feature, the AI loads the parent feature for context.

**Goal:** the AI understands the feature's purpose, scope, and target users.

#### 2. Draft

The AI creates the feature spec document following the project's feature conventions: title, status (`Conceptual`), summary, problem, behavior, and outstanding questions. The document is committed to the feature branch.

**Produces:** `feature` — a feature specification document.

#### 3. Iterate

The user and AI refine the spec within the chat. The user can request changes to any section, add detail, adjust scope, or split the feature into sub-features. Each iteration updates the document on the branch.

**Goal:** the feature spec is comprehensive enough for review.

#### 4. Finalize

Behavior depends on role:

| Role | Finalize behavior |
|---|---|
| Maintainer | Can merge directly to the default branch or create a PR |
| Contributor | Creates a PR for review |

**Produces:** `pull-request` (or direct merge for maintainers)

### Workflow definition

```yaml
name: create-feature
title: New Feature
description: Create a new feature specification
anchor-types: [feature-index, feature]
produces: [feature, pull-request]
roles: ["*"]
allow-create: true
context:
  load:
    - anchor-document
    - feature-conventions    # resolves to the Feature spec (spec/features/feature/README.md) and project conventions (CLAUDE.md)
retention: archive
ui:
  icon: plus-circle
  variant: primary
  sort-order: 5
steps:
  - name: understand
    prompt: gather-feature-requirements
    description: Understand what the new feature should do
  - name: draft
    prompt: draft-feature-spec
    description: Create the feature specification document
    produces: [feature]
  - name: iterate
    prompt: refine-feature-spec
    description: Refine the specification based on feedback
  - name: finalize
    prompt: finalize-feature
    description: Submit for review or merge
    produces: [pull-request]
```

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Chat](../../README.md) | This workflow runs on the chat layer. |
| [Workflow](../README.md) | This is one of the built-in workflows. |
| [Feature](../../../feature/README.md) | The primary artifact produced is a feature spec document following the feature conventions. |
| [Proposals](../../../proposals/README.md) | A newly created feature may later receive proposals through the "Create Proposal" workflow. |

## Outstanding Questions

- When creating a sub-feature, should the parent feature's README be automatically updated with the new Contents entry, or should that be a separate step?
- Should the system suggest related existing features during the Understand step to avoid duplicates?
- How should the feature slug be determined — AI-suggested based on the description, user-specified, or a combination?
