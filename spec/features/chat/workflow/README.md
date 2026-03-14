# Feature: Workflow

**Status:** Conceptual

## Summary

A workflow is a declarative YAML recipe that defines what happens when a user initiates an action on a document. It specifies what context to load, what AI prompt/skill to use, what steps to follow, what artifacts can be produced, and who can use it. Workflows are the bridge between user-facing actions ("Create a Proposal") and the [chat](../README.md) mechanics underneath.

Synchestra ships with a predefined set of built-in workflows that are useful out of the box and showcase the system's capabilities. Projects can customize these workflows — adjusting prompts, adding validation checks, configuring rules, and tuning steps — to match their specific contribution guidelines and quality standards.

## Contents

| Directory | Description |
|---|---|
| [create-proposal/](create-proposal/README.md) | Workflow for brainstorming and drafting change requests against existing features |
| [create-feature/](create-feature/README.md) | Workflow for creating new feature specifications from scratch |
| [raise-issue/](raise-issue/README.md) | Workflow for articulating and submitting well-structured issues to the project's tracker |
| [tweak-document/](tweak-document/README.md) | Workflow for quick fixes — typos, formatting, small code changes — with minimal ceremony |

### create-proposal

Guides a user through brainstorming, exploring approaches, and drafting a formal change request (proposal) for an existing feature. Supports a fast path for maintainers where the change is implemented immediately and a development plan is produced as a report.

### create-feature

Guides a user through defining a new feature specification from scratch. Creates the feature document in a new branch, iterates on the spec within the chat, and submits it for review.

### raise-issue

Helps a user articulate a problem or suggestion clearly, classifies the issue type, optionally finds and quotes relevant code, proposes a fix, and submits a well-structured issue to the project's issue tracker.

### tweak-document

Handles quick fixes to spec documents or code files — typos, formatting, small wording or code changes. Behavior varies by user role: maintainers can direct-commit spec changes, while contributors create PRs. Code changes always go through PRs with CI validation.

## Problem

User-facing actions ("Create a Proposal," "Raise an Issue") need to be translated into chat behavior — what the AI should do, what context it needs, what it can produce. Without a structured way to define this mapping, each action would require custom code, making the system rigid and project-agnostic.

Workflows solve this by providing a declarative format that separates **what and who** (the YAML configuration) from **how** (the AI skills/prompts). This makes the system both useful out of the box and customizable per project.

## Design Philosophy

**Useful out of the box, fully customizable.** Synchestra ships with built-in workflows that demonstrate best practices and provide immediate value. Projects customize the substance (questions to ask, rules to enforce, checks to run) without changing the structure.

**Declarative over imperative.** Workflows are YAML configuration, not code. This makes them readable, validatable, and accessible to non-developers. Complex logic lives in the referenced skills/prompts, not in the workflow definition.

**Extensible by design.** The workflow schema is designed so that user-defined custom workflows can be added in the future without breaking existing workflows. Custom workflows are out of scope for v1 but architecturally accounted for.

## Behavior

### Workflow location

Built-in workflow definitions ship with Synchestra. Project-level overrides and customizations live in `synchestra-project.yaml`.

The canonical workflow specs live under `spec/features/chat/workflow/` in the Synchestra spec repo.

### Workflow definition schema

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
    prompt: self-review
    description: Review the draft for completeness
    optional: true
