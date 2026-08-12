package replication

// Features implemented: state-store/topology

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// TestDrainOutboxDeliversInCursorOrderAndAcksExactlyOnce is the basic happy
// path: every pending row is applied to the replica and acked (deleted) from
// the source, in cursor order, and a second drain of the now-empty outbox is
// a true no-op.
func TestDrainOutboxDeliversInCursorOrderAndAcksExactlyOnce(t *testing.T) {
	ctx := context.Background()
	source := NewMemoryJournal("mirror")
	replica := NewMemoryJournal()
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

// ackFailer wraps an OutboxSource and fails AckOutbox exactly once for one
// chosen event, simulating a process crash after the replica durably applied
// the event but before the source's outbox row was deleted. This is the
// specific window the drain/ack design must survive without loss or
// double-apply.
type ackFailer struct {
	OutboxSource
	failFor string
	failed  bool
}

func (a *ackFailer) AckOutbox(ctx context.Context, replicaID, eventID string) error {
	if !a.failed && eventID == a.failFor {
		a.failed = true
		return errors.New("simulated crash between apply and ack")
	}
	return a.OutboxSource.AckOutbox(ctx, replicaID, eventID)
}

// TestDrainOutboxCrashBetweenApplyAndAckConvergesOnResume kills delivery
// immediately after the replica accepts an event but before its outbox row
// is acked, then resumes with a fresh DrainOutbox call standing in for a
// restarted worker. The event must not be applied twice, and the outbox must
// end up fully drained.
func TestDrainOutboxCrashBetweenApplyAndAckConvergesOnResume(t *testing.T) {
	ctx := context.Background()
	source := NewMemoryJournal("mirror")
	replica := NewMemoryJournal()
	events := relayEvents(t)
	appendAll(t, source, events)

	failing := &ackFailer{OutboxSource: source, failFor: events[1].EventID}
	health, err := DrainOutbox(ctx, failing, replica, "mirror")
	if err == nil {
		t.Fatal("expected the injected crash between apply and ack to surface")
	}
	if health.Cursor != events[1].Cursor {
		t.Fatalf("crash health cursor = %+v, want the applied-but-unacked event's cursor %+v", health.Cursor, events[1].Cursor)
	}
	// The replica already durably has events[0] and events[1] — the crash is
	// strictly after apply — but the source outbox still holds events[1]
	// onward because the ack for events[1] never completed.
	got, err := replica.After(ctx, Cursor{})
	if err != nil || len(got) != 2 {
		t.Fatalf("replica after crash = %#v, %v; want exactly the 2 applied events", got, err)
	}
	pending, err := source.PendingOutbox(ctx, "mirror")
	if err != nil || len(pending) != len(events)-1 {
		t.Fatalf("outbox after crash = %d rows, %v; want %d unacked rows", len(pending), err, len(events)-1)
	}

	// Resume: a fresh drain redelivers events[1] (idempotent no-op on the
	// already-caught-up replica), completes its ack, and finishes the rest.
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

// TestDrainOutboxConcurrentDrainersDoNotDoubleApply runs two independent
// DrainOutbox calls against the same source outbox and the same replica at
// the same time. Neither call coordinates with the other — the safety
// property comes entirely from Journal.IngestReplica's idempotent dedup-by-
// event-ID and AckOutbox's idempotent delete, exercised here under real
// goroutine concurrency rather than argued about.
func TestDrainOutboxConcurrentDrainersDoNotDoubleApply(t *testing.T) {
	ctx := context.Background()
	source := NewMemoryJournal("mirror")
	replica := NewMemoryJournal()
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
	source := NewMemoryJournal("mirror")
	replica := NewMemoryJournal()
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
