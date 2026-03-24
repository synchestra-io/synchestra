package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"
	"errors"

	"github.com/synchestra-io/synchestra/pkg/state"
)

var errNotImplemented = errors.New("gitstore: not implemented")

// GitStoreOptions holds git-backend-specific configuration.
type GitStoreOptions struct {
	state.StoreOptions        // embeds shared options including SyncConfig
	RunID              string // agent branch: agent/<run-id>
}

// GitStateStore is the git-backed implementation of state.Store.
// It maps interface methods to file operations, markdown rendering,
// and atomic commit-and-push in a state repository.
type GitStateStore struct {
	stateRepoPath string
	specRepoPaths []string
	sync          state.SyncConfig
	runID         string
}

// New creates a new GitStateStore with git-backend-specific options.
func New(_ context.Context, opts GitStoreOptions) (state.Store, error) {
	sync := opts.Sync
	if sync.Pull == "" {
		sync.Pull = state.SyncOnCommit
	}
	if sync.Push == "" {
		sync.Push = state.SyncOnCommit
	}
	return &GitStateStore{
		stateRepoPath: opts.StateRepoPath,
		specRepoPaths: opts.SpecRepoPaths,
		sync:          sync,
		runID:         opts.RunID,
	}, nil
}

func (s *GitStateStore) Task() state.TaskStore       { return &gitTaskStore{store: s} }
func (s *GitStateStore) Chat() state.ChatStore       { return &gitChatStore{store: s} }
func (s *GitStateStore) Project() state.ProjectStore { return &gitProjectStore{store: s} }
func (s *GitStateStore) State() state.StateSync      { return &gitStateSync{store: s} }

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
