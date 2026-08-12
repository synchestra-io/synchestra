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
	// AckOutbox deletes replicaID's row for eventID. It MUST be a no-op (not
	// an error) when the row is already gone, so a duplicate drain worker or
	// a resumed-after-crash drain never fails on an ack that already landed.
	AckOutbox(ctx context.Context, replicaID, eventID string) error
}

// DrainOutbox consumes source's durable outbox for replicaID in cursor
// order, applies each event to replica through the same idempotent
// ReplicaIngestor seam Replicate uses, and acknowledges (deletes) the source
// outbox row only after the replica accepts it.
//
// Both halves are independently safe to repeat:
//   - Applying twice is a no-op. Journal.Append/IngestReplica dedupe by event
//     ID before touching sequence/checksum state (see DALJournal.append and
//     MemoryJournal.append), so redelivering an event the replica already has
//     just re-confirms it.
//   - Acknowledging twice is a no-op. AckOutbox deletes a row that may already
//     be gone; every OutboxSource implementation in this package documents
//     that as success, not an error.
//
// Together this makes every failure mode safe to resume, with no separate
// recovery log:
//   - A crash between apply and ack leaves the outbox row in place. The next
//     DrainOutbox call redelivers it, the replica no-ops (already applied),
//     and the ack completes — never a double effect, never a lost delivery.
//   - Two drain workers racing over the same pending rows each apply and ack
//     every event; whichever call reaches the replica first performs the real
//     write, the other's call and both acks observe success without altering
//     state twice.
//
// DrainOutbox does not decide convergence: Replicate's head/cursor/checksum
// comparison remains the source of truth for detecting a stale, diverged, or
// role-fenced replica. DrainOutbox only pushes ready-to-apply work; callers
// that need a convergence verdict should pair it with Replicate or read
// ReplicaHealth.Cursor/EventLag from this call's own result.
func DrainOutbox(ctx context.Context, source OutboxSource, replica Journal, replicaID string) (ReplicaHealth, error) {
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
	remaining := int64(len(pending))
	for _, event := range pending {
		if err := ingestor.IngestReplica(ctx, event); err != nil {
			return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: remaining, LastError: err.Error()}, err
		}
		if err := source.AckOutbox(ctx, replicaID, event.EventID); err != nil {
			// The replica already durably has this event; only the ack is
			// unresolved. The row is unchanged in the source outbox, so a
			// later DrainOutbox call safely redelivers (no-op) and retries
			// the ack.
			return ReplicaHealth{EndpointID: replicaID, Cursor: event.Cursor, EventLag: remaining - 1, LastError: err.Error()}, err
		}
		cursor = event.Cursor
		remaining--
	}
	return ReplicaHealth{EndpointID: replicaID, Cursor: cursor, EventLag: remaining, LastOK: time.Now().UTC()}, nil
}
