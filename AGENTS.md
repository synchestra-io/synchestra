# Synchestra — AI Agent Rules

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

## Go validation after code changes

After any change to `.go` files, agents must run the full Go validation sequence before considering the task complete:

- `gofmt -w <changed-go-files>` (or `gofmt -w ./...` when appropriate)
- `golangci-lint run ./...`
- `go test ./...`
- `go build ./...`
- `go vet ./...`

If the user explicitly says to skip one of these checks, follow the user's instruction and say which validation was skipped.
