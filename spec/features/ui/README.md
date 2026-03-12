# Feature: UI

**Status:** Conceptual

## Summary

The UI feature defines Synchestra's shared information architecture for human-facing interfaces — the screens, navigation, and workflows that both the web app and the terminal UI implement. This document is the single source of truth for what the UI shows and what actions it supports; the subfeature specs ([web-app](web-app/README.md), [tui](tui/README.md)) define how each surface delivers it.

## Contents

| Directory | Description |
|---|---|
| [web-app/](web-app/README.md) | Progressive web application (TypeScript, Nx, Angular, Ionic) — browser and mobile surface |
| [tui/](tui/README.md) | Terminal UI delivered through the Synchestra [CLI](../cli/README.md) — keyboard-driven surface |

### web-app

The graphical surface. A progressive web application that communicates with the Synchestra backend via the [HTTP API](../../../docs/api/README.md). Implementation lives in the [synchestra-app](https://github.com/synchestra-io/synchestra-app) repository. Covers authentication, responsive layout, PWA capabilities (offline access, installability), and browser-specific interaction patterns.

### tui

The terminal surface. Delivered as part of the [CLI](../cli/README.md) feature — not a separate runtime. Mutations delegate to existing CLI commands ([`task create`](../cli/task/create/README.md), [`task enqueue`](../cli/task/enqueue/README.md), etc.). Covers terminal rendering, keyboard navigation, and constraints of a text-only environment.

## Problem

Synchestra defines repository structure, task state, proposals, and CLI commands, but three gaps remain for human users:

1. **Proposal creation** — the [Proposals](../proposals/README.md) feature requires a [UI flow](../proposals/README.md#synchestra-ui-behavior) from a feature screen to create, link, and manage proposals. No such flow exists yet.
2. **Task creation and enqueueing** — creating a task today requires knowing the CLI syntax ([`task create`](../cli/task/create/README.md), [`task enqueue`](../cli/task/enqueue/README.md)). Users need a project-level interface that exposes these actions directly.
3. **Worker visibility** — users need to see which execution environments are available to run tasks. Workers are not yet defined as a standalone feature; this spec introduces the concept at the UI level and defers the full worker lifecycle to a future feature spec.

The UI feature provides the [human-steering](../../../docs/features/human-steering.md) layer that sits on top of Synchestra's agent-first coordination primitives.

## Proposed Behavior

### Information architecture

Both surfaces implement the same navigation tree:

```
Home (project list)
 └─ Project menu
     ├─ Features
     │    └─ Feature detail
     │         └─ New proposal
     ├─ Tasks
     │    └─ New task
     └─ Workers
```

This is the MVP navigation surface. Additional entries (e.g., Settings, Agents, Audit log) are expected in later iterations.

### Home screen

Shows the list of [projects](../../../spec/project-definition/README.md) the current user is working with. Each project entry is derived from a `synchestra-project.yaml` file the user has access to.

Selecting a project opens the project menu.

### Project menu

The initial project menu contains:

| Entry | Description |
|---|---|
| Features | Root features for the selected project |
| Tasks | Root tasks for the selected project ([task-status-board](../task-status-board/README.md) format) |
| Workers | Execution environments available to the project |

### Features screen

Shows the list of root features for the selected project.

Selecting a feature opens the **feature detail screen**, which must include:

1. The feature's current specification content (rendered from its `README.md`)
2. The feature's **Proposals** section — the last *N* proposals ordered ascending by creation date, as defined in the [Proposals spec](../proposals/README.md#feature-readme-proposals-section). *N* is controlled by the project setting `proposals.feature_page.limit` (default 3).
3. An action to open the **New proposal** screen

The **New proposal** screen creates a proposal for the selected feature following the [Proposals UI behavior](../proposals/README.md#synchestra-ui-behavior) contract:
- Creates the proposal directory and `README.md`
- Updates the feature's `proposals/README.md` and the parent feature's Proposals section
- Optionally creates and links a GitHub issue (MVP tracker)
- Surfaces tracker errors explicitly rather than silently dropping the link

The feature screen must clearly distinguish current specification from non-normative proposals. Proposal content is excluded from "what is the system today?" workflows per the [Proposals interaction rules](../proposals/README.md#interaction-with-current-state-understanding).

### Tasks screen

Shows the root [task-status-board](../task-status-board/README.md) for the selected project — the same table format defined in that feature spec (Task, Status, Depends on, Branch, Agent, Requester, Time).

For MVP, the tasks screen supports:

| Action | Underlying operation |
|---|---|
| View root tasks | Read the task board from `tasks/README.md` |
| Create a task | [`synchestra task create`](../cli/task/create/README.md) / [synchestra-task-create skill](../../../skills/synchestra-task-create/README.md) |
| Enqueue a task | [`synchestra task enqueue`](../cli/task/enqueue/README.md) / [synchestra-task-enqueue skill](../../../skills/synchestra-task-enqueue/README.md) |

The UI does not redefine task state or transitions. It must respect the existing [task lifecycle](../task-status-board/README.md#status-lifecycle) and mutation rules defined by the [CLI task commands](../cli/task/README.md).

### Workers screen

Shows the execution environments available to the selected project.

For this feature, a **worker** is an environment that can run tasks and project validation. Examples:

- Local machine
- VM
- Docker container
- Cloud worker (e.g., GitHub Actions runner, remote VM)

The initial requirement is **visibility** — users see which workers exist and their current state. Scheduling policy, provisioning, and worker lifecycle management are out of scope for this feature's MVP.

> **Note:** Workers are not yet defined as a standalone feature spec. This screen introduces the concept at the UI level. A dedicated `spec/features/workers/` feature should be created to define the worker data model, registration, health checking, and scheduling before the workers screen can move beyond visibility.

### Dependencies on other features

The UI feature presents and triggers workflows defined elsewhere — it does not replace their underlying rules.

| Dependency | Role |
|---|---|
| [Proposals](../proposals/README.md) | Feature-level change-request flow and UI behavior contract |
| [CLI](../cli/README.md) | TUI delivery surface; task mutation semantics; command contract |
| [Task Status Board](../task-status-board/README.md) | Board format, status lifecycle, claiming protocol |
| [Agent Skills](../agent-skills/README.md) | Skills that back the task actions (create, enqueue) |
| [Project Definition](../../../spec/project-definition/README.md) | `synchestra-project.yaml` — source of the project list |
| [HTTP API](../../../docs/api/README.md) | Backend for the web app surface |

## Outstanding Questions

- Should the workers screen be deferred entirely until a `workers` feature spec is defined, or is UI-level visibility enough for an MVP?
- How should the home screen discover projects — scan local filesystem for `synchestra-project.yaml` files, query the API for a user's project list, or both depending on the surface?
- Should the task-status-board rendering in the UI be a generic markdown table renderer, or should tasks have a purpose-built component?
- What authentication model does the UI use? The [API](../../../docs/api/README.md) requires Bearer tokens created via `synchestra auth token create`. The root README mentions GitHub OAuth and Firebase. Which applies to each surface?
- Should the feature detail screen support navigating into sub-features, or only root features for MVP?
