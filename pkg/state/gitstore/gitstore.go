package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"
	"errors"

	"github.com/synchestra-io/synchestra/pkg/state"
)

var errNotImplemented = errors.New("gitstore: not implemented")

// SyncMode controls how the git backend synchronizes with the remote.
type SyncMode string

const (
	// SyncModeSync pulls before reads and pushes after writes.
	// This is the safe default for multi-host/distributed agent setups.
	SyncModeSync SyncMode = "sync"

	// SyncModeLocal operates only on the local clone — no pull/push per
	// operation. The caller is responsible for periodic sync with the remote.
	// Ideal for single-host setups where all agents share one local clone.
	SyncModeLocal SyncMode = "local" // TODO: Implement
)

// Options holds git-backend-specific configuration.
type Options struct {
	StateRepoPath string
	SpecRepoPaths []string
	SyncMode      SyncMode // defaults to SyncModeSync if empty
}

// GitStateStore is the git-backed implementation of state.Store.
// It maps interface methods to file operations, markdown rendering,
// and atomic commit-and-push in a state repository.
type GitStateStore struct {
	stateRepoPath string
	specRepoPaths []string
	syncMode      SyncMode
}

// New creates a new GitStateStore with git-backend-specific options.
func New(_ context.Context, opts Options) (state.Store, error) {
	syncMode := opts.SyncMode
	if syncMode == "" {
		syncMode = SyncModeSync
	}
	return &GitStateStore{
		stateRepoPath: opts.StateRepoPath,
		specRepoPaths: opts.SpecRepoPaths,
		syncMode:      syncMode,
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
func (t *gitTaskStore) Complete(_ context.Context, _, _ string) error  { return errNotImplemented }
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

func (b *gitBoard) Rebuild(_ context.Context) error { return errNotImplemented }
func (b *gitBoard) Get(_ context.Context) (state.BoardView, error) {
	return state.BoardView{}, errNotImplemented
}

// --- ArtifactStore ---

type gitArtifactStore struct{ store *GitStateStore }

func (a *gitArtifactStore) Put(_ context.Context, _ string, _ []byte) error { return errNotImplemented }
func (a *gitArtifactStore) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, errNotImplemented
}
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
