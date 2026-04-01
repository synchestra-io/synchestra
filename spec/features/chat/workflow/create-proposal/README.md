# Feature: Workflow — Create Proposal

**Status:** Conceptual

## Summary

The "Create a Proposal" workflow guides a user through brainstorming, exploring approaches, and drafting a formal change request ([proposal](../../../proposals/README.md)) for an existing feature. It is the flagship [workflow](../README.md) and the most common entry point for contributing changes to a Synchestra-managed project.

For maintainers and authorized contributors, a fast path allows the system to implement the change during the conversation and produce a plan as a report of what was done.

## Problem

Contributing a well-formed proposal to a project requires understanding the feature's current specification, the proposal format, and enough domain context to articulate a coherent change request. This is a high bar — especially for first-time contributors or users who have a good idea but lack familiarity with the project's conventions.

## Behavior

### Trigger

A user navigates to a feature or a section of a feature in the web UI and clicks "Create a Proposal."

### Anchor types

- `feature` — the full feature document
- `feature-section` — a specific section of a feature document

### Steps

#### 1. Understand

The AI reads the anchored feature (or section) and asks the user what they want to change and why. Project-configured additional questions are woven into this conversation naturally — not presented as a checklist.

**Goal:** the AI has a clear understanding of the desired change, its motivation, and which users or components are affected.

#### 2. Explore

The AI proposes 2-3 approaches to the change with trade-offs, and discusses them with the user until an approach is chosen.

**Goal:** an approach is selected and the user understands the trade-offs.

#### 3. Draft

The AI produces a proposal document following the project's [proposal format](../../../proposals/README.md). The proposal is created in the chat's `artifacts/` directory. The user can iterate on the draft within the chat — requesting changes, adding detail, or adjusting scope.

**Produces:** `proposal` — a proposal document ready for submission.

#### 4. Review (optional)

The AI reviews the draft proposal for completeness, clarity, and compliance with project-configured rules. If checks are configured (spell-check, link validation, custom scripts), they run against the draft. Failures are surfaced to the user with actionable output.

**Goal:** the proposal meets the project's quality standards before submission.

### Fast path

When a maintainer or authorized contributor initiates this workflow, the AI may suggest the fast path after the Explore step:

> "This change looks straightforward — would you like me to implement it now and produce a plan as a report?"

If accepted:

1. Tasks are created in the state repo
2. Agents implement the change in a code branch
3. A plan is produced documenting what was done (plan-as-report)
4. A PR is prepared with the code changes

**Fast-path constraints:**
- Available only to users with `maintainer` or authorized contributor roles
- Limited to changes affecting a **single code repository**
- If multi-repo changes are detected, the AI suggests the standard path instead
- All code changes go through a **PR with CI validation**

**Produces (fast path):** `proposal`, `plan`, `pull-request`

### Standard-path output

After finalization, the proposal is committed to the spec repo under the feature's `proposals/` directory. From there it enters the normal Synchestra pipeline:

```
Proposal -> [Review] -> Plan -> Tasks -> Implementation -> PR
```

### Workflow definition

```yaml
name: create-proposal
title: Create a Proposal
description: Brainstorm and draft a change request for a feature
anchor-types: [feature, feature-section]
produces: [proposal]
roles: ["*"]
context:
  load:
    - anchor-document
    - existing-proposals
retention: archive
ui:
  icon: lightbulb
  variant: primary
  sort-order: 10
steps:
  - name: understand
    prompt: gather-requirements
    description: Understand what the user wants to change and why
  - name: explore
    prompt: brainstorm-approaches
    description: Explore approaches and trade-offs
  - name: draft
    prompt: draft-proposal
    description: Write the proposal document
    produces: [proposal]
  - name: review
    prompt: self-review-proposal
    description: Review the draft for completeness and clarity
    optional: true
```

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Chat](../../README.md) | This workflow runs on the chat layer. Each "Create a Proposal" action starts a new chat. |
| [Workflow](../README.md) | This is one of the built-in workflows. Its structure and steps follow the workflow schema. |
| [Proposals](../../../proposals/README.md) | The primary artifact produced is a proposal document, committed to the feature's `proposals/` directory. |
| [Plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md) | Fast-path produces a plan-as-report. Standard-path proposals later trigger plan creation through the normal pipeline. |
| [Feature](https://github.com/synchestra-io/specscore/blob/main/spec/features/feature/README.md) | The workflow anchors to features and produces proposals that attach to them. |

## Outstanding Questions

- Should the fast path be offered proactively by the AI, or should the user explicitly request it via a UI toggle?
- When the fast path produces a PR, should the chat remain open until the PR is merged, or finalize once the PR is created?
- Should the Explore step be skippable for users who already know exactly what they want?
