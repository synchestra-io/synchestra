package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/topology, agent-coordination

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func newCheckpointSource(t *testing.T) (*MemoryJournal, []Event) {
	t.Helper()
	source, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	events := relayEvents(t)
	appendAll(t, source, events)
	return source, events
}

func TestNewCheckpointRefusesAnEmptyJournal(t *testing.T) {
	source, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewCheckpoint(context.Background(), source, "active", Cursor{}); !errors.Is(err, ErrCheckpointEmpty) {
		t.Fatalf("err = %v, want ErrCheckpointEmpty", err)
	}
}

func TestNewCheckpointAtZeroCursorCapturesCurrentHead(t *testing.T) {
	ctx := context.Background()
	source, events := newCheckpointSource(t)
	head, headHash, err := source.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}

	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatalf("NewCheckpoint: %v", err)
	}
	if checkpoint.Cursor != head || checkpoint.ContentHash != headHash {
		t.Fatalf("checkpoint cursor/hash = %v/%s, want head %v/%s", checkpoint.Cursor, checkpoint.ContentHash, head, headHash)
	}
	if checkpoint.EventCount != len(events) || len(checkpoint.Events) != len(events) {
		t.Fatalf("checkpoint has %d events, want %d", len(checkpoint.Events), len(events))
	}
	if checkpoint.SourceID != "active" || checkpoint.ProjectID != "github.com/fair-split/relay" {
		t.Fatalf("checkpoint identity = %+v", checkpoint)
	}
	if err := checkpoint.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestNewCheckpointAtMidJournalCursorCapturesExactPrefix(t *testing.T) {
	ctx := context.Background()
	source, events := newCheckpointSource(t)
	mid := events[1].Cursor // second of four events

	checkpoint, err := NewCheckpoint(ctx, source, "active", mid)
	if err != nil {
		t.Fatalf("NewCheckpoint: %v", err)
	}
	if checkpoint.Cursor != mid || checkpoint.ContentHash != events[1].Checksum {
		t.Fatalf("checkpoint cursor/hash = %v/%s, want %v/%s", checkpoint.Cursor, checkpoint.ContentHash, mid, events[1].Checksum)
	}
	if checkpoint.EventCount != 2 {
		t.Fatalf("checkpoint has %d events, want the first 2", checkpoint.EventCount)
	}
	if err := checkpoint.Verify(); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestNewCheckpointRejectsCursorAheadOfHead(t *testing.T) {
	ctx := context.Background()
	source, events := newCheckpointSource(t)
	beyond := Cursor{Epoch: events[len(events)-1].Cursor.Epoch, Sequence: events[len(events)-1].Cursor.Sequence + 1}
	if _, err := NewCheckpoint(ctx, source, "active", beyond); err == nil {
		t.Fatal("expected an error for a cursor beyond the source head")
	}
}

func TestNewCheckpointRejectsCursorWithNoMatchingEvent(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	// Epoch 1 sequence 3 exists; epoch 2 sequence 1 does not (no promotion
	// ever happened on this journal).
	if _, err := NewCheckpoint(ctx, source, "active", Cursor{Epoch: 2, Sequence: 1}); err == nil {
		t.Fatal("expected an error for a cursor beyond the source head")
	}
}

func TestCheckpointVerifyRejectsTamperedContent(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatal(err)
	}

	tampered := checkpoint
	tampered.Events = append([]Event(nil), checkpoint.Events...)
	tampered.Events[0].Payload = []byte(`{"body":"forged"}`)
	if err := tampered.Verify(); !errors.Is(err, ErrCheckpointInvalid) {
		t.Fatalf("err = %v, want ErrCheckpointInvalid for a tampered payload", err)
	}

	tamperedHash := checkpoint
	tamperedHash.ContentHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := tamperedHash.Verify(); !errors.Is(err, ErrCheckpointInvalid) {
		t.Fatalf("err = %v, want ErrCheckpointInvalid for a forged content hash", err)
	}

	truncated := checkpoint
	truncated.EventCount = len(checkpoint.Events) - 1
	if err := truncated.Verify(); !errors.Is(err, ErrCheckpointInvalid) {
		t.Fatalf("err = %v, want ErrCheckpointInvalid for a declared/actual event-count mismatch", err)
	}
}

func TestRestoreReplaysACheckpointIntoAFreshReplicaAndConverges(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatal(err)
	}

	target, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "restored", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}

	health, err := Restore(ctx, checkpoint, target, "restored")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if health.Cursor != checkpoint.Cursor || health.EventLag != 0 {
		t.Fatalf("restore health = %+v, want caught up at %v", health, checkpoint.Cursor)
	}

	verify, err := VerifyConvergence(ctx, source, target, Cursor{})
	if err != nil {
		t.Fatalf("VerifyConvergence: %v", err)
	}
	if !verify.Equivalent {
		t.Fatalf("restored target diverged from source: %+v", verify)
	}
}

func TestRestoreRefusesANonEmptyTarget(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatal(err)
	}

	target, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "restored", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.IngestReplica(ctx, checkpoint.Events[0]); err != nil {
		t.Fatal(err)
	}

	if _, err := Restore(ctx, checkpoint, target, "restored"); !errors.Is(err, ErrRestoreTargetNotEmpty) {
		t.Fatalf("err = %v, want ErrRestoreTargetNotEmpty", err)
	}
}

