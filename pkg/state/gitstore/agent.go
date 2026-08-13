package gitstore

// Features implemented: state-store/topology
// Features depended on:  state-store, state-store/topology, agent-coordination

import (
	"context"
	"fmt"

	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/agentstore"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// Agent returns the agent-coordination sub-store, wired to a
// replication.Journal built directly over StateRepoPath through the DALgo-
// to-inGitDB adapter (the same construction the replication package's own
// tests use — see pkg/state/replication/git_durability_test.go). This is
// thin wiring: the effort/run/worktree-claim/message/activity/lease/health
// domain logic lives once in pkg/state/agentstore, not here.
//
// The underlying *agentstore.Store (and its journal) is constructed once
// per GitStateStore instance and cached (s.agentOnce) — repeated Agent()
// calls on the SAME GitStateStore return the SAME instance, matching a
// single CLI invocation's or long-running server's actual lifecycle, so (1)
// group-commit batching (below) can actually group writes issued through
// separate Agent() calls, and (2) GitStateStore.Close closes the journal
// that was actually written to rather than a freshly-built empty one. A
// distinct GitStateStore (e.g. a restarted CLI process, or a test's
// newStore() helper) gets its own fresh journal, exactly as before.
//
// StateRepoPath must already be an initialized Git repository (a clone or
// `git init`); construction is fail-closed rather than silently succeeding
// against a bogus path — every method on the returned AgentStore surfaces
// the same construction error until a new GitStateStore is constructed.
//
// agentCore/agentErr are written and read under s.agentMu (gitstore.go), not
// just guarded by agentOnce: sync.Once's happens-before guarantee only
// covers goroutines that themselves call Do, and Close (gitstore.go) reads
// these fields WITHOUT calling Do (it must not force construction as a side
// effect of closing) — so the mutex, not the Once alone, is what makes that
// cross-method read race-free. See GitStateStore's doc comment.
func (s *GitStateStore) Agent() state.AgentStore {
	s.agentOnce.Do(func() {
		core, err := s.buildAgentCore()
		s.agentMu.Lock()
		s.agentCore, s.agentErr = core, err
		s.agentMu.Unlock()
	})
	s.agentMu.Lock()
	core, err := s.agentCore, s.agentErr
	s.agentMu.Unlock()
	if err != nil {
		return unavailableAgentStore{err: fmt.Errorf("gitstore: Agent() unavailable: %w", err)}
	}
	return core
}

func (s *GitStateStore) buildAgentCore() (*agentstore.Store, error) {
	opts := s.agentOptions.withDefaults()
	if s.stateRepoPath == "" {
		return nil, fmt.Errorf("gitstore: StateRepoPath is required")
	}
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("gitstore: GitStoreOptions.ProjectID is required")
	}
	database, err := dalgo2ingitdb.NewDatabase(s.stateRepoPath, validator.NewCollectionsReader())
	if err != nil {
		return nil, fmt.Errorf("gitstore: open Git DAL database at %q: %w", s.stateRepoPath, err)
	}
	ctx := context.Background()
	if err := replication.EnsureDALJournalSchema(ctx, database); err != nil {
		return nil, fmt.Errorf("gitstore: ensure journal schema: %w", err)
	}
	journal, err := replication.NewDALJournal(database, replication.DALJournalOptions{
		ProjectID:      opts.ProjectID,
		EndpointID:     opts.AgentEndpointID,
		Role:           opts.AgentRole,
		AuthorityEpoch: opts.AuthorityEpoch,
		ReplicaIDs:     opts.ReplicaIDs,
		CommitMessage:  "synchestra agent state",
		// Batching now uses the documented default (100 items/1000ms, nil
		// MaxBatchItems/MaxBatchDelayMS) rather than the explicit-zero pin
		// this wiring carried before task-3: Agent() is now constructed once
		// per GitStateStore and cached (see the doc comment above), so
		// concurrent callers sharing one GitStateStore actually have a batch
		// to group into, and Close (GitStateStore.Close, gitstore.go) is now
		// wired through the Store lifecycle so a one-shot CLI invocation's
		// lone pending Append does not have to wait out the full window —
		// see pkg/cli/agent's runFlushed helper and
		// pkg/state/agentstore/README.md's Open Questions for the measured
		// before/after.
	})
	if err != nil {
		return nil, fmt.Errorf("gitstore: configure Git journal: %w", err)
	}
	return agentstore.New(journal, agentstore.Options{
		ProjectID:      opts.ProjectID,
		EndpointID:     opts.AgentEndpointID,
		ActorID:        "gitstore:" + s.runID,
		AuthorityEpoch: opts.AuthorityEpoch,
	})
}

// withDefaults fills in the Agent()-only fields left zero by a caller that
// only needed Task()/Chat()/Project()/State(). It never mutates opts, so
// GitStateStore.New remains a plain, side-effect-free copy of the caller's
// options.
func (opts GitStoreOptions) withDefaults() GitStoreOptions {
	if opts.AgentEndpointID == "" {
		opts.AgentEndpointID = "git"
	}
	if opts.AgentRole == "" {
		opts.AgentRole = replication.RoleActive
	}
	if opts.AuthorityEpoch == 0 {
		opts.AuthorityEpoch = 1
	}
	return opts
}

// unavailableAgentStore reports the same construction error from every
// sub-store, mirroring the errNotImplemented stub pattern already used by
// gitChatStore/gitProjectStore/gitArtifactStore for methods this backend
// has not wired yet.
type unavailableAgentStore struct{ err error }

var _ state.AgentStore = unavailableAgentStore{}

