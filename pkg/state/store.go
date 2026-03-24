package state

// Features implemented: state-store

import "context"

// Store is the top-level state store interface. Consumers navigate to a
// domain (Task, Chat, Project) and then call operations on the sub-interface.
//
// Example:
//
//	store.Task().Claim(ctx, "implement-auth", state.ClaimParams{Run: "run-1", Model: "claude-opus-4-6"})
//	store.Chat().Finalize(ctx, "chat-abc123")
//	store.Project().Config(ctx)
type Store interface {
	// Task returns the task sub-interface for task lifecycle operations.
	Task() TaskStore

	// Chat returns the chat sub-interface for chat operations.
	Chat() ChatStore

	// Project returns the project sub-interface for project configuration.
	Project() ProjectStore

	// State returns the sync sub-interface for manual sync controls.
	State() StateSync
}

// StoreFactory is a constructor function that each backend provides.
// The CLI selects the backend based on project configuration and calls
// the factory at startup.
type StoreFactory func(ctx context.Context, opts StoreOptions) (Store, error)
