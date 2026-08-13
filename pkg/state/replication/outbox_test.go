package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/topology, agent-coordination, state-store/journal-batching

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func newDrainPair(t *testing.T, replicaID string) (*MemoryJournal, *MemoryJournal) {
	t.Helper()
	source, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1, ReplicaIDs: []string{replicaID}, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	replica, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: replicaID, Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	return source, replica
}

// TestDrainOutboxDeliversInCursorOrderAndAcksExactlyOnce is the basic happy
// path: every pending row is applied to the replica and acked (deleted) from
// the source, in cursor order, and a second drain of the now-empty outbox is
// a true no-op.
func TestDrainOutboxDeliversInCursorOrderAndAcksExactlyOnce(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)

	health, err := DrainOutbox(ctx, source, replica, "mirror")
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if health.EventLag != 0 || health.Cursor != events[len(events)-1].Cursor {
		t.Fatalf("health = %+v, want caught-up cursor", health)
	}
	got, err := replica.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(events) {
		t.Fatalf("replica has %d events, want %d", len(got), len(events))
	}
	for i, event := range got {
		if event.EventID != events[i].EventID {
			t.Fatalf("replica event %d = %q, want %q (order not preserved)", i, event.EventID, events[i].EventID)
		}
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != 0 {
		t.Fatalf("outbox after drain = %#v, %v; want empty", pending, err)
	}

	// A second drain of an empty outbox is a true no-op: nothing to apply,
	// nothing to ack, no error.
	again, err := DrainOutbox(ctx, source, replica, "mirror")
	if err != nil || again.EventLag != 0 {
		t.Fatalf("drain of empty outbox = %+v, %v", again, err)
	}
}

// ackFailer wraps an OutboxSource and fails its FIRST batched AckOutbox call
// outright (before any row is deleted), simulating a crash during the single
// per-drain ack transaction after every event in the batch was already
// durably applied to the replica. This is the specific window the
// batched-ack design must survive without loss or double-apply.
type ackFailer struct {
	OutboxSource
	failed bool
}

func (a *ackFailer) AckOutbox(ctx context.Context, replicaID string, eventIDs ...string) error {
	if !a.failed {
		a.failed = true
		return errors.New("simulated crash during batched ack")
	}
	return a.OutboxSource.AckOutbox(ctx, replicaID, eventIDs...)
}

// TestDrainOutboxCrashDuringBatchedAckConvergesOnResume kills delivery
// immediately after every pending event is durably applied to the replica
// but before the single batched ack transaction commits, then resumes with a
// fresh DrainOutbox call standing in for a restarted worker. No event may be
// applied twice, and the outbox must end up fully drained.
func TestDrainOutboxCrashDuringBatchedAckConvergesOnResume(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)

	failing := &ackFailer{OutboxSource: source}
	health, err := DrainOutbox(ctx, failing, replica, "mirror")
	if err == nil {
		t.Fatal("expected the injected batched-ack crash to surface")
	}
	if health.EventLag != 0 || health.Cursor != events[len(events)-1].Cursor {
		t.Fatalf("crash health = %+v, want every event already applied despite the failed ack", health)
	}
	// The replica already durably has every event -- the crash is strictly in
	// the batched ack, after every apply succeeded -- but the source outbox
	// still holds every row because that one ack transaction never committed.
	got, err := replica.After(ctx, Cursor{})
	if err != nil || len(got) != len(events) {
		t.Fatalf("replica after crash = %#v, %v; want all %d events already applied", got, err, len(events))
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != len(events) {
		t.Fatalf("outbox after crash = %d rows, %v; want all %d unacked rows", len(pending), err, len(events))
	}

	// Resume: a fresh drain redelivers every event (idempotent no-op on the
	// already-caught-up replica), and this time the batched ack succeeds.
	resumed, err := DrainOutbox(ctx, failing, replica, "mirror")
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.EventLag != 0 || resumed.Cursor != events[len(events)-1].Cursor {
		t.Fatalf("resume health = %+v, want fully drained", resumed)
	}
	finalPending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(finalPending) != 0 {
		t.Fatalf("outbox after resume = %#v, %v; want empty", finalPending, err)
	}
	finalReplica, err := replica.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(finalReplica) != len(events) {
		t.Fatalf("replica after resume has %d events, want %d (no double-apply, no loss)", len(finalReplica), len(events))
	}
	for i, event := range finalReplica {
		if event.Checksum != events[i].Checksum {
			t.Fatalf("replica event %d checksum = %q, want %q", i, event.Checksum, events[i].Checksum)
		}
	}
}

// fenceAfterN wraps a *MemoryJournal replica and, starting with its (n+1)th
// IngestReplica call, flips the underlying journal's role away from
// RoleReplica before delegating -- simulating a concurrent promotion fencing
// this replica mid-drain (e.g. because it was itself just promoted, or
// fenced further downstream). It lets
// TestDrainOutboxMidLoopIngestFailureLeavesResumableState exercise the exact
// failure mode finding #13 calls out without needing a full concurrent
// Promote() call.
type fenceAfterN struct {
	*MemoryJournal
	n     int
	calls int
}

func (f *fenceAfterN) IngestReplica(ctx context.Context, event Event) error {
	f.calls++
	if f.calls > f.n {
		f.mu.Lock()
		f.role = RoleActive
		f.mu.Unlock()
	}
	return f.MemoryJournal.IngestReplica(ctx, event)
}

