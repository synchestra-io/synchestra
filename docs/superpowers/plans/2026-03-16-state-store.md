# State Store Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define the compilable Go interface package (`pkg/state/`) that formalizes all Synchestra project state operations.

**Architecture:** A `pkg/state/` package containing only interfaces, types, and constants — no implementations. The package defines `state.Store` as the top-level interface with hierarchical sub-interfaces (`TaskStore`, `ChatStore`, `ProjectStore`) accessed via accessor methods. Each sub-interface is in its own file. A placeholder `gitstore` package is stubbed to verify the interfaces are implementable.

**Tech Stack:** Go 1.26, standard library only (no external dependencies for interface definitions).

**Spec:** `spec/features/state-store/README.md` and sub-feature READMEs.

**AGENTS.md rules to follow:**
- Every `.go` file must have `// Features implemented:` / `// Features depended on:` comments after the `package` declaration
- After any change to `.go` files: `gofmt -w`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`, `go vet ./...`
- Every directory MUST have a `README.md` with an "Outstanding Questions" section

**Note:** `errors.go` is a plan-level addition not in the spec's package structure listing, justified by the error semantics described in the spec prose (ErrNotFound, ErrConflict, ErrInvalidTransition).

---

## File Structure

```
pkg/
  state/
    README.md       — package documentation
    store.go        — state.Store interface (top-level, accessor methods only)
    task.go         — state.TaskStore, state.Board, state.ArtifactStore interfaces
    chat.go         — state.ChatStore interface
    project.go      — state.ProjectStore interface
    types.go        — all shared types: Task, Chat, ProjectConfig, TaskStatus, ChatStatus, etc.
    errors.go       — sentinel errors for invalid transitions, not-found, conflict
    gitstore/
      README.md     — package documentation
      gitstore.go   — stub GitStateStore struct satisfying state.Store (returns not-implemented errors)
```

Each file has one responsibility. Interfaces are in separate files from types to keep them focused.

---

## Chunk 1: Interface Package

### Task 1: Create all `pkg/state/` interface and type files

All files in the `state` package are created together since they form a single compilation unit and cannot be validated individually.

**Files:**
- Create: `pkg/state/README.md`
- Create: `pkg/state/types.go`
- Create: `pkg/state/errors.go`
- Create: `pkg/state/store.go`
- Create: `pkg/state/task.go`
- Create: `pkg/state/chat.go`
- Create: `pkg/state/project.go`

- [ ] **Step 1: Create `pkg/state/README.md`**

```markdown
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
```

- [ ] **Step 2: Create `pkg/state/types.go`**

```go
package state

// Features implemented: state-store

import "time"

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPlanning   TaskStatus = "planning"
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusClaimed    TaskStatus = "claimed"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusAborted    TaskStatus = "aborted"
)

// Task represents a unit of work in the state store.
type Task struct {
	Slug      string
	Title     string
	Status    TaskStatus
	Parent    string // parent task slug, empty for root tasks
	DependsOn []string
	Run       string // agent run ID (populated when claimed/in_progress)
	Model     string // agent model ID (populated when claimed/in_progress)
	Requester string
	Reason    string // block/fail/abort reason
	Summary   string // completion summary
	CreatedAt time.Time
	ClaimedAt *time.Time
	UpdatedAt time.Time
}

// TaskCreateParams holds parameters for creating a new task.
type TaskCreateParams struct {
	Slug      string
	Title     string
	Parent    string
	DependsOn []string
	Requester string
}

// ClaimParams holds parameters for claiming a task.
type ClaimParams struct {
	Run   string
	Model string
}

// TaskFilter holds optional filters for listing tasks.
// Nil pointer fields mean "don't filter on this field."
type TaskFilter struct {
	Status *TaskStatus
	Parent *string
}

// BoardView represents a rendered task board.
type BoardView struct {
	Rows []BoardRow
}

// BoardRow represents a single row in the task board.
type BoardRow struct {
	Task      string
	Status    TaskStatus
	DependsOn []string
	Branch    string
	Agent     string
	Requester string
	StartedAt *time.Time
	Duration  *time.Duration
}

// ChatStatus represents the lifecycle state of a chat.
type ChatStatus string

const (
	ChatStatusCreated   ChatStatus = "created"
	ChatStatusActive    ChatStatus = "active"
	ChatStatusFinalized ChatStatus = "finalized"
	ChatStatusAbandoned ChatStatus = "abandoned"
)

