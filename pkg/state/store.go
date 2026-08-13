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

	// Agent returns the agent-coordination sub-interface: effort, run,
	// worktree claim, message, activity, replication-journal access, cursor,
	// authority lease, and health. See AgentStore.
	Agent() AgentStore

	// Close releases any resources this Store's lifecycle owns — in
	// particular, flushing a pending group-commit batch on the Agent()
	// journal, if one was ever constructed (state-store/journal-batching).
	// A backend with nothing to flush treats this as a safe no-op. A caller
	// that owns a Store's lifecycle — a one-shot CLI invocation in
	// particular — must call Close before process exit; see AgentStore's
	// Close doc comment. Safe to call more than once.
	Close(ctx context.Context) error
}

// StoreFactory is a constructor function that each backend provides.
// The CLI selects the backend based on project configuration and calls
// the factory at startup.
type StoreFactory func(ctx context.Context, opts StoreOptions) (Store, error)
