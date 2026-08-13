package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology, state-store/journal-batching

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// zeroBatch is a shared "disable batching" knob pointer for this package's
// journal fixtures (state-store/journal-batching). Most of this package's
// tests append events one at a time and assert on each Append's immediate
// outcome; without this, the journal's own documented default (100 items /
// 1000ms, state-store/journal-batching#ac:defaults-are-100-items-1000ms)
// would make every sequential single Append wait out the full window before
// returning. Tests that specifically exercise batching (e.g. lease
// uniqueness under concurrency with batching enabled) construct their own
// explicit small window instead of using this.
var zeroBatch = func() *int { v := 0; return &v }()

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

// newTestJournal builds a fresh, valid active MemoryJournal for pairing with
// newTestStore. Its ProjectID must match newTestStore's fixed ProjectID
// ("github.com/fair-split/relay") and its AuthorityEpoch must match the
// epoch the paired Store is constructed with -- MemoryJournal (like
// DALJournal) enforces both on every Append, and either mismatch compiles
// cleanly but fails at runtime (a project mismatch errors immediately, an
// epoch mismatch surfaces as a RoleFenceError/state.ErrLeaseFenced) rather
// than at construction, so centralizing construction here is what keeps the
// two values from drifting apart across this package's tests.
func newTestJournal(t *testing.T, epoch int64, endpointID string) *replication.MemoryJournal {
	t.Helper()
	journal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: "github.com/fair-split/relay", EndpointID: endpointID, Role: replication.RoleActive, AuthorityEpoch: epoch,
		MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch,
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal: %v", err)
	}
	return journal
}

func TestLeaseAcquireIsExclusivePerResource(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
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

// TestLeaseAcquireRejectsNegativeTTL proves a negative TTL is refused
// outright rather than silently folded into TTL<=0's "no TTL, never
// expires" behavior (leaseExpired, lease.go) -- a negative duration is
// almost always a caller bug (e.g. a subtraction done backwards), and
// granting an indefinite, unreclaimable lease in response would hide it
// rather than surface it.
func TestLeaseAcquireRejectsNegativeTTL(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{
		Resource: "r", HolderRunID: "run-a", TTL: -time.Minute,
	}); !errors.Is(err, state.ErrInvalidArgument) {
		t.Fatalf("Acquire with a negative TTL error = %v, want state.ErrInvalidArgument", err)
	}
	// The rejected Acquire must not have left a partial/held lease behind.
	if _, err := store.Lease().Get(ctx, "r"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Get after rejected negative-TTL Acquire error = %v, want ErrNotFound", err)
	}
}

// TestLeaseAcquireIsExclusiveUnderConcurrency proves the uniqueness guarantee
// holds under real goroutine concurrency, not just sequential simulation:
// every goroutine reads an empty projection before any of them append, yet
// exactly one Acquire succeeds. This is the mechanism behind
// agent-coordination#ac:one-writer-claim-is-fenced.
func TestLeaseAcquireIsExclusiveUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
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

