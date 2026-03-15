# Synchestra — AI Agent Rules

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

## Go validation after code changes

After any change to `.go` files, agents must run the full Go validation sequence before considering the task complete:

- `gofmt -w <changed-go-files>` (or `gofmt -w ./...` when appropriate)
- `golangci-lint run ./...`
- `go test ./...`
- `go build ./...`
- `go vet ./...`

If the user explicitly says to skip one of these checks, follow the user's instruction and say which validation was skipped.