// TestDrainOutboxMidLoopIngestFailureLeavesResumableState fences the replica
// out from under DrainOutbox after two of four pending events have already
// been applied. The call must fail with a *RoleFenceError (never anything
// else -- the ingest seam itself refuses cleanly), report EventLag/Cursor
// for exactly what was and was not applied, and leave the source outbox
// holding exactly the not-yet-applied rows so a later drain (once the
// replica is un-fenced) can resume and finish.
func TestDrainOutboxMidLoopIngestFailureLeavesResumableState(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)

	fencing := &fenceAfterN{MemoryJournal: replica, n: 2}
	health, err := DrainOutbox(ctx, source, fencing, "mirror")
	if err == nil {
		t.Fatal("expected the mid-drain fence to surface")
	}
	var fence *RoleFenceError
	if !errors.As(err, &fence) {
		t.Fatalf("mid-drain ingest failure error = %v, want *RoleFenceError", err)
	}
	if fence.Role != RoleActive {
		t.Fatalf("fence evidence role = %v, want %v (the simulated concurrent promotion)", fence.Role, RoleActive)
	}
	if health.Cursor != events[1].Cursor || health.EventLag != 2 {
		t.Fatalf("mid-drain health = %+v, want cursor at events[1] and 2 remaining", health)
	}
	got, err := replica.After(ctx, Cursor{})
	if err != nil || len(got) != 2 {
		t.Fatalf("replica after mid-drain fence = %#v, %v; want exactly the 2 applied events", got, err)
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != 2 {
		t.Fatalf("outbox after mid-drain fence = %d rows, %v; want exactly the 2 not-yet-applied rows", len(pending), err)
	}
	for _, event := range pending {
		if event.Cursor.Sequence <= 2 {
			t.Fatalf("outbox after mid-drain fence retained an already-applied row: %+v", event)
		}
	}

	// Resume: un-fence the replica (a real caller would do this by promoting
	// it back, or simply because the simulated race is over) and drain again.
	replica.mu.Lock()
	replica.role = RoleReplica
	replica.mu.Unlock()
	resumed, err := DrainOutbox(ctx, source, replica, "mirror")
	if err != nil {
		t.Fatalf("resume after un-fencing: %v", err)
	}
	if resumed.EventLag != 0 || resumed.Cursor != events[len(events)-1].Cursor {
		t.Fatalf("resume health = %+v, want fully drained", resumed)
	}
	finalPending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(finalPending) != 0 {
		t.Fatalf("outbox after resume = %#v, %v; want empty", finalPending, err)
	}
}

// TestDrainOutboxConcurrentDrainersDoNotDoubleApply runs two independent
// DrainOutbox calls against the same source outbox and the same replica at
// the same time. Neither call coordinates with the other — the safety
// property comes entirely from Journal.IngestReplica's idempotent dedup-by-
// event-ID and AckOutbox's idempotent delete, exercised here under real
// goroutine concurrency rather than argued about.
func TestDrainOutboxConcurrentDrainersDoNotDoubleApply(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)

	var wg sync.WaitGroup
	results := make([]error, 2)
	start := make(chan struct{})
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := DrainOutbox(ctx, source, replica, "mirror")
			results[i] = err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range results {
		if err != nil {
			t.Fatalf("racing drainer %d: %v", i, err)
		}
	}
	got, err := replica.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(events) {
		t.Fatalf("replica has %d events after racing drainers, want exactly %d (no double-apply)", len(got), len(events))
	}
	for i, event := range got {
		if event.Checksum != events[i].Checksum {
			t.Fatalf("replica event %d checksum = %q, want %q", i, event.Checksum, events[i].Checksum)
		}
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != 0 {
		t.Fatalf("outbox after racing drainers = %#v, %v; want fully acked", pending, err)
	}
}

// TestDrainOutboxAcksEventsAlreadyDeliveredByDirectReplicate proves the
// outbox drain path and Replicate's direct head/cursor comparison remain
// coherent when mixed: an operator who already ran Replicate (which never
// touches the outbox) still gets a fully-acked outbox from a later
// DrainOutbox call, because IngestReplica no-ops on the already-present
// events and AckOutbox still deletes their now-stale outbox rows.
func TestDrainOutboxAcksEventsAlreadyDeliveredByDirectReplicate(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)

	if _, err := Replicate(ctx, source, replica, "mirror"); err != nil {
		t.Fatalf("direct replicate: %v", err)
	}
	pendingBeforeDrain, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pendingBeforeDrain) != len(events) {
		t.Fatalf("outbox before drain = %d rows, %v; want all %d rows still queued (Replicate does not touch the outbox)", len(pendingBeforeDrain), err, len(events))
	}

	health, err := DrainOutbox(ctx, source, replica, "mirror")
	if err != nil {
		t.Fatalf("drain after direct replicate: %v", err)
	}
	if health.EventLag != 0 {
		t.Fatalf("health = %+v, want no lag", health)
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != 0 {
		t.Fatalf("outbox after drain = %#v, %v; want empty", pending, err)
	}
	got, err := replica.After(ctx, Cursor{})
	if err != nil || len(got) != len(events) {
		t.Fatalf("replica after drain = %#v, %v; want exactly %d events (no double-apply from the earlier Replicate)", got, err, len(events))
	}
}
