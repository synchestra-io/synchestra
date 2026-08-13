package state

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import "context"

// AgentStore composes the effort/run/worktree-claim/message/activity/
// replication-journal/cursor/authority-lease/health domain contracts behind
// one navigable sub-store, following the same composition seam as
// Task()/Chat()/Project()/State(): consumers call store.Agent().Worktree().Claim(...)
// rather than a sibling top-level coordinator interface. See Open Question 1
// in spec/plans/synchestra-coordination-foundation.md for the resolution.
//
// This is a data-model and contract surface only. It intentionally does not
// expose task-3's full effort/run lifecycle state machine (planning, active,
// handoff_pending, awaiting_merge, ..., archived), CLI verbs, or the
// Promote() administrative workflow — those remain Planned.
type AgentStore interface {
	// Effort returns the effort sub-interface: the durable unit of requested
	// work that runs execute against.
	Effort() EffortStore

	// Run returns the run sub-interface: one execution attempt within an effort.
	Run() RunStore

	// Worktree returns the worktree-claim sub-interface: the one-concurrent-
	// writer binding between a run and a repository/branch/worktree.
	Worktree() WorktreeStore

	// Message returns the audited message-thread sub-interface.
	Message() MessageStore

	// Activity returns the activity sub-interface used for audit/observability
	// views (project/repository agent views).
	Activity() ActivityStore

	// Journal returns read-only access to the backend-neutral replication
	// journal this Agent() sub-store is built on. It does not grant a second
	// Append path — domain writes go through the sub-stores above.
	Journal() JournalStore

	// Cursor returns the named-consumer cursor bookmark sub-interface (e.g.
	// dashboard pagination, replica-lag reporting) — distinct from the
	// journal's own authoritative head.
	Cursor() CursorStore

	// Lease returns the authority-lease sub-interface: the fencing primitive
	// that Worktree().Claim and other exclusive operations are built on.
	Lease() LeaseStore

	// Health returns the health/status sub-interface for this endpoint.
	Health() HealthStore
}

// EffortStore defines operations on efforts — the durable unit of requested
// work. It intentionally exposes no lifecycle-transition verbs; those belong
// to task-3.
type EffortStore interface {
	// Create records a new effort. The store generates the effort ID.
	Create(ctx context.Context, params EffortCreateParams) (Effort, error)

	// Get returns an effort by ID. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, effortID string) (Effort, error)

	// List returns efforts matching the given filter.
	List(ctx context.Context, filter EffortFilter) ([]Effort, error)
}

// RunStore defines operations on runs — one execution attempt within an
// effort. It records facts (start/end/terminal reason) rather than
// implementing task-3's full lifecycle state machine.
type RunStore interface {
	// Start records a new run within an effort. The store generates the run ID.
	Start(ctx context.Context, params RunStartParams) (Run, error)

	// Get returns a run by ID. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, runID string) (Run, error)

	// List returns runs matching the given filter.
	List(ctx context.Context, filter RunFilter) ([]Run, error)

	// Finish records a run's end timestamp and terminal reason. It is a fact
	// recording, not a delivery/cleanup disposition decision — that
	// remains task-3 scope.
	Finish(ctx context.Context, runID string, terminalReason string) (Run, error)

	// Correct appends an audited correction event naming the superseded
	// field/value and the replacement, without rewriting the original run
	// event. Used for optional model provenance correction.
	Correct(ctx context.Context, runID string, correction RunCorrection) (Run, error)
}

// WorktreeStore defines operations on worktree claims — the one-concurrent-
// writer binding between a run and one repository/branch/worktree.
type WorktreeStore interface {
	// Claim atomically binds one run to one repository/branch/worktree key.
	// If a live claim already holds that key, Claim returns an error wrapping
	// ErrConflict. If the caller's authority fence is no longer current (e.g.
	// after promotion), Claim returns an error wrapping ErrLeaseFenced. Exactly
	// one caller succeeds for a given key.
	Claim(ctx context.Context, params WorktreeClaimParams) (WorktreeClaim, error)

	// Renew proves the fence token presented is still current and extends the
	// claim's lease without changing ownership. A fenced-out caller receives
	// an error wrapping ErrLeaseFenced.
	Renew(ctx context.Context, claimID string, fence LeaseFence) (WorktreeClaim, error)

	// Release explicitly ends a claim. The fence token must still be current.
	Release(ctx context.Context, claimID string, fence LeaseFence) error

	// Get returns a worktree claim by ID. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, claimID string) (WorktreeClaim, error)

	// List returns worktree claims matching the given filter.
	List(ctx context.Context, filter WorktreeFilter) ([]WorktreeClaim, error)
}

