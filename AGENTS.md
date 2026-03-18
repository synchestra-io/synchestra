# Synchestra — AI Agent Rules

## Build, test, and lint commands

This repository contains both the Synchestra specification and the CLI implementation (Go). The standard build and validation commands are:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`

These are also run by the CI workflow in `.github/workflows/go-ci.yml`.

For documentation-only changes, validate the affected Markdown files against the surrounding specs and conventions.

## High-level architecture

This repository is both the specification and the CLI implementation for Synchestra:

- `spec/` is the technical source of truth. Use it for behavior, data model, task lifecycle, CLI semantics, and repository layout.
- `pkg/` contains the Go implementation of the CLI, organized by domain (`cli/`, `state/`).
- `main.go` is the CLI entry point.
- `skills/` packages the CLI into agent-facing skills. Each skill is a concrete wrapper around a single `synchestra` command and links back to the relevant CLI spec.
- `docs/` contains user-facing explanations of the platform and API surface. Use it when you need the conceptual stack or public interface rather than internal feature requirements.

Key specification files:

- `README.md` explains Synchestra as a git-backed coordination layer for multi-platform AI agents. It also defines the key ideas: hierarchical task trees, naming conventions as API, git as the database, token-efficient context loading, and claim-and-push concurrency.
- `spec/features/project-definition/README.md` defines the three repository types (state, spec, code), the two supported layouts for spec repositories (dedicated or multi-project), and the `synchestra-spec-repo.yaml` contract. The state repository (`{project}-synchestra`) is always separate and holds only tasks and coordination state.
- `spec/features/cli/README.md` defines the canonical CLI contract. The `synchestra` CLI is the shared interface for both agents and humans, and mutation commands are expected to perform atomic commit-and-push operations.
- `spec/features/agent-skills/README.md` defines how skills are structured and distributed. Skills do not replace the CLI; they standardize when to call it, which parameters to pass, and how to interpret exit codes.
- `spec/features/task-status-board/README.md` defines the markdown table claiming mechanism for optimistic locking, including the conflict resolution protocol for concurrent claims.
- `docs/features/README.md` captures the conceptual feature stack: state synchronization at the base, then agent coordination and progress reporting, then workflow orchestration, with human steering on top.
- `docs/api/README.md` documents the public REST API that mirrors the platform capabilities.

## Directory structure

- Every directory MUST have a `README.md` file, **except `.github/` itself** (where a `README.md` would override the root one on GitHub's repository page; see `.github/README.not.md`). Subdirectories under `.github/` (e.g., `.github/workflows/`, `.github/hooks/`) MUST still have a `README.md`.
- Every `README.md` MUST have an "Outstanding Questions" section. If there are none, it explicitly states "None at this time." — never omit the section.
- Every `README.md` that has child directories MUST include a brief summary (1–7 sentences) for each immediate child after the index table. This gives readers high-level context without requiring them to open each child.
- CLI arguments are documented in `_args/` directories under `spec/features/cli/`. Each argument has its own `.md` file at the level where it applies (global, command-group, or command-specific). See the [`_args` convention in the CLI spec](spec/features/cli/README.md#the-args-directory-convention) for the full format and placement rules.

## Go source file feature references

Every `.go` file MUST include structured comments near the top of the file (after the `package` declaration and before `import`) that reference:

1. **Features it implements or supports** — the spec features this file directly realizes.
2. **Features it depends on** — spec features whose behavior this file relies on at runtime.

Use the following format:

```go
// Features implemented: <feature-path>[, <feature-path>...]
// Features depended on:  <feature-path>[, <feature-path>...]
```

Feature paths are relative to `spec/features/` and use `/` separators (e.g., `cli/task/claim`, `chat/workflow/fast-path`). Omit the `spec/features/` prefix.

If a file implements features but depends on none, omit the "depended on" line. If a file only provides utilities consumed by feature code, omit the "implemented" line and list only dependencies.

### Example

```go
package task

// Features implemented: cli/task/claim, cli/task/update
// Features depended on:  state-sync/pull, project-definition/state-repo

import (
    ...
)
```

### Why

These annotations let agents and developers trace from source code back to specifications, understand cross-feature coupling, and assess the impact of spec changes on implementation.

## Diagrams in specifications

Use **mermaid diagrams** instead of ASCII art in all specification documents. Mermaid provides:
- Better visual clarity and maintainability
- Native support in GitHub markdown rendering
- Support for flowcharts, sequence diagrams, state diagrams, dependency graphs, and more

When adding or updating diagrams in specs, convert ASCII art to mermaid or create new diagrams using mermaid syntax.

## Development plans location and format

All development plans must be created in `spec/plans/` and follow the structure defined in [Development Plan specification](spec/features/development-plan/README.md).

- Plans start in `draft` status and follow the approval workflow: `draft` → `in_review` → `approved` → (optionally) `superseded`
- Once approved, plans are immutable — edits require creating a new superseding plan
- Plans live nowhere else — not in `docs/superpowers/`, not in project directories, not in temporary locations
- Use `synchestra plan create` to scaffold a new plan; use `synchestra plan submit` and `synchestra plan approve` for workflow progression

See the [Development Plan specification](spec/features/development-plan/README.md#behavior) for complete structure, field requirements, and task generation rules.

## Key conventions

When working in this repo, treat specs as normative and docs as explanatory. If a skill README and a feature spec disagree, reconcile the change against the CLI or feature spec instead of editing one file in isolation.

Skill directories follow a fixed pattern: one skill per CLI action, stored at `skills/{skill-name}/README.md`. When editing or adding a skill, preserve the established structure: when to use it, command, parameters, exit codes, examples, and notes.

The CLI contract is consistent across commands. Exit codes have shared meanings (`0` success, `1` conflict, `2` invalid arguments, `3` not found, `4` invalid state transition, `10+` unexpected error), and mutation commands are described as atomic commit-and-push operations while read commands pull first for freshness.

Task state names are canonical. Use the statuses defined in `spec/features/cli/README.md`: `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, and `aborted`. `abort_requested` is a flag, not a standalone status.

Some platform components (daemon, server, HTTP API, web UI) are implemented in separate repos. Before adding operational details for those, check whether the content belongs here as specification or in the appropriate runtime repo.

## Go validation after code changes

After any change to `.go` files, agents must run the full Go validation sequence before considering the task complete:

- `gofmt -w <changed-go-files>` (or `gofmt -w ./...` when appropriate)
- `golangci-lint run ./...`
- `go test ./...`
- `go build ./...`
- `go vet ./...`

If the user explicitly says to skip one of these checks, follow the user's instruction and say which validation was skipped.
