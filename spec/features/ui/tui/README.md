# Feature: UI / TUI

**Status:** Conceptual

## Summary

The TUI is Synchestra's terminal-based user interface. It provides a structured, keyboard-driven way to navigate projects, inspect features and tasks, create proposals, create/enqueue tasks, and view workers, all through the Synchestra CLI.

## Problem

Some Synchestra users work primarily in terminals and want a richer interface than raw command invocations, but they should not need a separate graphical app to access project navigation and core workflows.

## Proposed Behavior

### Delivery model

The TUI is provided by the Synchestra [CLI](../../cli/README.md) feature.

It is a terminal UI surface, not a separate product. It should reuse Synchestra's existing project context, auth/session context, and command semantics where applicable.

### Entry flow

The TUI starts with the user's project list. Selecting a project opens the project menu.

The initial project menu contains:

- `Features`
- `Tasks`
- `Workers`

### Features flow

The `Features` screen shows root features for the selected project.

Selecting a feature opens a detail view that includes:

- The feature content
- The feature's proposals list
- An action to open the `New proposal` flow

The `New proposal` flow creates a change request for the selected feature and follows the [Proposals](../../proposals/README.md) rules, including optional GitHub issue creation for MVP.

### Tasks flow

The `Tasks` screen shows root tasks for the selected project.

For MVP, it supports:

- Viewing root tasks
- Creating tasks
- Enqueueing tasks

Because the TUI is part of the CLI surface, it should respect the same task state model and mutation rules as the underlying task commands.

### Workers flow

The `Workers` screen shows available workers that can execute tasks and run project validation.

The initial TUI requirement is visibility of worker options such as local, VM, Docker, and cloud-backed environments.

## Additional Rules

- The TUI should expose the same MVP information architecture as the web app even if terminal-specific interaction details differ.
- The TUI is part of the CLI feature, so any future TUI-specific commands or entrypoints must remain consistent with the CLI's overall contract.

## Outstanding Questions

None at this time.
