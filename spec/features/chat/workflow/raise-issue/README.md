# Feature: Workflow — Raise Issue

**Status:** Conceptual

## Summary

The "Raise an Issue" workflow helps a user articulate a problem or suggestion clearly, classifies it, optionally finds and quotes relevant code, proposes a fix, and submits a well-structured issue to the project's issue tracker (e.g., GitHub Issues).

## Problem

Users often spot problems or have suggestions while reading a feature spec or browsing code, but translating that observation into a well-structured issue requires context-gathering effort. The result is either vague issues that waste maintainer time or ideas that are never reported at all.

## Behavior

### Trigger

A user clicks "Raise an Issue" on a feature, a section of a feature, a proposal, or a code file in the web UI.

### Anchor types

- `feature` — the full feature document
- `feature-section` — a specific section of a feature document
- `proposal` — an existing proposal
- `code-file` — a file in a code repository (requires `repo` reference)

### Steps

#### 1. Understand

The AI reads the anchored document and asks the user to describe the problem or suggestion. Helps clarify vague descriptions with targeted questions. Project-configured additional questions are woven in naturally.

**Goal:** the AI has a clear understanding of what the user observed, what they expected, and what impact it has.

#### 2. Classify

The AI determines the issue type: bug, enhancement request, question, documentation issue, or other. Suggests appropriate labels and tags based on project conventions.

**Goal:** the issue is categorized for efficient triage.

#### 3. Find relevant code

The AI searches the project's code repositories to find and quote code relevant to the reported issue. This provides maintainers with immediate context without requiring them to search themselves.

**Enabled by default** for public repositories, **disabled by default** for private repositories. Projects can override via configuration:

```yaml
workflows:
  raise-issue:
    steps:
      - name: find-relevant-code
        enabled: auto    # auto | true | false
```

When `auto`, the system defaults based on repository visibility.

**Goal:** relevant code snippets are attached to the issue for context.

#### 4. Propose a fix (optional)

The AI analyzes the issue and proposes a potential solution. This is a suggestion, not an implementation — it gives maintainers a starting point and helps the reporter validate that their issue was understood correctly.

**Goal:** a proposed fix or approach is described in the issue body.

#### 5. Draft

The AI produces a well-structured issue with:
- Title
- Description
- Issue type and labels
- Reproduction steps (if applicable)
- Expected vs actual behavior (if applicable)
- Relevant code quotes (from step 3)
- Proposed fix (from step 4, if applicable)
- Backlink to the anchored document or code file

The user can review and iterate on the draft.

**Produces:** `issue` — an issue document ready for submission.

#### 6. Submit

The issue is posted to the project's configured issue tracker (GitHub Issues for MVP) with all structured metadata. A backlink from the Synchestra spec to the external issue is recorded.

Optionally, a linked task can be created in the state repo for tracking within Synchestra.

**Produces:** `issue-tracker-entry`, optionally `task`

### Workflow definition

```yaml
name: raise-issue
title: Raise an Issue
description: Report a problem or suggest an improvement
anchor-types: [feature, feature-section, proposal, code-file]
produces: [issue, issue-tracker-entry]
roles: ["*"]
context:
  load:
    - anchor-document
retention: archive
ui:
  icon: alert-circle
  variant: secondary
  sort-order: 20
steps:
  - name: understand
    prompt: gather-issue-details
    description: Understand the problem or suggestion
  - name: classify
    prompt: classify-issue
    description: Determine issue type and appropriate labels
  - name: find-relevant-code
    prompt: find-and-quote-relevant-code
    description: Search for and quote relevant code
    enabled: auto
  - name: propose-fix
    prompt: propose-a-fix
    description: Suggest a potential solution
    optional: true
  - name: draft
    prompt: draft-issue
    description: Write the issue document
    produces: [issue]
  - name: submit
    prompt: submit-issue
    description: Post the issue to the project tracker
    produces: [issue-tracker-entry]
```

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Chat](../../README.md) | This workflow runs on the chat layer. |
| [Workflow](../README.md) | This is one of the built-in workflows. |
| [Proposals](../../../proposals/README.md) | Issues can anchor to proposals. An issue about a proposal may lead to a revised proposal. |
| [Feature](https://github.com/synchestra-io/specscore/blob/main/spec/features/feature/README.md) | Issues can anchor to features and link back to them. |
| [Task Status Board](../../../task-status-board/README.md) | Optionally creates a linked task for tracking the issue within Synchestra. |

## Outstanding Questions

- Should the "propose a fix" step be able to escalate to the "Tweak Document" or "Create Proposal" workflow if the fix is straightforward enough?
- For private repos where code quoting is disabled, should the AI still be able to reference file paths and line numbers without quoting content?
- What issue tracker integrations should be supported beyond GitHub Issues (GitLab, Jira, Linear), and should this be part of the initial spec or deferred?
