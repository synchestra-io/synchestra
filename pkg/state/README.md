# pkg/state

Go interface definitions for the Synchestra state store — the pluggable abstraction layer for all project coordination state.

This package contains only interfaces, types, and constants. Implementations live in sub-packages (e.g., `gitstore/`).

## Usage

```go
// Construct a store (backend-specific)
store, err := gitstore.New(ctx, state.StoreOptions{...})

// Navigate to domain, then call operations
task, err := store.Task().Get(ctx, "implement-auth")
err = store.Task().Claim(ctx, "implement-auth", state.ClaimParams{Run: "run-1", Model: "claude-opus-4-6"})
err = store.Chat().Finalize(ctx, "chat-abc123")
config, err := store.Project().Config(ctx)
```

## Spec

See `spec/features/state-store/` for the full feature specification.

## Outstanding Questions

None at this time.
