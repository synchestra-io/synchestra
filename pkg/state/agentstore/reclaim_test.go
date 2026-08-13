package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// manualClock is a Now source a test advances explicitly, unlike
// tickingClock's fixed one-second-per-call cadence — needed here because
// defaultWorktreeLeaseTTL (worktree.go) is a fixed one hour and TTL/expiry
// tests need to jump straight past it without one wall-clock hour of ticks.
type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func newManualClock(base time.Time) *manualClock { return &manualClock{now: base} }

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// TestLeaseAcquireReclaimsExpiredLease is the direct proof for
// agent-coordination#ac:abandoned-run-is-resumable's lease-fencing half: an
// agent dies holding a lease with a TTL; once that TTL has elapsed, another
// agent's Acquire succeeds by reclaiming it, and the dead holder's fence
// token is refused on any later Renew/Release.
func TestLeaseAcquireReclaimsExpiredLease(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	journal := newTestJournal(t, 1, "server")
	store, err := New(journal, Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	original, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-dead", TTL: time.Hour})
	if err != nil {
		t.Fatalf("original Acquire: %v", err)
	}

	// Before expiry, a competing Acquire still conflicts normally.
	clock.Advance(30 * time.Minute)
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-successor", TTL: time.Hour}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("Acquire before expiry error = %v, want ErrConflict", err)
	}

	// run-dead never renews. Once the TTL has elapsed, a successor reclaims.
	clock.Advance(45 * time.Minute) // total 75 minutes since AcquiredAt > 60-minute TTL
	reclaimed, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-successor", TTL: time.Hour})
	if err != nil {
		t.Fatalf("Acquire after expiry: %v", err)
	}
	if !reclaimed.Reclaimed {
		t.Fatalf("reclaimed lease does not report Reclaimed: %+v", reclaimed)
	}
	if reclaimed.SupersededLeaseID != original.ID {
		t.Fatalf("SupersededLeaseID = %q, want %q", reclaimed.SupersededLeaseID, original.ID)
	}
	if reclaimed.HolderRunID != "run-successor" {
		t.Fatalf("reclaimed holder = %q, want run-successor", reclaimed.HolderRunID)
	}
	if reclaimed.Fence == original.Fence {
		t.Fatalf("reclaimed lease reused the dead holder's fence token")
	}

	// The dead holder's fence is now permanently refused for any operation
	// that would extend or assume continued authority (Renew). Release on an
	// already-released lease is documented as a safe idempotent no-op
	// regardless of why it was released (see TestLeaseReleaseRejectsStale
	// FenceAndIsIdempotent) — its end state (resource free/reassigned) is
	// identical either way, so this is not a fencing gap.
	if _, err := store.Lease().Renew(ctx, original.ID, original.Fence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("dead holder Renew error = %v, want ErrLeaseFenced", err)
	}
	if err := store.Lease().Release(ctx, original.ID, original.Fence); err != nil {
		t.Fatalf("dead holder Release (idempotent no-op) error = %v, want nil", err)
	}

	// A THIRD caller can no longer reclaim what the successor now legitimately
	// holds (its own TTL has not elapsed).
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-third", TTL: time.Hour}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("Acquire against the live successor error = %v, want ErrConflict", err)
	}

	// The successor's own Get reflects the reclaimed lease.
	current, err := store.Lease().Get(ctx, "worktree:p:repo:main")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.ID != reclaimed.ID || current.HolderRunID != "run-successor" {
		t.Fatalf("Get after reclaim = %+v, want the reclaimed lease", current)
	}
}

// TestLeaseAcquireWithoutTTLNeverExpires proves TTL<=0 preserves the
// pre-task-3 behavior: a lease with no TTL is never reclaimable, no matter
// how much time passes.
func TestLeaseAcquireWithoutTTLNeverExpires(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run-a"}); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	clock.Advance(24 * time.Hour)
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run-b"}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("Acquire a day later without TTL error = %v, want ErrConflict (never expires)", err)
	}
}

// TestLeaseAcquireReclaimUnderConcurrencyHasExactlyOneWinner proves the
// exclusivity guarantee holds under real goroutine concurrency for the
// reclaim path too, not just the fresh-acquire path
// TestLeaseAcquireIsExclusiveUnderConcurrency already covers.
func TestLeaseAcquireReclaimUnderConcurrencyHasExactlyOneWinner(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-dead", TTL: time.Minute}); err != nil {
		t.Fatalf("original Acquire: %v", err)
	}
	clock.Advance(2 * time.Minute) // past the 1-minute TTL

	const racers = 12
	var wg sync.WaitGroup
	var successes int32
	results := make(chan error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-successor", TTL: time.Minute})
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	conflicts := 0
	for err := range results {
		if err == nil {
			continue
		}
		if !errors.Is(err, state.ErrConflict) {
			t.Fatalf("unexpected concurrent reclaim error: %v", err)
		}
		conflicts++
	}
	if successes != 1 {
		t.Fatalf("successful reclaims = %d, want exactly 1", successes)
	}
	if conflicts != racers-1 {
		t.Fatalf("conflicting reclaims = %d, want %d", conflicts, racers-1)
	}
}

// TestLeaseTransferMovesHolderAndFencesOldToken proves voluntary handoff
// (LeaseStore.Transfer): the outgoing holder's fence stops working the
// instant the transfer commits, and the new holder's fresh fence works.
func TestLeaseTransferMovesHolderAndFencesOldToken(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	lease, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "r", HolderRunID: "run-a"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	if _, err := store.Lease().Transfer(ctx, lease.ID, state.LeaseFence{Epoch: lease.Fence.Epoch, Token: "wrong"}, "run-b"); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Transfer with wrong fence error = %v, want ErrLeaseFenced", err)
	}

	transferred, err := store.Lease().Transfer(ctx, lease.ID, lease.Fence, "run-b")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if transferred.ID != lease.ID {
		t.Fatalf("Transfer changed the lease ID: got %q, want %q (same lease, new holder)", transferred.ID, lease.ID)
	}
	if transferred.HolderRunID != "run-b" {
		t.Fatalf("transferred holder = %q, want run-b", transferred.HolderRunID)
	}
	if transferred.Fence == lease.Fence {
		t.Fatalf("Transfer reused the old holder's fence token")
	}

	// The old holder's fence is refused; the new holder's fence works.
	if _, err := store.Lease().Renew(ctx, lease.ID, lease.Fence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("old holder Renew error = %v, want ErrLeaseFenced", err)
	}
	if _, err := store.Lease().Renew(ctx, lease.ID, transferred.Fence); err != nil {
		t.Fatalf("new holder Renew: %v", err)
	}
}
