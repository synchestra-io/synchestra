package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// tickingClock returns a Now func that advances by one second on every call,
// starting at base. Deterministic (no wall-clock flakiness) but strictly
// increasing, so assertions like "RenewedAt advanced" are meaningful.
func tickingClock(base time.Time) func() time.Time {
	var mu sync.Mutex
	next := base
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		current := next
		next = next.Add(time.Second)
		return current
	}
}

func newTestStore(t *testing.T, journal replication.Journal, epoch int64, actorID string) *Store {
	t.Helper()
	store, err := New(journal, Options{
		ProjectID: "github.com/fair-split/relay", ActorID: actorID, AuthorityEpoch: epoch,
		Now: tickingClock(time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store
}

func TestLeaseAcquireIsExclusivePerResource(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, replication.NewMemoryJournal(), 1, "server")
	lease, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-a"})
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if lease.ID == "" || lease.Fence.Token == "" || lease.Fence.Epoch != 1 {
		t.Fatalf("unexpected lease: %+v", lease)
	}
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-b"}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("second Acquire error = %v, want ErrConflict", err)
	}
	// Releasing frees the resource for a fresh Acquire — claim/release/re-
	// claim cycles must not exhaust the resource key.
	if err := store.Lease().Release(ctx, lease.ID, lease.Fence); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-b"}); err != nil {
		t.Fatalf("Acquire after release: %v", err)
	}
}

// TestLeaseAcquireIsExclusiveUnderConcurrency proves the uniqueness guarantee
// holds under real goroutine concurrency, not just sequential simulation:
// every goroutine reads an empty projection before any of them append, yet
// exactly one Acquire succeeds. This is the mechanism behind
// agent-coordination#ac:one-writer-claim-is-fenced.
func TestLeaseAcquireIsExclusiveUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, replication.NewMemoryJournal(), 1, "server")
	const racers = 12
	var wg sync.WaitGroup
	results := make(chan error, racers)
	var successes int32
	var mu sync.Mutex
	var winners []state.AuthorityLease
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			lease, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run"})
			if err == nil {
				mu.Lock()
				successes++
				winners = append(winners, lease)
				mu.Unlock()
			}
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		if err == nil {
			continue
		}
		if !errors.Is(err, state.ErrConflict) {
			t.Fatalf("unexpected concurrent Acquire error: %v", err)
		}
		conflicts++
	}
	if successes != 1 {
		t.Fatalf("successful Acquire calls = %d, want exactly 1 (winners=%+v)", successes, winners)
	}
	if conflicts != racers-1 {
		t.Fatalf("conflicting Acquire calls = %d, want %d", conflicts, racers-1)
	}
}

func TestLeaseRenewRejectsStaleFence(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, replication.NewMemoryJournal(), 1, "server")
	lease, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run-a"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	staleFence := state.LeaseFence{Epoch: lease.Fence.Epoch, Token: "not-the-real-token"}
	if _, err := store.Lease().Renew(ctx, lease.ID, staleFence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Renew with wrong token error = %v, want ErrLeaseFenced", err)
	}
	renewed, err := store.Lease().Renew(ctx, lease.ID, lease.Fence)
	if err != nil {
		t.Fatalf("Renew with correct fence: %v", err)
	}
	if renewed.Fence != lease.Fence {
		t.Fatalf("renew changed fence unexpectedly: got %+v want %+v", renewed.Fence, lease.Fence)
	}
}

func TestLeaseReleaseRejectsStaleFenceAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, replication.NewMemoryJournal(), 1, "server")
	lease, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run-a"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := store.Lease().Release(ctx, lease.ID, state.LeaseFence{Epoch: lease.Fence.Epoch, Token: "wrong"}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Release with wrong token error = %v, want ErrLeaseFenced", err)
	}
	if err := store.Lease().Release(ctx, lease.ID, lease.Fence); err != nil {
		t.Fatalf("Release: %v", err)
	}
	// A second release of an already-released lease is not an error.
	if err := store.Lease().Release(ctx, lease.ID, lease.Fence); err != nil {
		t.Fatalf("second Release: %v", err)
	}
	if _, err := store.Lease().Get(ctx, "r"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Get after release error = %v, want ErrNotFound", err)
	}
}

// TestPromotionFencesFormerActiveStoreOnNextWrite is the direct proof for
// state-store/topology#ac:promotion-fences-former-active's claim/lease half:
// once a new active endpoint (epoch 2) has recorded at least one event on
// the shared journal, a Store instance still configured for the old epoch
// (1) can never again succeed at Lease().Acquire — every attempt is rejected
// as fenced, exactly like DALJournal enforces for Git/SQLite (see
// pkg/state/replication/dal_journal.go's Append entry check plus
// validateNext's epoch comparison, which MemoryJournal mirrors for this
// backend-neutral conformance test).
func TestPromotionFencesFormerActiveStoreOnNextWrite(t *testing.T) {
	ctx := context.Background()
	journal := replication.NewMemoryJournal()
	oldActive := newTestStore(t, journal, 1, "old-active")
	newActive := newTestStore(t, journal, 2, "new-active")

	// The old active successfully holds a lease before promotion.
	lease, err := oldActive.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:feature", HolderRunID: "run-old"})
	if err != nil {
		t.Fatalf("pre-promotion Acquire: %v", err)
	}

	// Promotion: the new active endpoint writes at epoch 2. In production
	// this event is the recorded promotion checkpoint (topology spec,
	// "Promotion and Recovery"); for this contract-level test any epoch-2
	// event is sufficient to move the journal head past epoch 1.
	if _, err := newActive.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:other", HolderRunID: "run-new"}); err != nil {
		t.Fatalf("promotion write at new epoch: %v", err)
	}

	// The former active can no longer renew its pre-promotion lease...
	if _, err := oldActive.Lease().Renew(ctx, lease.ID, lease.Fence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("post-promotion Renew on old active error = %v, want ErrLeaseFenced", err)
	}
	// ...nor acquire a brand new one.
	if _, err := oldActive.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:another", HolderRunID: "run-old"}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("post-promotion Acquire on old active error = %v, want ErrLeaseFenced", err)
	}
	// The new active is unaffected and keeps working normally.
	if _, err := newActive.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:third", HolderRunID: "run-new"}); err != nil {
		t.Fatalf("new active Acquire after promotion: %v", err)
	}
}
