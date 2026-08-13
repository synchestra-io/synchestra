package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology, state-store/journal-batching

import (
	"context"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

func TestCursorAdvanceIsMonotonic(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")

	zero, err := store.Cursor().Get(ctx, "dashboard")
	if err != nil {
		t.Fatalf("Get before any Advance: %v", err)
	}
	if !zero.IsZero() {
		t.Fatalf("unexpected initial cursor: %+v", zero)
	}

	if err := store.Cursor().Advance(ctx, "dashboard", state.Cursor{Epoch: 1, Sequence: 5}); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	got, err := store.Cursor().Get(ctx, "dashboard")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != (state.Cursor{Epoch: 1, Sequence: 5}) {
		t.Fatalf("Get after Advance = %+v", got)
	}

	if err := store.Cursor().Advance(ctx, "dashboard", state.Cursor{Epoch: 1, Sequence: 3}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("backward Advance error = %v, want ErrConflict", err)
	}
	if err := store.Cursor().Advance(ctx, "dashboard", state.Cursor{Epoch: 1, Sequence: 5}); err != nil {
		t.Fatalf("Advance to the same position: %v", err)
	}
	if err := store.Cursor().Advance(ctx, "dashboard", state.Cursor{Epoch: 2, Sequence: 1}); err != nil {
		t.Fatalf("Advance across epoch: %v", err)
	}

	// A different consumer starts from zero independently.
	other, err := store.Cursor().Get(ctx, "other-consumer")
	if err != nil {
		t.Fatalf("Get other consumer: %v", err)
	}
	if !other.IsZero() {
		t.Fatalf("unrelated consumer cursor is not independent: %+v", other)
	}
}

func TestHealthReportReflectsConfiguredEndpoint(t *testing.T) {
	ctx := context.Background()
	// AuthorityEpoch must be 1 here: both MemoryJournal and DALJournal require
	// the very first event ever appended to a fresh journal to be epoch
	// 1/sequence 1 (a brand new project always starts at epoch 1; epoch only
	// advances via a recorded promotion event, which this test does not
	// perform). See store.go's nextSequence doc comment. The journal's
	// ProjectID/EndpointID must match the Options passed to New below --
	// MemoryJournal enforces ProjectID on every Append, and a fresh
	// MemoryJournal always starts RoleActive at the given epoch.
	journal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: "p", EndpointID: "sqlite-active", Role: replication.RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch,
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal: %v", err)
	}
	store, err := New(journal, Options{ProjectID: "p", ActorID: "server", EndpointID: "sqlite-active", AuthorityEpoch: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	report, err := store.Health().Report(ctx)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if report.EndpointID != "sqlite-active" || report.Role != "active" || report.AuthorityEpoch != 1 {
		t.Fatalf("unexpected health report: %+v", report)
	}
	if !report.Cursor.IsZero() {
		t.Fatalf("fresh journal should report a zero cursor: %+v", report.Cursor)
	}

	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run"}); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	after, err := store.Health().Report(ctx)
	if err != nil {
		t.Fatalf("Report after write: %v", err)
	}
	if after.Cursor.IsZero() || after.Cursor.Epoch != 1 {
		t.Fatalf("health cursor did not advance: %+v", after)
	}
}
