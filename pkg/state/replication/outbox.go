package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/backends/git, state-store/backends/sqlite

import (
	"context"
	"fmt"
	"time"
)

// OutboxSource is the durable per-replica delivery seam behind DrainOutbox. A
// physical journal that also has downstream replicas configured — currently
// *DALJournal, backed by the outboxCollection rows written inside the same
// transaction as every domain/journal/head write, and MemoryJournal, backed
// by an in-memory map for fast conformance and concurrency tests — exposes
// it so a drain worker can consume pending rows in cursor order and
// acknowledge each one only after the replica durably accepts it.
type OutboxSource interface {
	Journal
	// PendingOutbox returns replicaID's undelivered rows in cursor order.
	PendingOutbox(ctx context.Context, replicaID string) ([]Event, error)
	// AckOutbox deletes replicaID's rows for eventIDs in one batch (DrainOutbox
	// issues exactly one AckOutbox call per drain, covering however many
	// events it successfully applied, instead of one transaction per event).
	// It MUST be a no-op (not an error) for any eventID whose row is already
	// gone, so a duplicate drain worker or a resumed-after-crash drain never
	// fails on an ack that already landed.
	AckOutbox(ctx context.Context, replicaID string, eventIDs ...string) error
}

// DrainOutbox consumes source's durable outbox for replicaID in cursor
// order, applies each event to replica through the same idempotent
// ReplicaIngestor seam Replicate uses, and acknowledges (deletes) every
// successfully-applied row in ONE batched AckOutbox call at the end of the
// attempt — covering whatever was applied, whether the loop fully drained
// the outbox or stopped early on an ingest failure — rather than one
// transaction per event.
//
// Both halves are independently safe to repeat:
//   - Applying twice is a no-op. Journal.Append/IngestReplica dedupe by event
//     ID before touching sequence/checksum state (see DALJournal.append and
//     MemoryJournal.appendLocked), so redelivering an event the replica
//     already has just re-confirms it.
//   - Acknowledging twice is a no-op. AckOutbox deletes rows that may already
//     be gone; every OutboxSource implementation in this package documents
//     that as success, not an error.
//
// Together this makes every failure mode safe to resume, with no separate
// recovery log:
//   - A crash before the batched ack completes leaves those outbox rows in
//     place. The next DrainOutbox call redelivers them, the replica no-ops
//     (already applied), and the ack completes — never a double effect,
//     never a lost delivery.
//   - Two drain workers racing over the same pending rows each apply and ack
//     every event; whichever call reaches the replica first performs the real
//     write, the other's call and both acks observe success without altering
//     state twice.
//   - An ingest failure partway through (including one caused by the replica
//     being fenced by a concurrent promotion) still acks whatever was
//     successfully applied before the failure, and reports the exact
//     resumable cursor/lag for what remains — see
//     TestDrainOutboxMidLoopIngestFailureLeavesResumableState.
//
// DrainOutbox does not decide convergence: Replicate's head/cursor/checksum
// comparison remains the source of truth for detecting a stale, diverged, or
// role-fenced replica. DrainOutbox only pushes ready-to-apply work; callers
// that need a convergence verdict should pair it with Replicate or read
// ReplicaHealth.Cursor/EventLag from this call's own result.
//
// DrainOutbox refuses a nil replica interface value up front. It does not
// attempt to detect a typed-nil concrete pointer wrapped in a non-nil
// interface (e.g. a nil *DALJournal assigned to a Journal variable) via
// reflection — every constructor in this package (NewDALJournal,
// NewMemoryJournal, NewGitPushJournal) never returns a nil pointer on
// success, so a caller that only ever passes a successfully-constructed
// journal cannot produce one.
func DrainOutbox(ctx context.Context, source OutboxSource, replica Journal, replicaID string) (ReplicaHealth, error) {
	if source == nil || replica == nil {
		return ReplicaHealth{EndpointID: replicaID}, fmt.Errorf("replication: drain needs a source and a replica")
	}
	pending, err := source.PendingOutbox(ctx, replicaID)
	if err != nil {
		return ReplicaHealth{EndpointID: replicaID, LastError: err.Error()}, err
	}
	cursor, _, err := replica.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: replicaID, LastError: err.Error()}, err
	}
	ingestor, ok := replica.(ReplicaIngestor)
	if !ok {
		err := fmt.Errorf("replication: endpoint %q has no replica-ingest seam", replicaID)
		return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: int64(len(pending)), LastError: err.Error()}, err
	}

	health, applied, ingestErr := ingestEvents(ctx, ingestor, replicaID, cursor, pending)
	cursor, remaining := health.Cursor, health.EventLag

	var ackErr error
	if len(applied) > 0 {
		ackErr = source.AckOutbox(ctx, replicaID, applied...)
	}

	switch {
	case ingestErr != nil:
		return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: remaining, LastError: ingestErr.Error()}, ingestErr
	case ackErr != nil:
		// Every applied event is durably on the replica; only the batched
		// ack is unresolved. The rows are unchanged in the source outbox, so
		// a later DrainOutbox call safely redelivers them (no-ops) and
		// retries the ack.
		return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: remaining, LastError: ackErr.Error()}, ackErr
	default:
		return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: remaining, LastOK: time.Now().UTC()}, nil
	}
}