// Chat represents a conversational session in the state store.
type Chat struct {
	ID        string
	Anchor    string // what the chat is about
	Workflow  string // workflow name
	Status    ChatStatus
	User      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatCreateParams holds parameters for creating a new chat.
type ChatCreateParams struct {
	Anchor   string
	Workflow string
	User     string
}

// ChatFilter holds optional filters for listing chats.
type ChatFilter struct {
	Status *ChatStatus
}

// ChatMessage represents a single message in a chat history.
type ChatMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// ArtifactRef describes a named artifact without its content.
type ArtifactRef struct {
	Name string
	Size int64
}

// ProjectConfig holds the project-level configuration stored in the state store.
type ProjectConfig struct {
	Title    string
	SpecRepo string
}

// StoreOptions holds configuration for constructing a Store.
type StoreOptions struct {
	SpecRepoPath  string
	StateRepoPath string
}
```

- [ ] **Step 3: Create `pkg/state/errors.go`**

```go
package state

// Features implemented: state-store

import "errors"

var (
	// ErrNotFound is returned when a task, chat, or artifact does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInvalidTransition is returned when a status transition is not allowed
	// from the current state (e.g., calling Complete on a queued task).
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrConflict is returned when a concurrent modification prevents the
	// operation (e.g., two agents claiming the same task).
	ErrConflict = errors.New("conflict")
)
```

- [ ] **Step 4: Create `pkg/state/store.go`**

```go
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
}

// StoreFactory is a constructor function that each backend provides.
// The CLI selects the backend based on project configuration and calls
// the factory at startup.
type StoreFactory func(ctx context.Context, opts StoreOptions) (Store, error)
```

- [ ] **Step 5: Create `pkg/state/task.go`**

```go
package state

// Features implemented: state-store, state-store/task-store

import "context"

