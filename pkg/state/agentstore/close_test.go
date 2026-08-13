package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/journal-batching

import (
	"context"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// TestStoreCloseFlushesPendingBatch proves Store.Close reaches the
// underlying batched journal (state-store/journal-batching#ac:close-
// flushes-pending-batch): a lone Append, blocked waiting for its batch's
// flush trigger against a deliberately huge window, is unblocked promptly
// once Close runs — mirroring
// pkg/state/replication/batch_test.go's TestBatchCloseFlushesPendingBatch,
// but through agentstore.Store's own Close rather than the journal
// directly, proving the duck-typed pass-through (store.go) actually reaches
// it.
func TestStoreCloseFlushesPendingBatch(t *testing.T) {
	ctx := context.Background()
	const hugeWindowMS = 60_000
	journal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: "github.com/fair-split/relay", EndpointID: "server", Role: replication.RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: intPtrForTest(0), MaxBatchDelayMS: intPtrForTest(hugeWindowMS),
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal: %v", err)
	}
	store := newTestStore(t, journal, 1, "server")

	done := make(chan error, 1)
	go func() {
		_, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "r", Title: "t"})
		done <- err
	}()
	// Give the Append a moment to actually enqueue before closing — the same
	// synchronization the replication package's own Close test uses.
	time.Sleep(10 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() { closeDone <- store.Close(ctx) }()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Store.Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Store.Close did not return promptly -- it must flush the pending batch itself rather than waiting out the window")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("pending Effort().Create after Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending Effort().Create never resolved after Close flushed")
	}

	// A second Close is a safe no-op.
	if err := store.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestStoreCloseIsANoOpWithoutBatching proves Close is harmless for a
// journal with batching disabled (the zeroBatch fixture most of this
// package's tests use) — a backend/test double with nothing pending must
// never error just because Close was called.
func TestStoreCloseIsANoOpWithoutBatching(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close on a non-batched journal: %v", err)
	}
}
