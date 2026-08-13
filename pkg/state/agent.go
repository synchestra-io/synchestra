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
// task-3 adds the effort/run lifecycle state machine (EffortStore/RunStore
// Transition), explicit worktree handoff (WorktreeStore.Handoff), typed
// negotiation message kinds (Message.Kind/Evidence), lease TTL/expiry
// reclaim (LeaseStore.Acquire), and Close. The Promote() administrative
// workflow remains Planned (state-store/topology, task-5/6).
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

	// Close flushes any pending group-commit batch on the underlying
	// replication journal and durably commits it before returning
	// (state-store/journal-batching#ac:close-flushes-pending-batch), then
	// refuses further writes. A backend whose journal has no batching (or
	// has it disabled) treats this as a safe no-op. A caller that owns an
	// AgentStore's lifecycle — a one-shot CLI invocation in particular —
	// must call Close before process exit for its own writes to return
	// promptly rather than waiting out the configured batching window; see
	// pkg/state/replication's README "Batching" section. Safe to call more
	// than once.
	Close(ctx context.Context) error
}

// EffortStore defines operations on efforts — the durable unit of requested
// work.
type EffortStore interface {
	// Create records a new effort. The store generates the effort ID and
	// starts it at LifecycleStatusPlanning.
	Create(ctx context.Context, params EffortCreateParams) (Effort, error)

	// Get returns an effort by ID. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, effortID string) (Effort, error)

	// List returns efforts matching the given filter.
	List(ctx context.Context, filter EffortFilter) ([]Effort, error)

	// Transition appends a typed, audited lifecycle transition event. An
	// illegal move (not present in the lifecycle table, e.g. completed ->
	// active) returns an error wrapping ErrInvalidTransition; a terminal
	// target (completed/aborted/failed) without Disposition set is also
	// rejected.
	Transition(ctx context.Context, effortID string, params EffortTransitionParams) (Effort, error)
}

// RunStore defines operations on runs — one execution attempt within an
// effort. Start/Get/List/Finish/Correct record facts; Transition records the
// audited lifecycle state machine — see Run's Status field doc comment for
// how the two compose.
type RunStore interface {
	// Start records a new run within an effort. The store generates the run
	// ID and starts it at LifecycleStatusActive.
	Start(ctx context.Context, params RunStartParams) (Run, error)

	// Get returns a run by ID. Returns ErrNotFound if it does not exist.
	Get(ctx context.Context, runID string) (Run, error)

	// List returns runs matching the given filter.
	List(ctx context.Context, filter RunFilter) ([]Run, error)

	// Finish records a run's end timestamp and terminal reason. It is a fact
	// recording, not a lifecycle transition — see Transition.
	Finish(ctx context.Context, runID string, terminalReason string) (Run, error)

	// Correct appends an audited correction event naming the superseded
	// field/value and the replacement, without rewriting the original run
	// event. Used for optional model provenance correction.
	Correct(ctx context.Context, runID string, correction RunCorrection) (Run, error)

	// Transition appends a typed, audited lifecycle transition event. An
	// illegal move returns an error wrapping ErrInvalidTransition; a
	// terminal target without Disposition set is also rejected.
	Transition(ctx context.Context, runID string, params RunTransitionParams) (Run, error)
}