// TestLeaseAcquireIsExclusiveUnderConcurrencyWithBatchingEnabled is
// TestLeaseAcquireIsExclusiveUnderConcurrency's group-commit-batching
// sibling (state-store/journal-batching): the same 12 racers now share a
// journal with a small but nonzero batching window, so their first-round
// tryAppendAt calls are very likely to land in the SAME physical batch
// rather than each committing in its own transaction. Store.Lease().Acquire
// re-derives its projection on every retry of a lost sequence race
// (lease.go's doc comment), and a batch's own commitOneLocked/appendData
// loop resolves a same-cursor collision the identical way an unbatched
// per-event race does (exactly one winner, everyone else ErrSequenceGap ->
// agentstore's errSequenceRace -> retried) -- so this must converge to the
// same exactly-one-winner outcome, proving appendWithRetry/CAS's retry
// ladder is unchanged by batching, just as
// pkg/state/replication/batch_test.go's
// TestBatchedEventsPreserveFencingAndOrder proves at the journal level.
func TestLeaseAcquireIsExclusiveUnderConcurrencyWithBatchingEnabled(t *testing.T) {
	ctx := context.Background()
	journal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: "github.com/fair-split/relay", EndpointID: "server", Role: replication.RoleActive, AuthorityEpoch: 1,
		// A small window keeps the test fast while still forcing real
		// group-commit batching: MaxItems below the racer count means the
		// time window is what actually triggers at least the first flush.
		MaxBatchItems: intPtrForTest(6), MaxBatchDelayMS: intPtrForTest(20),
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal: %v", err)
	}
	if journal.BatchSettings() != (replication.BatchSettings{MaxItems: 6, MaxDelayMS: 20}) {
		t.Fatalf("BatchSettings = %+v, want the configured small window (batching must actually be enabled for this test to prove anything)", journal.BatchSettings())
	}
	store := newTestStore(t, journal, 1, "server")
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
			t.Fatalf("unexpected concurrent Acquire error under batching: %v", err)
		}
		conflicts++
	}
	if successes != 1 {
		t.Fatalf("successful Acquire calls under batching = %d, want exactly 1 (winners=%+v)", successes, winners)
	}
	if conflicts != racers-1 {
		t.Fatalf("conflicting Acquire calls under batching = %d, want %d", conflicts, racers-1)
	}
}

// intPtrForTest avoids colliding with any *int helper another file in this
// package might define; kept local and trivial on purpose.
func intPtrForTest(v int) *int { return &v }

func TestLeaseRenewRejectsStaleFence(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
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
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
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
// once a real replication.Promote() call has fenced the former active and
// promoted the candidate to active at the new epoch, a Store instance still
// configured for the old epoch can never again succeed at Lease().Acquire —
// every attempt is rejected as fenced, exactly like DALJournal enforces for
// Git/SQLite (see pkg/state/replication/dal_journal.go's Append entry check
// plus validateNext's epoch comparison, which MemoryJournal mirrors for this
// backend-neutral conformance test).
//
// This drives the real fence-first Promote() workflow across two distinct
// MemoryJournals, one per endpoint -- the same topology production
// promotion uses -- rather than the pre-redesign trick of appending
// different-epoch events onto one journal shared by two Store instances,
// which the fence-first MemoryJournal now correctly refuses with a
// RoleFenceError (a MemoryJournal has exactly one role/epoch for its whole
// lifetime; role/epoch transitions only ever happen through
// FenceAsReplica/PromoteToActive, i.e. through Promote).
func TestPromotionFencesFormerActiveStoreOnNextWrite(t *testing.T) {
	ctx := context.Background()
	const projectID = "github.com/fair-split/relay"
	activeJournal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: projectID, EndpointID: "old-active", Role: replication.RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch,
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal(active): %v", err)
	}
	candidateJournal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: projectID, EndpointID: "new-active", Role: replication.RoleReplica, AuthorityEpoch: 1,
		MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch,
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal(candidate): %v", err)
	}
	oldActive := newTestStore(t, activeJournal, 1, "old-active")
	newActive := newTestStore(t, candidateJournal, 2, "new-active")

	// The old active successfully holds a lease before promotion.
	lease, err := oldActive.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:feature", HolderRunID: "run-old"})
	if err != nil {
		t.Fatalf("pre-promotion Acquire: %v", err)
	}

	// Promotion: fence the old active and promote the candidate to active at
	// the new epoch via the real fence-first workflow (fence active -> catch
	// up candidate -> promote candidate), exactly as an operator would run
	// it in production.
	if _, err := replication.Promote(ctx, activeJournal, candidateJournal, nil, replication.PromotionRequest{
		ActorID: "operator", CommandID: "promote-fences-former-active",
	}); err != nil {
		t.Fatalf("Promote: %v", err)
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
