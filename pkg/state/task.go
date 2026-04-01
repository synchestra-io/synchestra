package state

// Features implemented: state-store, state-store/task-store

import "context"

// TaskStore defines operations on tasks — creation, querying, status
// transitions, and access to nested board and artifact sub-interfaces.
type TaskStore interface {
	// Create creates a new task in planning status.
	Create(ctx context.Context, params TaskCreateParams) (CoordinatedTask, error)

	// Get returns a task by its slug. Returns ErrNotFound if the task does not exist.
	Get(ctx context.Context, slug string) (CoordinatedTask, error)

	// List returns tasks matching the given filter.
	List(ctx context.Context, filter TaskFilter) ([]CoordinatedTask, error)

	// Enqueue transitions a task from planning to queued.
	Enqueue(ctx context.Context, slug string) error

	// Claim atomically transitions a task from queued to claimed.
	// If two agents call Claim concurrently, exactly one succeeds;
	// the other receives ErrConflict.
	Claim(ctx context.Context, slug string, claim ClaimParams) error

	// Start transitions a task from claimed to in_progress.
	Start(ctx context.Context, slug string) error

	// Complete transitions a task from in_progress to completed.
	Complete(ctx context.Context, slug string, summary string) error

	// Fail transitions a task from in_progress to failed.
	Fail(ctx context.Context, slug string, reason string) error

	// Block transitions a task from in_progress to blocked.
	Block(ctx context.Context, slug string, reason string) error

	// Unblock transitions a task from blocked to in_progress.
	Unblock(ctx context.Context, slug string) error

	// Release transitions a task from claimed back to queued.
	Release(ctx context.Context, slug string) error

	// RequestAbort sets the abort_requested flag on a claimed or in_progress task.
	RequestAbort(ctx context.Context, slug string) error

	// ConfirmAbort transitions a task from claimed or in_progress to aborted.
	ConfirmAbort(ctx context.Context, slug string) error

	// Board returns the board sub-interface for rendered task board views.
	Board() Board

	// Artifact returns the artifact sub-interface scoped to the given task.
	Artifact(ctx context.Context, taskSlug string) ArtifactStore
}

// Board defines operations on the rendered task board view.
type Board interface {
	// Rebuild regenerates the board from all task records.
	Rebuild(ctx context.Context) error

	// Get returns the current board as structured data.
	Get(ctx context.Context) (BoardView, error)
}

// ArtifactStore defines operations on named artifacts scoped to a task or chat.
type ArtifactStore interface {
	// Put stores an artifact with the given name and data.
	Put(ctx context.Context, name string, data []byte) error

	// Get retrieves an artifact by name. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, name string) ([]byte, error)

	// List returns references to all artifacts in this scope.
	List(ctx context.Context) ([]ArtifactRef, error)
}
