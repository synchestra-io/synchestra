# Feature: UI

**Status:** Conceptual

## Summary

The UI feature defines Synchestra's human-facing interfaces for browsing projects, inspecting project state, creating proposals, and creating/enqueuing tasks. It includes two delivery surfaces: a progressive web app and a terminal UI exposed through the Synchestra CLI.

## Contents

| Directory | Description |
|---|---|
| [web-app/](web-app/README.md) | Progressive web application built with TypeScript, Nx, Angular, and Ionic |
| [tui/](tui/README.md) | Terminal UI delivered through the Synchestra CLI |

### web-app

The web app is the primary graphical interface. It starts from a home page of projects, then lets the user navigate into project-level Features, Tasks, and Workers screens. It also provides the initial proposal-creation flow needed by the Proposals feature.

### tui

The TUI brings the same core project navigation and proposal/task flows into the terminal. It is part of the CLI feature rather than a separate runtime, giving terminal-first users a structured interface without requiring the web app.

## Problem

Synchestra already defines repository structure, task state, proposals, and CLI interactions, but humans still need an efficient way to navigate those concepts without reading raw Markdown trees or invoking low-level commands for every action.

Two gaps matter immediately:

- Proposal creation needs a user-facing flow from a feature screen
- Task creation and enqueueing need a project-level interface that does not require memorizing CLI syntax

Synchestra also needs a consistent place to show workers that can execute tasks and project validation jobs.

## Proposed Behavior

### Two UI surfaces

The UI feature has two subfeatures:

- A **web app** for browser and mobile-friendly access
- A **terminal UI** for keyboard-driven access inside the Synchestra CLI

The two surfaces should present the same high-value workflows even if they differ in layout and interaction details.

### Entry flow

The UI starts at a home screen that shows the list of projects the current user is working with.

Selecting a project opens a project menu with these initial entries:

- `Features`
- `Tasks`
- `Workers`

This is the minimum MVP project navigation surface for the UI feature.

### Features screen

The `Features` entry shows the list of root features for the selected project.

Selecting a feature opens that feature's detail screen. For the MVP, the feature screen must include:

- The feature's current content
- The feature's `Proposals` section
- An action to open a `New proposal` screen

The `New proposal` screen lets the user create a proposal change request for the selected feature. This flow must align with the [Proposals](../proposals/README.md) feature, including the ability to create and link a GitHub issue for MVP when requested.

The feature screen must preserve the Proposals rule that proposals are non-normative and are not part of the current feature spec unless incorporated into the feature's main `README.md`.

### Tasks screen

The `Tasks` entry shows the list of root tasks for the selected project.

For the MVP, the tasks UI must support:

- Viewing root tasks
- Creating a new task
- Enqueueing a task so it becomes available for agents to claim

The UI does not redefine task state or transitions. It must respect the existing task lifecycle and mutation rules defined by the CLI and task-related features.

### Workers screen

The `Workers` entry shows the list of workers available to the selected project.

For this feature, a worker is an execution environment that can run tasks and project validation, such as:

- VM
- Docker container
- Cloud worker
- Local machine

The initial screen requirement is visibility: users should be able to see which workers are available. Scheduling policy, provisioning, and worker lifecycle management remain outside this feature's MVP scope unless defined elsewhere.

### Relationship to existing features

The UI feature depends on existing and planned Synchestra capabilities:

- [Proposals](../proposals/README.md) for the feature-level change-request flow
- [CLI](../cli/README.md) for the TUI delivery surface and task mutation semantics
- Task-related features for listing, creating, and enqueueing tasks

The UI presents and triggers those workflows; it does not replace their underlying rules.

## Additional Rules

- The web app and TUI should expose the same MVP navigation concepts: projects, project menu, features, tasks, workers, and proposal/task creation flows.
- The TUI is provided by the CLI feature, not by a separate standalone runtime.
- The web app is the progressive web application surface and should be treated as the default graphical UI.

## Outstanding Questions

None at this time.