// TaskStore defines operations on tasks — creation, querying, status
// transitions, and access to nested board and artifact sub-interfaces.
type TaskStore interface {
	// Create creates a new task in planning status.
	Create(ctx context.Context, params TaskCreateParams) (Task, error)

	// Get returns a task by its slug. Returns ErrNotFound if the task does not exist.
	Get(ctx context.Context, slug string) (Task, error)

	// List returns tasks matching the given filter.
	List(ctx context.Context, filter TaskFilter) ([]Task, error)

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
```

- [ ] **Step 6: Create `pkg/state/chat.go`**

```go
package state

// Features implemented: state-store, state-store/chat-store

import "context"

// ChatStore defines operations for chat lifecycle management,
// append-only message history, and chat-scoped artifacts.
type ChatStore interface {
	// Create creates a new chat in created status. The store generates the chat ID.
	Create(ctx context.Context, params ChatCreateParams) (Chat, error)

	// Get returns a chat by its ID. Returns ErrNotFound if the chat does not exist.
	Get(ctx context.Context, chatID string) (Chat, error)

	// List returns chats matching the given filter.
	List(ctx context.Context, filter ChatFilter) ([]Chat, error)

	// Finalize transitions a chat to finalized, flushing message history
	// to durable storage.
	Finalize(ctx context.Context, chatID string) error

	// Abandon transitions a chat to abandoned.
	Abandon(ctx context.Context, chatID string) error

	// AppendMessages appends messages to the chat history.
	// On the first call for a created chat, this implicitly transitions
	// the chat status to active.
	AppendMessages(ctx context.Context, chatID string, messages []ChatMessage) error

	// Messages returns the full message history for a chat.
	Messages(ctx context.Context, chatID string) ([]ChatMessage, error)

	// Artifact returns the artifact sub-interface scoped to the given chat.
	Artifact(ctx context.Context, chatID string) ArtifactStore
}
```

- [ ] **Step 7: Create `pkg/state/project.go`**

```go
package state

// Features implemented: state-store, state-store/project-store

import "context"

// ProjectStore defines operations on project-level state —
// configuration back-references and README generation.
type ProjectStore interface {
	// Config returns the project configuration from the state store.
	Config(ctx context.Context) (ProjectConfig, error)

	// UpdateConfig writes updated project configuration.
	UpdateConfig(ctx context.Context, config ProjectConfig) error

	// RebuildREADME regenerates the auto-generated project overview.
	RebuildREADME(ctx context.Context) error
}
```

- [ ] **Step 8: Run full Go validation**

```bash
gofmt -w pkg/state/
golangci-lint run ./pkg/state/...
go build ./pkg/state/...
go vet ./pkg/state/...
```

Expected: All pass. (No `go test` yet — no test files in this package.)

- [ ] **Step 9: Fix any issues, then commit**

```bash
git add pkg/state/README.md pkg/state/types.go pkg/state/errors.go pkg/state/store.go pkg/state/task.go pkg/state/chat.go pkg/state/project.go
git commit -m "feat(state): define state store interfaces and types

Introduces pkg/state/ with the full state.Store interface hierarchy:
- Store (top-level) with Task(), Chat(), Project() accessors
- TaskStore with explicit status transition methods and nested Board/ArtifactStore
- ChatStore with append-only message history
- ProjectStore for config and README generation
- Shared types and sentinel errors"
```

---

## Chunk 2: Git Backend Stub

### Task 2: Create `pkg/state/gitstore/` — stub implementation and interface verification test

This stub verifies that the interfaces are implementable. Every method returns `errNotImplemented`. The real git backend implementation is a separate plan.

**Files:**
- Create: `pkg/state/gitstore/README.md`
- Create: `pkg/state/gitstore/gitstore.go`
- Create: `pkg/state/gitstore/gitstore_test.go`

- [ ] **Step 1: Create `pkg/state/gitstore/README.md`**

```markdown
# pkg/state/gitstore

Git-backed implementation of `state.Store`. Maps every interface method to file operations, markdown table rendering, and atomic commit-and-push in a Synchestra state repository.

This is the default state store backend — it requires no external infrastructure beyond a git remote.

## Status

Stub implementation. All methods return `errNotImplemented`. The full implementation is tracked separately.

## Spec

See `spec/features/state-store/backends/git/` for the method-to-git-operation mapping.

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `pkg/state/gitstore/gitstore.go`**

```go
package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"
	"errors"

	"github.com/synchesta-io/synchestra/pkg/state"
)

var errNotImplemented = errors.New("gitstore: not implemented")

// GitStateStore is the git-backed implementation of state.Store.
// It maps interface methods to file operations, markdown rendering,
// and atomic commit-and-push in a state repository.
type GitStateStore struct {
	stateRepoPath string
	specRepoPath  string
}

// New creates a new GitStateStore. This is the StoreFactory for the git backend.
func New(ctx context.Context, opts state.StoreOptions) (state.Store, error) {
	return &GitStateStore{
		stateRepoPath: opts.StateRepoPath,
		specRepoPath:  opts.SpecRepoPath,
	}, nil
}

func (s *GitStateStore) Task() state.TaskStore       { return &gitTaskStore{store: s} }
func (s *GitStateStore) Chat() state.ChatStore       { return &gitChatStore{store: s} }
func (s *GitStateStore) Project() state.ProjectStore { return &gitProjectStore{store: s} }

// --- TaskStore ---

type gitTaskStore struct{ store *GitStateStore }

func (t *gitTaskStore) Create(_ context.Context, _ state.TaskCreateParams) (state.Task, error) {
	return state.Task{}, errNotImplemented
}
func (t *gitTaskStore) Get(_ context.Context, _ string) (state.Task, error) {
	return state.Task{}, errNotImplemented
}
func (t *gitTaskStore) List(_ context.Context, _ state.TaskFilter) ([]state.Task, error) {
	return nil, errNotImplemented
}
func (t *gitTaskStore) Enqueue(_ context.Context, _ string) error { return errNotImplemented }
func (t *gitTaskStore) Claim(_ context.Context, _ string, _ state.ClaimParams) error {
	return errNotImplemented
}
func (t *gitTaskStore) Start(_ context.Context, _ string) error        { return errNotImplemented }
func (t *gitTaskStore) Complete(_ context.Context, _, _ string) error   { return errNotImplemented }
func (t *gitTaskStore) Fail(_ context.Context, _, _ string) error      { return errNotImplemented }
func (t *gitTaskStore) Block(_ context.Context, _, _ string) error     { return errNotImplemented }
func (t *gitTaskStore) Unblock(_ context.Context, _ string) error      { return errNotImplemented }
func (t *gitTaskStore) Release(_ context.Context, _ string) error      { return errNotImplemented }
func (t *gitTaskStore) RequestAbort(_ context.Context, _ string) error { return errNotImplemented }
func (t *gitTaskStore) ConfirmAbort(_ context.Context, _ string) error { return errNotImplemented }
func (t *gitTaskStore) Board() state.Board                             { return &gitBoard{store: t.store} }
func (t *gitTaskStore) Artifact(_ context.Context, _ string) state.ArtifactStore {
	return &gitArtifactStore{store: t.store}
}

// --- Board ---

type gitBoard struct{ store *GitStateStore }

func (b *gitBoard) Rebuild(_ context.Context) error                { return errNotImplemented }
func (b *gitBoard) Get(_ context.Context) (state.BoardView, error) { return state.BoardView{}, errNotImplemented }

// --- ArtifactStore ---

type gitArtifactStore struct{ store *GitStateStore }

func (a *gitArtifactStore) Put(_ context.Context, _ string, _ []byte) error { return errNotImplemented }
func (a *gitArtifactStore) Get(_ context.Context, _ string) ([]byte, error) { return nil, errNotImplemented }
func (a *gitArtifactStore) List(_ context.Context) ([]state.ArtifactRef, error) {
	return nil, errNotImplemented
}

// --- ChatStore ---

type gitChatStore struct{ store *GitStateStore }

func (c *gitChatStore) Create(_ context.Context, _ state.ChatCreateParams) (state.Chat, error) {
	return state.Chat{}, errNotImplemented
}
func (c *gitChatStore) Get(_ context.Context, _ string) (state.Chat, error) {
	return state.Chat{}, errNotImplemented
}
func (c *gitChatStore) List(_ context.Context, _ state.ChatFilter) ([]state.Chat, error) {
	return nil, errNotImplemented
}
func (c *gitChatStore) Finalize(_ context.Context, _ string) error { return errNotImplemented }
func (c *gitChatStore) Abandon(_ context.Context, _ string) error  { return errNotImplemented }
func (c *gitChatStore) AppendMessages(_ context.Context, _ string, _ []state.ChatMessage) error {
	return errNotImplemented
}
func (c *gitChatStore) Messages(_ context.Context, _ string) ([]state.ChatMessage, error) {
	return nil, errNotImplemented
}
func (c *gitChatStore) Artifact(_ context.Context, _ string) state.ArtifactStore {
	return &gitArtifactStore{store: c.store}
}

// --- ProjectStore ---

type gitProjectStore struct{ store *GitStateStore }

func (p *gitProjectStore) Config(_ context.Context) (state.ProjectConfig, error) {
	return state.ProjectConfig{}, errNotImplemented
}
func (p *gitProjectStore) UpdateConfig(_ context.Context, _ state.ProjectConfig) error {
	return errNotImplemented
}
func (p *gitProjectStore) RebuildREADME(_ context.Context) error { return errNotImplemented }
```

- [ ] **Step 3: Create `pkg/state/gitstore/gitstore_test.go`**

```go
package gitstore_test

// Features depended on: state-store, state-store/backends/git

import (
	"context"
	"testing"

	"github.com/synchesta-io/synchestra/pkg/state"
	"github.com/synchesta-io/synchestra/pkg/state/gitstore"
)

// TestGitStateStoreImplementsStore verifies that GitStateStore satisfies state.Store
// and that New can be used as a state.StoreFactory.
func TestGitStateStoreImplementsStore(t *testing.T) {
	// Verify New satisfies StoreFactory signature
	var factory state.StoreFactory = gitstore.New

	store, err := factory(context.Background(), state.StoreOptions{
		StateRepoPath: t.TempDir(),
		SpecRepoPath:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Verify sub-interface accessors return non-nil
	if store.Task() == nil {
		t.Error("Task() returned nil")
	}
	if store.Chat() == nil {
		t.Error("Chat() returned nil")
	}
	if store.Project() == nil {
		t.Error("Project() returned nil")
	}

	// Verify nested accessors return non-nil
	if store.Task().Board() == nil {
		t.Error("Task().Board() returned nil")
	}
	if store.Task().Artifact(context.Background(), "test-task") == nil {
		t.Error("Task().Artifact() returned nil")
	}
	if store.Chat().Artifact(context.Background(), "test-chat") == nil {
		t.Error("Chat().Artifact() returned nil")
	}
}
```

- [ ] **Step 4: Run full Go validation**

```bash
gofmt -w pkg/state/gitstore/
golangci-lint run ./pkg/state/...
go test ./pkg/state/... -v
go build ./...
go vet ./pkg/state/...
```

Expected: All pass. Test output shows `TestGitStateStoreImplementsStore` passing.

- [ ] **Step 5: Fix any issues, then commit**

```bash
git add pkg/state/gitstore/README.md pkg/state/gitstore/gitstore.go pkg/state/gitstore/gitstore_test.go
git commit -m "feat(state): add gitstore stub with interface verification test

Stub GitStateStore satisfying state.Store — all methods return
errNotImplemented. Test verifies interface satisfaction and non-nil
accessor returns."
```