// WorktreeStore defines operations on worktree claims — the one-concurrent-
// writer binding between a run and one repository/branch/worktree.
type WorktreeStore interface {
	// Claim atomically binds one run to one repository/branch/worktree key.
	// If a live, unexpired claim already holds that key, Claim returns an
	// error wrapping ErrConflict. If a claim on that key exists but its
	// underlying lease has expired (LeaseAcquireParams.TTL elapsed) and was
	// never explicitly released, Claim instead RECLAIMS it: the stale claim
	// is marked released, its fence is permanently invalidated (any later
	// Renew/Release with the old fence returns ErrLeaseFenced), and this
	// call succeeds as the new sole writer
	// (agent-coordination#ac:abandoned-run-is-resumable). If the caller's
	// own authority fence is no longer current (e.g. after promotion), Claim
	// returns an error wrapping ErrLeaseFenced. Exactly one caller succeeds
	// for a given key at a given moment.
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

	// Handoff performs explicit sequential handoff: the outgoing run proves
	// its current fence, and the claim's ownership moves to ToRunID under a
	// freshly minted fence token in the same audited event. The outgoing
	// run's old fence is refused by any later Renew/Release
	// (agent-coordination#ac:one-writer-claim-is-fenced still holds across a
	// voluntary handoff, not just a forced reclaim). A fenced-out or
	// already-released caller receives an error wrapping ErrLeaseFenced.
	Handoff(ctx context.Context, claimID string, params WorktreeHandoffParams) (WorktreeClaim, error)
}

// MessageStore defines operations on the audited message thread shared by
// linked runs. Send/Acknowledge/Thread events reuse the exact
// "message.sent"/"message.acknowledged" journal Kind strings the Git
// fallback allowlist recognizes (state-store/topology/offline-fallback), so
// a message survives a transport switch under one shared vocabulary
// (agent-coordination#ac:messages-survive-transport-switch); Message.Kind
// (coordination.request/proposal/counterexample/decision.accepted, or the
// general-purpose default) is carried inside that same envelope's payload,
// not as a distinct journal/fallback Kind.
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
	// Acquire takes a new lease on a named resource. If a live, unexpired
	// lease already holds that resource, Acquire returns an error wrapping
	// ErrConflict. LeaseAcquireParams.TTL is recorded onto
	// AuthorityLease.ExpiresAt and IS enforced on a wall-clock basis
	// (task-3): if the currently-held lease's ExpiresAt is non-zero and not
	// after this call's observed time, it is treated as abandoned and
	// reclaimable rather than conflicting — Acquire appends one audited
	// "reclaimed" event that both invalidates the stale lease's fence (any
	// later Renew/Release against it returns ErrLeaseFenced) and grants the
	// new lease, and the returned AuthorityLease reports
	// Reclaimed/SupersededLeaseID
	// (agent-coordination#ac:abandoned-run-is-resumable). Expiry is
	// evaluated using each caller's own local clock (normally UTC
	// wall-clock time); it is not a consensus-checked deadline, so it is
	// only ever a lower bound on how long a lease is honored, subject to
	// clock skew between machines — see pkg/state/agentstore/README.md.
	// TTL<=0 never expires (matches pre-task-3 behavior: the lease is held
	// until an explicit fenced Release).
	Acquire(ctx context.Context, params LeaseAcquireParams) (AuthorityLease, error)

	// Renew proves the fence token presented is still current and extends the
	// lease. A fenced-out caller receives an error wrapping ErrLeaseFenced.
	Renew(ctx context.Context, leaseID string, fence LeaseFence) (AuthorityLease, error)

	// Release explicitly ends a lease. The fence token must still be current.
	Release(ctx context.Context, leaseID string, fence LeaseFence) error

	// Get returns the current lease for a resource. Returns ErrNotFound if no
	// live lease exists.
	Get(ctx context.Context, resource string) (AuthorityLease, error)

	// Transfer proves the fence token presented is still current and moves
	// this SAME lease's holder to toHolderRunID under a freshly minted fence
	// token, in one audited event. Unlike Acquire's reclaim path, Transfer
	// is a voluntary handoff by the current holder, not a takeover of an
	// abandoned resource; the lease ID is unchanged, but the old fence token
	// is refused by any later Renew/Release. WorktreeStore.Handoff is built
	// on this. A fenced-out caller receives an error wrapping
	// ErrLeaseFenced.
	Transfer(ctx context.Context, leaseID string, fence LeaseFence, toHolderRunID string) (AuthorityLease, error)
}

// HealthStore reports this endpoint's replication health.
type HealthStore interface {
	// Report returns this endpoint's current authority/replication health.
	Report(ctx context.Context) (HealthReport, error)
}
