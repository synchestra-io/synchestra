package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"
	"errors"
	"sync"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/agentstore"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

var errNotImplemented = errors.New("gitstore: not implemented")

// GitStoreOptions holds git-backend-specific configuration.
type GitStoreOptions struct {
	state.StoreOptions        // embeds shared options including SyncConfig
	RunID              string // agent branch: agent/<run-id>

	// Agent() sub-store wiring (state-store/topology, Task 1 scope). These
	// configure the replication.Journal that GitStateStore.Agent() builds
	// over StateRepoPath — see agent.go. ProjectID is required to use
	// Agent(); the rest have defaults (see agent.go's withDefaults).
	ProjectID       string
	AgentEndpointID string           // defaults to "git"
	AgentRole       replication.Role // defaults to replication.RoleActive
	AuthorityEpoch  int64            // defaults to 1
	ReplicaIDs      []string
}

// GitStateStore is the git-backed implementation of state.Store.
// It maps interface methods to file operations, markdown rendering,
// and atomic commit-and-push in a state repository.
type GitStateStore struct {
	stateRepoPath string
	specRepoPaths []string
	sync          state.SyncConfig
	runID         string
	agentOptions  GitStoreOptions

	// agentOnce/agentCore/agentErr cache the *agentstore.Store (and, through
	// it, the replication.Journal) buildAgentCore constructs, so every call
	// to Agent() on ONE GitStateStore instance shares the SAME journal
	// instance rather than opening a fresh Git-backed database and journal
	// per call. This matters for two things task-3 wires together: (1) with
	// group-commit batching enabled (agent.go), concurrent writes through
	// this store's Agent() can only ever land in the same pending batch if
	// they share a journal; (2) Close (below) can only flush the batch that
	// was actually written to if it closes that SAME cached journal, not a
	// newly-constructed empty one. See gitstore/README.md.
	//
	// agentMu guards every read/write of agentCore/agentErr OUTSIDE
	// agentOnce.Do's own closure: sync.Once only guarantees a happens-before
	// edge for goroutines that themselves call Do (agent.go's Agent()) — a
	// caller that reads the fields directly without going through Do (Close,
	// below) has no such guarantee and races the field write on the first
	// concurrent Agent() call. Close must still be a no-op when Agent() was
	// never called (nothing to flush), so it cannot itself call
	// agentOnce.Do(buildFunc) — that would force construction as a side
	// effect of Close, which is exactly the behavior its doc comment
	// promises NOT to have. Routing both sides through agentMu instead
	// keeps Close a safe read of "whatever has been set so far" (nil until
	// the first Agent() call's Do closure finishes) without ever triggering
	// that closure itself, and without a data race for -race to catch.
	agentOnce sync.Once
	agentMu   sync.Mutex
	agentCore *agentstore.Store
	agentErr  error
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
		agentOptions:  opts,
	}, nil
}

func (s *GitStateStore) Task() state.TaskStore       { return &gitTaskStore{store: s} }
func (s *GitStateStore) Chat() state.ChatStore       { return &gitChatStore{store: s} }
func (s *GitStateStore) Project() state.ProjectStore { return &gitProjectStore{store: s} }
func (s *GitStateStore) State() state.StateSync      { return &gitStateSync{store: s} }

// Close flushes any pending group-commit batch on the Agent() journal, if
// one was ever constructed on this GitStateStore instance
// (state-store/journal-batching#ac:close-flushes-pending-batch). A
// GitStateStore whose Agent() was never called (or failed to construct) has
// nothing to flush and this is a safe no-op. Safe to call more than once —
// it delegates to agentstore.Store.Close, itself idempotent. Reads agentCore
// under agentMu — the same guard Agent() (agent.go) writes it under — so a
// long-running server's concurrent Agent()+Close call pair (the scenario
// this type's doc comment above anticipates) is never a data race, only a
// benign "Close ran before the first Agent() call finished building, so
// there was nothing yet to flush" ordering.
func (s *GitStateStore) Close(ctx context.Context) error {
	s.agentMu.Lock()
	core := s.agentCore
	s.agentMu.Unlock()
	if core == nil {
		return nil
	}
	return core.Close(ctx)
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