```

### Schema fields

#### Top-level fields

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Unique identifier, used in URLs, API, and CLI |
| `title` | yes | — | Human-readable label for the UI button |
| `description` | yes | — | Short description shown in UI tooltips or action menus |
| `anchor-types` | yes | — | Document types this workflow can attach to (e.g., `feature`, `feature-section`, `code-file`) |
| `prompt` | conditional | — | AI prompt/skill name (required if no `steps` defined) |
| `produces` | yes | — | Artifact types this workflow can create |
| `roles` | yes | — | Who can use this workflow (`["*"]` = everyone) |
| `context.load` | no | `[anchor-document]` | Additional context documents to load alongside the anchor |
| `retention` | no | project default | What happens to chat on finalize: `archive`, `summarize`, `dispose` |

#### UI fields

| Field | Required | Default | Description |
|---|---|---|---|
| `ui.icon` | no | none | Icon identifier from a standard icon set (e.g., Lucide) |
| `ui.variant` | no | `secondary` | Button style controlling color and prominence: `primary`, `secondary`, `subtle`, `warning` |
| `ui.sort-order` | no | `100` | Ordering among buttons on the same document (lower = appears first) |

#### Step fields

| Field | Required | Default | Description |
|---|---|---|---|
| `steps[].name` | yes | — | Step identifier |
| `steps[].prompt` | yes | — | AI prompt/skill for this phase |
| `steps[].description` | yes | — | Shown to user as progress indicator |
| `steps[].produces` | no | — | Artifact types produced at this step |
| `steps[].optional` | no | `false` | Whether the user can skip this step |
| `steps[].enabled` | no | `true` | Whether this step is active. Accepts `true`, `false`, or `auto` (system decides based on context, e.g., repo visibility) |

### Simple vs multi-step workflows

A workflow can use either a single top-level `prompt` (simple) or a `steps` array (multi-step). A single `prompt` is equivalent to a one-step workflow.

The conversation is continuous across steps — the user does not see hard boundaries. The server transitions between steps based on the AI's assessment of when the current step's goal is met. The UI may show a subtle progress indicator.

### Future extension: paths

When role-based routing is needed, the flat `prompt`/`steps` + `roles` can be replaced with `paths`:

```yaml
paths:
  fast:
    roles: [maintainer]
    steps:
      - name: understand
        prompt: gather-requirements
      - name: implement
        prompt: fast-implement
        produces: [proposal, development-plan, pull-request]
  standard:
    roles: ["*"]
    steps:
      - name: understand
        prompt: gather-requirements
      - name: explore
        prompt: brainstorm-approaches
      - name: draft
        prompt: draft-proposal
        produces: [proposal]
```

This is architecturally accounted for but **not implemented in v1.** The schema is designed so adding `paths` is backward-compatible — a flat workflow is equivalent to a single-path workflow.

### Discovery

The web UI reads available workflows for a given document and renders them as action buttons. Which buttons appear depends on:

1. **Document type** — must match the workflow's `anchor-types`
2. **User role** — must match the workflow's `roles`
3. **UI configuration** — buttons are ordered by `ui.sort-order` and styled by `ui.variant`

For entity creation (e.g., "New feature" on the features index page), the UI renders buttons for workflows that have `allow-create: true` for that entity type.

### Project customization

Projects customize built-in workflows in `synchestra-project.yaml`. The built-in workflows define the structure (steps, order, artifact types); the project configures the substance (questions, rules, checks).

```yaml
workflows:
  create-proposal:
    context:
      guidelines: docs/contributing/proposal-guidelines.md
    checks:
      - name: spell-check
        run: npx cspell --no-progress {artifact}
      - name: link-validator
        run: markdown-link-check {artifact}
      - name: custom-rules
        run: ./scripts/validate-proposal.sh {artifact}
    prompts:
      additional-questions:
        - "Which users or personas does this change affect?"
        - "Have you checked the roadmap for conflicts?"
      rules:
        - "All proposals must specify backward compatibility impact"
        - "Proposals touching the API must include OpenAPI schema changes"
```

#### Customization fields

| Field | Description |
|---|---|
| `context.guidelines` | Project-specific docs loaded alongside the anchor as additional context |
| `checks` | Commands run against produced artifacts before finalization — like GitHub Actions pre-merge checks. If a check fails, the user sees the output and can iterate. |
| `prompts.additional-questions` | Questions the AI must ask during conversational steps. The AI weaves them naturally into the conversation, not as a checklist dump. |
| `prompts.rules` | Constraints the AI enforces during drafting and review steps. The AI validates these before allowing finalization. |

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Chat](../README.md) | Workflows configure the chat layer. Each chat instance runs exactly one workflow. |
| [Agent Skills](../../agent-skills/README.md) | Workflow steps reference skills/prompts from the skill infrastructure. |
| [UI](../../ui/README.md) | The web UI renders workflow buttons based on `anchor-types`, `roles`, and `ui` configuration. |
| [Project Definition](../../project-definition/README.md) | Workflow customization lives in `synchestra-project.yaml`. |
| [CLI](../../cli/README.md) | `synchestra workflow list` shows available workflows. Workflows are primarily a web UI concept. |

## Outstanding Questions

- Should custom workflows (user-defined YAML) be validated against a JSON Schema, and if so, where does that schema live?
- How should workflow versioning work — when Synchestra ships an updated built-in workflow, how does it interact with project-level customizations?
- Should there be a `synchestra workflow test` command that simulates a workflow run for validation purposes?
- What is the exact set of `anchor-types` values — is it a fixed enum or extensible?