func (u unavailableAgentStore) Effort() state.EffortStore     { return unavailableEffortStore(u) }
func (u unavailableAgentStore) Run() state.RunStore           { return unavailableRunStore(u) }
func (u unavailableAgentStore) Worktree() state.WorktreeStore { return unavailableWorktreeStore(u) }
func (u unavailableAgentStore) Message() state.MessageStore   { return unavailableMessageStore(u) }
func (u unavailableAgentStore) Activity() state.ActivityStore { return unavailableActivityStore(u) }
func (u unavailableAgentStore) Journal() state.JournalStore   { return unavailableJournalStore(u) }
func (u unavailableAgentStore) Cursor() state.CursorStore     { return unavailableCursorStore(u) }
func (u unavailableAgentStore) Lease() state.LeaseStore       { return unavailableLeaseStore(u) }
func (u unavailableAgentStore) Health() state.HealthStore     { return unavailableHealthStore(u) }
func (u unavailableAgentStore) Close(context.Context) error   { return u.err }

type unavailableEffortStore struct{ err error }

func (u unavailableEffortStore) Create(context.Context, state.EffortCreateParams) (state.Effort, error) {
	return state.Effort{}, u.err
}
func (u unavailableEffortStore) Get(context.Context, string) (state.Effort, error) {
	return state.Effort{}, u.err
}
func (u unavailableEffortStore) List(context.Context, state.EffortFilter) ([]state.Effort, error) {
	return nil, u.err
}
func (u unavailableEffortStore) Transition(context.Context, string, state.EffortTransitionParams) (state.Effort, error) {
	return state.Effort{}, u.err
}

type unavailableRunStore struct{ err error }

func (u unavailableRunStore) Start(context.Context, state.RunStartParams) (state.Run, error) {
	return state.Run{}, u.err
}
func (u unavailableRunStore) Get(context.Context, string) (state.Run, error) {
	return state.Run{}, u.err
}
func (u unavailableRunStore) List(context.Context, state.RunFilter) ([]state.Run, error) {
	return nil, u.err
}
func (u unavailableRunStore) Finish(context.Context, string, string) (state.Run, error) {
	return state.Run{}, u.err
}
func (u unavailableRunStore) Correct(context.Context, string, state.RunCorrection) (state.Run, error) {
	return state.Run{}, u.err
}
func (u unavailableRunStore) Transition(context.Context, string, state.RunTransitionParams) (state.Run, error) {
	return state.Run{}, u.err
}

type unavailableWorktreeStore struct{ err error }

func (u unavailableWorktreeStore) Claim(context.Context, state.WorktreeClaimParams) (state.WorktreeClaim, error) {
	return state.WorktreeClaim{}, u.err
}
func (u unavailableWorktreeStore) Renew(context.Context, string, state.LeaseFence) (state.WorktreeClaim, error) {
	return state.WorktreeClaim{}, u.err
}
func (u unavailableWorktreeStore) Release(context.Context, string, state.LeaseFence) error {
	return u.err
}
func (u unavailableWorktreeStore) Get(context.Context, string) (state.WorktreeClaim, error) {
	return state.WorktreeClaim{}, u.err
}
func (u unavailableWorktreeStore) List(context.Context, state.WorktreeFilter) ([]state.WorktreeClaim, error) {
	return nil, u.err
}
func (u unavailableWorktreeStore) Handoff(context.Context, string, state.WorktreeHandoffParams) (state.WorktreeClaim, error) {
	return state.WorktreeClaim{}, u.err
}

type unavailableMessageStore struct{ err error }

func (u unavailableMessageStore) Send(context.Context, state.MessageSendParams) (state.Message, error) {
	return state.Message{}, u.err
}
func (u unavailableMessageStore) Acknowledge(context.Context, string, string) (state.MessageAck, error) {
	return state.MessageAck{}, u.err
}
func (u unavailableMessageStore) Thread(context.Context, string) ([]state.Message, error) {
	return nil, u.err
}

type unavailableActivityStore struct{ err error }

func (u unavailableActivityStore) Record(context.Context, state.ActivityRecordParams) (state.ActivityRecord, error) {
	return state.ActivityRecord{}, u.err
}
func (u unavailableActivityStore) List(context.Context, state.ActivityFilter) ([]state.ActivityRecord, error) {
	return nil, u.err
}

type unavailableJournalStore struct{ err error }

func (u unavailableJournalStore) Head(context.Context) (state.Cursor, string, error) {
	return state.Cursor{}, "", u.err
}
func (u unavailableJournalStore) After(context.Context, state.Cursor) ([]state.JournalEntry, error) {
	return nil, u.err
}

type unavailableCursorStore struct{ err error }

func (u unavailableCursorStore) Get(context.Context, string) (state.Cursor, error) {
	return state.Cursor{}, u.err
}
func (u unavailableCursorStore) Advance(context.Context, string, state.Cursor) error { return u.err }

type unavailableLeaseStore struct{ err error }

func (u unavailableLeaseStore) Acquire(context.Context, state.LeaseAcquireParams) (state.AuthorityLease, error) {
	return state.AuthorityLease{}, u.err
}
func (u unavailableLeaseStore) Renew(context.Context, string, state.LeaseFence) (state.AuthorityLease, error) {
	return state.AuthorityLease{}, u.err
}
func (u unavailableLeaseStore) Release(context.Context, string, state.LeaseFence) error { return u.err }
func (u unavailableLeaseStore) Get(context.Context, string) (state.AuthorityLease, error) {
	return state.AuthorityLease{}, u.err
}
func (u unavailableLeaseStore) Transfer(context.Context, string, state.LeaseFence, string) (state.AuthorityLease, error) {
	return state.AuthorityLease{}, u.err
}

type unavailableHealthStore struct{ err error }

func (u unavailableHealthStore) Report(context.Context) (state.HealthReport, error) {
	return state.HealthReport{}, u.err
}
