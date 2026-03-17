# Feature: State Store

**Status:** Conceptual

## Summary

The state store is the abstraction layer for all Synchestra project coordination state. It defines a composable, hierarchical Go interface (`state.Store`) that formally specifies every operation the system can perform on project state — tasks, artifacts, chat, and project configuration.

The default implementation is a git-backed store (`gitstore`) that maps directly to the current [state repository](../../architecture/repository-types.md#state-repository) design. Future implementations (SQLite, PostgreSQL, cloud databases) can be added by satisfying the same interface.

## Problem

The current architecture references the "state repository" as both the abstraction and the implementation. This conflation creates two issues:

1. **Vocabulary lock-in.** Every spec document says "state repo" when it means "where coordination state lives." This makes it harder to reason about alternative backends.
2. **No formal contract.** The set of operations on project state is defined implicitly across CLI commands, OpenAPI endpoints, and spec prose. There is no single, compilable source of truth for what the state layer supports.

## Design Principles

- **Hierarchical composition.** The interface is navigated like CLI subcommands: `store.Task().Claim(ctx, ...)`, not `store.ClaimTask(ctx, ...)`. This keeps autocomplete focused and each sub-interface small.
- **Explicit status transitions.** Each task lifecycle transition is its own method (`Claim`, `Start`, `Complete`, `Fail`, etc.) rather than a generic `UpdateStatus`. Invalid transitions are unrepresentable at the interface level.
- **Context on all leaf methods.** Every method that performs I/O takes `context.Context` as the first argument, enabling cancellation, timeouts, and retry policies.
- **Atomicity is a backend concern.** Operations like `Claim` must be atomic (exactly one caller succeeds), but the interface does not prescribe how. Git uses push-or-fail; SQL uses `UPDATE ... WHERE`; each backend chooses its mechanism.
- **Interface-only package.** `pkg/state/` contains interfaces, types, and constants. Implementations live in sub-packages (`pkg/state/gitstore/`, `pkg/state/sqlitestore/`, etc.).

## Package Structure

```
pkg/state/
  store.go           # state.Store interface
  task.go            # state.TaskStore, state.Board, state.ArtifactStore
  chat.go            # state.ChatStore
  project.go         # state.ProjectStore
  types.go           # shared types (TaskStatus, Task, Chat, etc.)
  gitstore/          # Default: git-backed implementation
  sqlitestore/       # Future: single-host SQLite
  pgstore/           # Future: PostgreSQL
```

## Top-Level Interface

```go
package state

type Store interface {
    Task() TaskStore
    Chat() ChatStore
    Project() ProjectStore
}
```

Consumers navigate to a domain first, then call operations. This mirrors the CLI command hierarchy and keeps each interface focused.

## Sub-Features

| Sub-feature | Description |
|---|---|
| [Task Store](task-store/) | Task lifecycle, status transitions, claiming, board views, and artifact storage |
| [Chat Store](chat-store/) | Chat lifecycle, message history, and chat artifacts |
| [Project Store](project-store/) | Project configuration and README generation |
| [Backends](backends/) | Pluggable implementations of the `state.Store` interface |

### Task Store

Manages the full task lifecycle — creation, status transitions, claiming with atomic guarantees, board rendering, and artifact storage. Board and artifact access are nested under the task namespace: `store.Task().Board()` and `store.Task().Artifact(ctx, slug)`. See [Task Store](task-store/).

### Chat Store

Manages chat lifecycle (create, finalize, abandon), append-only message history, and chat-scoped artifacts. Reuses the same `ArtifactStore` interface as tasks. See [Chat Store](chat-store/).

### Project Store

Thin interface for project-level configuration (the `synchestra-state.yaml` back-reference) and auto-generated README rebuilding. See [Project Store](project-store/).

### Backends

Registry of `state.Store` implementations. Each backend satisfies the full interface using its native storage and concurrency primitives. The git backend is the default and reference implementation; future backends (SQLite, PostgreSQL, cloud) live alongside it. See [Backends](backends/).

### Git Backend

The default `state.Store` implementation. Maps every interface method to git operations in the state repository — file writes, markdown table updates, atomic commit-and-push. See [Git Backend](backends/git/).

## Backend Matrix

| Backend | Use Case | Atomicity Mechanism | Status |
|---|---|---|---|
| **Git** (`gitstore`) | Default, works everywhere | Push-or-fail | Default implementation |
| **SQLite** (`sqlitestore`) | Single-host, high performance | Row-level locking | Future |
| **PostgreSQL** (`pgstore`) | Multi-host, K8s clusters | `UPDATE ... WHERE` | Future |
| **Cloud DB** | Managed cloud deployments | Provider-specific | Future |
| **Custom** | User-provided `state.Store` | User-defined | Supported via interface |

## Construction

```go
package state

type StoreFactory func(ctx context.Context, opts StoreOptions) (Store, error)

type StoreOptions struct {
    SpecRepoPaths []string
    StateRepoPath string
}
```

Each backend provides a `StoreFactory`. The CLI selects the backend based on project configuration and constructs the store at startup.

## Outstanding Questions

- Should there be a read-only `StoreReader` interface for consumers that only need to query state (e.g., dashboard views, derived plan status)?
- How should the store handle migrations when the state schema evolves (e.g., new fields on tasks)?
- Is a caching layer in front of the store useful? A cache introduces consistency risks with the atomic claim protocol — stale reads could cause phantom claims. If pursued, it should default to passthrough (no caching) and require explicit opt-in.
- Should `StoreOptions` include backend-specific configuration (connection strings, etc.), or should each backend define its own options type?