// TestRestoreNeverPopulatesAnActiveEndpoint is the structural proof behind
// state-store/topology's "a restored endpoint joins as a replica, never
// silently as an active" doctrine: Restore writes exclusively through
// ReplicaIngestor, so a target mistakenly constructed with Role: RoleActive
// fails closed on the very first event with the ordinary RoleFenceError,
// exactly like any other misdirected IngestReplica call would.
func TestRestoreNeverPopulatesAnActiveEndpoint(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatal(err)
	}

	activeTarget, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "not-a-replica", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Restore(ctx, checkpoint, activeTarget, "not-a-replica")
	var fenceErr *RoleFenceError
	if !errors.As(err, &fenceErr) {
		t.Fatalf("err = %v (%T), want *RoleFenceError -- restore must never silently write onto an active endpoint", err, err)
	}
	if head, _, _ := activeTarget.Head(ctx); !head.IsZero() {
		t.Fatalf("active target head = %v after a refused restore, want zero (no partial write)", head)
	}
}

func TestRestoreRefusesAnInvalidCheckpointWithoutTouchingTarget(t *testing.T) {
	ctx := context.Background()
	source, _ := newCheckpointSource(t)
	checkpoint, err := NewCheckpoint(ctx, source, "active", Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint.ContentHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	target, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "restored", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(ctx, checkpoint, target, "restored"); !errors.Is(err, ErrCheckpointInvalid) {
		t.Fatalf("err = %v, want ErrCheckpointInvalid", err)
	}
	if head, _, _ := target.Head(ctx); !head.IsZero() {
		t.Fatalf("target head = %v after a refused restore, want zero", head)
	}
}

func TestVerifyConvergenceDetectsDivergence(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	if _, err := DrainOutbox(ctx, source, replica, "mirror"); err != nil {
		t.Fatal(err)
	}

	verify, err := VerifyConvergence(ctx, source, replica, Cursor{})
	if err != nil {
		t.Fatalf("VerifyConvergence: %v", err)
	}
	if !verify.Equivalent || verify.SourceHash != verify.TargetHash {
		t.Fatalf("verify = %+v, want equivalent caught-up replica", verify)
	}
	if verify.AtCursor != events[len(events)-1].Cursor {
		t.Fatalf("verify.AtCursor = %v, want the shared head %v", verify.AtCursor, events[len(events)-1].Cursor)
	}
}

// TestVerifyConvergenceReportsDivergenceWhenChecksumsDiffer is the negative
// case behind ac:checkpoint-verify-detects-divergence: every other
// VerifyConvergence test in this file proves the CONVERGENT case (matching
// checksum chains report Equivalent: true); this one proves the actual
// divergence-DETECTING half by comparing two independently constructed
// journals that reach the identical cursor with different content -- e.g.
// what a backend bug or storage corruption producing a mismatched event at
// the same sequence number would look like. VerifyConvergence must report
// Equivalent: false, never infer convergence from matching cursors alone.
func TestVerifyConvergenceReportsDivergenceWhenChecksumsDiffer(t *testing.T) {
	ctx := context.Background()
	source, events := newCheckpointSource(t)

	diverged, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "diverged", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	// The first three events are identical to source's; the fourth is an
	// independently constructed event landing at the SAME cursor with
	// DIFFERENT payload content, so its checksum (and therefore the head
	// hash) diverges from source's even though both journals report the
	// identical cursor.
	for _, event := range events[:3] {
		if err := diverged.IngestReplica(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	last := events[3]
	payload, err := json.Marshal(map[string]any{"thread_id": "fair-split", "body": "decision.accepted", "result": "DIVERGED"})
	if err != nil {
		t.Fatal(err)
	}
	mismatched, err := NewEvent(Event{
		ProjectID: last.ProjectID, EventID: "decision-diverged",
		Cursor: last.Cursor, OccurredAt: last.OccurredAt, ActorID: last.ActorID,
		CommandID: last.CommandID, IdempotencyKey: "decision-diverged", Kind: last.Kind,
		CorrelationID: last.CorrelationID, Payload: payload, PreviousHash: events[2].Checksum,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := diverged.IngestReplica(ctx, mismatched); err != nil {
		t.Fatal(err)
	}

	verify, err := VerifyConvergence(ctx, source, diverged, Cursor{})
	if err != nil {
		t.Fatalf("VerifyConvergence: %v", err)
	}
	if verify.AtCursor != last.Cursor {
		t.Fatalf("verify.AtCursor = %v, want the shared cursor %v", verify.AtCursor, last.Cursor)
	}
	if verify.Equivalent || verify.SourceHash == verify.TargetHash {
		t.Fatalf("verify = %+v, want Equivalent: false for mismatched content at the same cursor", verify)
	}
}

func TestVerifyConvergenceComparesAtTheLowerSharedCursorWhenNotSpecified(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	// Deliver only the first two events -- replica intentionally lags.
	for _, event := range events[:2] {
		if err := replica.IngestReplica(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	verify, err := VerifyConvergence(ctx, source, replica, Cursor{})
	if err != nil {
		t.Fatalf("VerifyConvergence: %v", err)
	}
	if verify.AtCursor != events[1].Cursor {
		t.Fatalf("verify.AtCursor = %v, want the replica's lagging cursor %v", verify.AtCursor, events[1].Cursor)
	}
	if !verify.Equivalent {
		t.Fatalf("verify = %+v, want equivalent at the shared (lower) cursor", verify)
	}
}
