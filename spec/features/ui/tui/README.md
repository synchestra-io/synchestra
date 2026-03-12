# Feature: UI / TUI

**Status:** Conceptual

## Summary

The TUI is Synchestra's terminal-based user interface. It implements the [shared information architecture](../README.md#information-architecture) defined in the parent UI feature spec, delivered through the Synchestra [CLI](../../cli/README.md) as an interactive, keyboard-driven surface.

This document covers what is **unique to the terminal surface** — CLI integration model, rendering approach, and mutation semantics. For the screens and navigation tree, see the [parent UI spec](../README.md#proposed-behavior).

## Problem

Terminal-first users (developers, DevOps engineers, agents operating in headless mode) need structured project navigation without leaving the terminal. Raw CLI commands ([`task list`](../../cli/task/list/README.md), [`task create`](../../cli/task/create/README.md)) work for individual operations, but navigating features, browsing proposals, and managing tasks benefits from an interactive menu-driven interface.

## Proposed Behavior

### Delivery model

The TUI is part of the [CLI](../../cli/README.md) feature — not a separate binary or runtime. It is invoked through the CLI and shares the CLI's project context, configuration, and authentication.

The entry point is a CLI command (e.g., `synchestra ui` or `synchestra tui`). The exact command name is an open question, but it must fit within the existing [`synchestra <resource> <action>` command hierarchy](../../cli/README.md).

### Data source

Unlike the [web app](../web-app/README.md), which talks to the [HTTP API](../../../../docs/api/README.md), the TUI operates on the **local git repository** directly. It reads project definitions, feature specs, task boards, and proposals from the filesystem. This means:

- The TUI works offline (no API server required)
- Data freshness depends on the last `git pull`
- The TUI may offer a "refresh" action that runs `git pull` under the hood

For mutations, the TUI should also support API-based operation when a Synchestra server is available, but local-first / git-based operation is the default.

### Mutation model

When the user performs a write action (create task, enqueue task, create proposal), the TUI delegates to existing CLI commands:

| Action | CLI command | Skill |
|---|---|---|
| Create task | [`synchestra task create`](../../cli/task/create/README.md) | [synchestra-task-create](../../../../skills/synchestra-task-create/README.md) |
| Enqueue task | [`synchestra task enqueue`](../../cli/task/enqueue/README.md) | [synchestra-task-enqueue](../../../../skills/synchestra-task-enqueue/README.md) |
| Create proposal | Via [Proposals UI behavior](../../proposals/README.md#synchestra-ui-behavior) | — |

This ensures all mutations go through the same atomic commit-and-push flow defined in the CLI spec, preserving the git-as-source-of-truth model.

### Terminal rendering

The TUI renders to a terminal using a full-screen interactive layout (similar to tools like `lazygit`, `k9s`, or `gh dash`). Key considerations:

- **Keyboard navigation** — arrow keys, Enter to select, Escape to go back, `/` to search
- **No mouse required** — all actions reachable via keyboard
- **Responsive to terminal size** — adapts to narrow (80-col) and wide terminals
- **Color and formatting** — uses ANSI colors; degrades gracefully in no-color mode (`NO_COLOR` env var)

The specific rendering library (e.g., Go's `bubbletea`, `tview`, or `tcell`) is an implementation decision for [synchestra-go](https://github.com/synchestra-io/synchestra-go); this spec does not prescribe it.

### Surface-specific interaction notes

- **Feature content** is rendered from Markdown. The TUI should use a terminal Markdown renderer (e.g., `glamour`) to present feature specs and proposals with formatting.
- **Task board** renders the [task-status-board](../../task-status-board/README.md) table with status emojis and aligned columns.
- **Proposal list** on the feature screen uses the same ascending-date order and configurable limit as defined in the [parent spec](../README.md#features-screen).

## Outstanding Questions

- What is the CLI entry point for the TUI — `synchestra ui`, `synchestra tui`, or something else? Should it also support `synchestra` with no arguments launching the TUI by default?
- Should the TUI support the API as an alternative data source (for remote projects not cloned locally)?
- Which Go terminal UI library should be used? This affects the interaction model and visual capabilities.
- How should the TUI handle long-running mutations (e.g., `task create` with `--enqueue` that does commit-and-push)? Show a spinner? Stream CLI output?
- Should the TUI be available to agents in headless mode, or is it strictly for human interactive use?