// MessageStore defines operations on the audited message thread shared by
// linked runs.
type MessageStore interface {
	// Send records an immutable message envelope. The store generates the
	// message ID.
	Send(ctx context.Context, params MessageSendParams) (Message, error)

	// Acknowledge records a recipient's acknowledgement of a message as a
	// separate, immutable record.
	Acknowledge(ctx context.Context, messageID, recipientRunID string) (MessageAck, error)

	// Thread returns every message in a thread, in delivery order.
	Thread(ctx context.Context, threadID string) ([]Message, error)
}

// ActivityStore defines operations on the append-only activity feed backing
// project/repository agent views.
type ActivityStore interface {
	// Record appends one activity entry. The store generates the activity ID.
	Record(ctx context.Context, params ActivityRecordParams) (ActivityRecord, error)

	// List returns activity entries matching the given filter, in
	// chronological order.
	List(ctx context.Context, filter ActivityFilter) ([]ActivityRecord, error)
}

// JournalStore exposes read-only access to the backend-neutral replication
// journal beneath this Agent() sub-store.
type JournalStore interface {
	// Head returns the latest durable cursor and its checksum evidence.
	Head(ctx context.Context) (Cursor, string, error)

	// After returns journal entries strictly after cursor, in order.
	After(ctx context.Context, cursor Cursor) ([]JournalEntry, error)
}

// CursorStore tracks named-consumer read positions (e.g. a dashboard's last
// rendered cursor, a replica's lag-reporting bookmark) independently of the
// journal's own authoritative head.
type CursorStore interface {
	// Get returns the last cursor a named consumer durably advanced past.
	// A consumer with no recorded cursor gets the zero Cursor.
	Get(ctx context.Context, consumerID string) (Cursor, error)

	// Advance durably records a consumer's new position. It rejects a cursor
	// that would move backward relative to the consumer's last recorded value.
	Advance(ctx context.Context, consumerID string, cursor Cursor) error
}

// LeaseStore defines the authority-lease fencing primitive. It is a thin
// domain wrapper over the replication authority epoch — not a new consensus
// mechanism — used to fence exclusive operations such as worktree claims
// across promotion.
type LeaseStore interface {
	// Acquire takes a new lease on a named resource. If an unreleased lease
	// already exists for that resource, Acquire returns an error wrapping
	// ErrConflict. LeaseAcquireParams.TTL is recorded onto
	// AuthorityLease.ExpiresAt but is NOT YET ENFORCED: no implementation
	// evaluates ExpiresAt against the current time, so a lease stays held
	// until it is explicitly released via a fenced Release, no matter how
	// long ago it expired. Expiry evaluation/reconciliation for an abandoned
	// run is out of scope here — see
	// pkg/state/agentstore/README.md's "Open Questions" (task-3,
	// abandoned-run-is-resumable), which owns closing this gap.
	Acquire(ctx context.Context, params LeaseAcquireParams) (AuthorityLease, error)

	// Renew proves the fence token presented is still current and extends the
	// lease. A fenced-out caller receives an error wrapping ErrLeaseFenced.
	Renew(ctx context.Context, leaseID string, fence LeaseFence) (AuthorityLease, error)

	// Release explicitly ends a lease. The fence token must still be current.
	Release(ctx context.Context, leaseID string, fence LeaseFence) error

	// Get returns the current lease for a resource. Returns ErrNotFound if no
	// live lease exists.
	Get(ctx context.Context, resource string) (AuthorityLease, error)
}

// HealthStore reports this endpoint's replication health.
type HealthStore interface {
	// Report returns this endpoint's current authority/replication health.
	Report(ctx context.Context) (HealthReport, error)
}
