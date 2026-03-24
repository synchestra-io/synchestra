package state

// Features implemented: state-store

import "context"

// StateSync provides manual synchronization controls for the state store.
// Backends without a remote concept (e.g., SQLite) return a no-op implementation.
type StateSync interface {
	// Pull fetches the latest state from the remote.
	Pull(ctx context.Context) error

	// Push sends local state to the remote.
	Push(ctx context.Context) error

	// Sync performs a full round-trip: pull then push.
	Sync(ctx context.Context) error
}
