package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import "sort"

// Deterministic schema and migration ownership.
//
// agentstore is the sole owner of the event Kind + payload shapes below. Its
// only persistence surface is a replication.Journal (see store.go): unlike
// the SQLite backend — which owns a DALgo table schema and versions it
// through DALgo migrations, per spec/features/state-store/backends/sqlite
// ("Schema changes are versioned DALgo migrations... Synchestra owns its
// domain schema and migration history") — agentstore has no table schema of
// its own to migrate. Its schema is the set of event Kind strings and their
// JSON payload shapes, each pinned to an explicit version the same way
// replication pins replication.Event ("schema":
// "https://synchestra.io/schemas/state-change/v1") and the Git fallback pins
// its allowed kinds (replication.fallbackKinds).
//
// A payload shape change adds a new versioned Kind (e.g.
// "agent.worktree.claimed.v2") and, if old events must still be read, a
// translator from the old payload to the new one — it never mutates an
// existing Kind's payload shape in place. EventKinds lists every Kind this
// package may emit, so a caller (or a future schema-compatibility check) can
// enumerate the owned vocabulary deterministically.
const (
	KindEffortCreated       = "agent.effort.created"
	KindEffortTransitioned  = "agent.effort.transitioned" // task-3: lifecycle state machine
	KindRunStarted          = "agent.run.started"
	KindRunFinished         = "agent.run.finished"
	KindRunCorrected        = "agent.run.corrected"
	KindRunTransitioned     = "agent.run.transitioned" // task-3: lifecycle state machine
	KindLeaseAcquired       = "agent.lease.acquired"
	KindLeaseRenewed        = "agent.lease.renewed"
	KindLeaseReleased       = "agent.lease.released"
	KindLeaseReclaimed      = "agent.lease.reclaimed"   // task-3: TTL/expiry reclaim of an abandoned lease
	KindLeaseTransferred    = "agent.lease.transferred" // task-3: voluntary handoff to a new holder
	KindWorktreeClaimed     = "agent.worktree.claimed"
	KindWorktreeRenewed     = "agent.worktree.renewed"
	KindWorktreeReleased    = "agent.worktree.released"
	KindWorktreeReclaimed   = "agent.worktree.reclaimed"  // task-3: mirrors KindLeaseReclaimed for the bound claim
	KindWorktreeHandedOff   = "agent.worktree.handed_off" // task-3: mirrors KindLeaseTransferred for the bound claim
	KindActivityRecorded    = "agent.activity.recorded"
	KindCursorAdvanced      = "agent.cursor.advanced"
	KindMessageSent         = "message.sent"         // shared with the Git fallback allowlist (replication.fallbackKinds)
	KindMessageAcknowledged = "message.acknowledged" // shared with the Git fallback allowlist (replication.fallbackKinds)
)

// Per-payload schema identifiers, mirroring replication.EventSchemaV1's
// versioned-URI convention.
const (
	schemaEffortCreatedV1       = "https://synchestra.io/schemas/agent/effort-created/v1"
	schemaEffortTransitionedV1  = "https://synchestra.io/schemas/agent/effort-transitioned/v1"
	schemaRunStartedV1          = "https://synchestra.io/schemas/agent/run-started/v1"
	schemaRunFinishedV1         = "https://synchestra.io/schemas/agent/run-finished/v1"
	schemaRunCorrectedV1        = "https://synchestra.io/schemas/agent/run-corrected/v1"
	schemaRunTransitionedV1     = "https://synchestra.io/schemas/agent/run-transitioned/v1"
	schemaLeaseAcquiredV1       = "https://synchestra.io/schemas/agent/lease-acquired/v1"
	schemaLeaseRenewedV1        = "https://synchestra.io/schemas/agent/lease-renewed/v1"
	schemaLeaseReleasedV1       = "https://synchestra.io/schemas/agent/lease-released/v1"
	schemaLeaseReclaimedV1      = "https://synchestra.io/schemas/agent/lease-reclaimed/v1"
	schemaLeaseTransferredV1    = "https://synchestra.io/schemas/agent/lease-transferred/v1"
	schemaWorktreeClaimedV1     = "https://synchestra.io/schemas/agent/worktree-claimed/v1"
	schemaWorktreeRenewedV1     = "https://synchestra.io/schemas/agent/worktree-renewed/v1"
	schemaWorktreeReleasedV1    = "https://synchestra.io/schemas/agent/worktree-released/v1"
	schemaWorktreeReclaimedV1   = "https://synchestra.io/schemas/agent/worktree-reclaimed/v1"
	schemaWorktreeHandedOffV1   = "https://synchestra.io/schemas/agent/worktree-handed-off/v1"
	schemaActivityRecordedV1    = "https://synchestra.io/schemas/agent/activity-recorded/v1"
	schemaCursorAdvancedV1      = "https://synchestra.io/schemas/agent/cursor-advanced/v1"
	schemaMessageSentV2         = "https://synchestra.io/schemas/agent/message-sent/v2"
	schemaMessageAcknowledgedV1 = "https://synchestra.io/schemas/agent/message-acknowledged/v1"
)

// EventKinds returns every event Kind this package owns and may emit through
// a replication.Journal, sorted for deterministic output.
func EventKinds() []string {
	kinds := []string{
		KindEffortCreated, KindEffortTransitioned,
		KindRunStarted, KindRunFinished, KindRunCorrected, KindRunTransitioned,
		KindLeaseAcquired, KindLeaseRenewed, KindLeaseReleased, KindLeaseReclaimed, KindLeaseTransferred,
		KindWorktreeClaimed, KindWorktreeRenewed, KindWorktreeReleased, KindWorktreeReclaimed, KindWorktreeHandedOff,
		KindActivityRecorded, KindCursorAdvanced,
		KindMessageSent, KindMessageAcknowledged,
	}
	sort.Strings(kinds)
	return kinds
}
