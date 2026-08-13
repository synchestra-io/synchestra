package agentstore

// Features implemented: state-store, agent-coordination

// This file recreates, deterministically, the double-writer race an
// adversarial review reproduced against Renew/Release/Transfer's
// pre-fix shape ("read the lease projection once, check the fence, then
// appendWithRetry — which re-reads Head for the append and retries WITHOUT
// re-checking the fence"): a concurrent TTL reclaim committing in the window
// between a resumed caller's fence check and its append must fence that
// caller, never let it silently succeed against a lease that is no longer
// its own (agent-coordination#ac:one-writer-claim-is-fenced). Every test
// here uses Store.testFenceCheckSeam (store.go) to pause a goroutine exactly
// inside that window instead of relying on goroutine-scheduling luck.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// pauseOnceAt returns a testFenceCheckSeam hook that pauses the FIRST call
// matching (wantOp, wantLeaseID) — closing paused, then blocking until
// resume is closed — and is a no-op for every other call (including later
// retries of the same op/leaseID once the race has already been decided).
func pauseOnceAt(wantOp, wantLeaseID string, paused, resume chan struct{}) func(op, leaseID string) {
	var once sync.Once
	return func(op, leaseID string) {
		if op != wantOp || leaseID != wantLeaseID {
			return
		}
		once.Do(func() {
			close(paused)
			<-resume
		})
	}
}

// TestLeaseRenewRefusesConcurrentReclaimInFenceCheckWindow is the direct,
// lease-layer reproduction: run-dead's Renew is paused immediately after its
// fence check passes (it still legitimately holds the lease at that
// instant); while paused, run-successor reclaims the same, now TTL-expired,
// resource. The resumed Renew's append then loses the CAS race and MUST
// retry into a typed fence error instead of blindly succeeding — the
// "double-writer" this closes: two runs believing they hold the same
// resource.
func TestLeaseRenewRefusesConcurrentReclaimInFenceCheckWindow(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	original, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-dead", TTL: time.Minute})
	if err != nil {
		t.Fatalf("original Acquire: %v", err)
	}
	clock.Advance(2 * time.Minute) // past the 1-minute TTL

	paused, resume := make(chan struct{}), make(chan struct{})
	store.testFenceCheckSeam = pauseOnceAt("renew", original.ID, paused, resume)

	renewDone := make(chan error, 1)
	go func() {
		_, err := store.Lease().Renew(ctx, original.ID, original.Fence)
		renewDone <- err
	}()

	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("Renew never reached the fence-check seam")
	}

	// A successor reclaims the TTL-expired resource while the dead holder's
	// Renew is paused mid-CAS.
	reclaimed, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-successor", TTL: time.Minute})
	if err != nil {
		t.Fatalf("concurrent reclaim Acquire: %v", err)
	}
	if !reclaimed.Reclaimed {
		t.Fatalf("concurrent Acquire did not reclaim: %+v", reclaimed)
	}

	close(resume)

	select {
	case err := <-renewDone:
		if !errors.Is(err, state.ErrLeaseFenced) {
			t.Fatalf("resumed dead-run Renew after concurrent reclaim error = %v, want ErrLeaseFenced (double-writer if nil)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Renew never returned after resume")
	}

	// The successor's lease is the one and only live grant.
	current, err := store.Lease().Get(ctx, "worktree:p:repo:main")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.HolderRunID != "run-successor" {
		t.Fatalf("current holder = %q, want run-successor (the resumed dead Renew must not have overwritten it)", current.HolderRunID)
	}
}

// TestWorktreeRenewRefusesConcurrentReclaimInFenceCheckWindow is the
// worktree-layer twin: WorktreeStore.Renew delegates its fencing decision
// entirely to LeaseStore.Renew (worktree.go's doc comment — "never
// re-implements check-then-append race handling"), so the SAME seam
// (fired from inside Lease().Renew, which worktree.Renew calls internally)
// proves the fix closes the race at the entry point the review actually
// exercised, not just at the lease package's own direct API.
func TestWorktreeRenewRefusesConcurrentReclaimInFenceCheckWindow(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	params := state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-dead",
		WorktreePath: "/work/relay", Branch: "agent/run-dead", TargetRef: "main",
	}
	original, err := store.Worktree().Claim(ctx, params)
	if err != nil {
		t.Fatalf("original Claim: %v", err)
	}
	clock.Advance(defaultWorktreeLeaseTTL + time.Minute) // past the 1-hour worktree lease TTL

	paused, resume := make(chan struct{}), make(chan struct{})
	// original.ID == the underlying lease ID (worktree.go: "claim ID == the
	// new lease ID"), so this fires from inside worktreeStore.Renew's call
	// to w.store.Lease().Renew.
	store.testFenceCheckSeam = pauseOnceAt("renew", original.ID, paused, resume)

	renewDone := make(chan error, 1)
	go func() {
		_, err := store.Worktree().Renew(ctx, original.ID, original.Fence)
		renewDone <- err
	}()

	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("Worktree Renew never reached the fence-check seam")
	}

	competing := params
	competing.RunID = "run-successor"
	reclaimed, err := store.Worktree().Claim(ctx, competing)
	if err != nil {
		t.Fatalf("competing Claim (reclaim): %v", err)
	}
	if reclaimed.ID == original.ID {
		t.Fatalf("reclaimed claim reused the original claim ID")
	}

	close(resume)

	select {
	case err := <-renewDone:
		if !errors.Is(err, state.ErrLeaseFenced) {
			t.Fatalf("resumed dead-run Worktree Renew after concurrent reclaim error = %v, want ErrLeaseFenced (double-writer if nil)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Worktree Renew never returned after resume")
	}

	// Exactly one active claim for this repository/branch, held by the
	// successor — the dead run's resumed Renew must not have resurrected
	// the abandoned claim as active.
	active, err := store.Worktree().List(ctx, state.WorktreeFilter{RepositoryID: "relay", ActiveOnly: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	activeForBranch := 0
	for _, claim := range active {
		if claim.Branch == params.Branch {
			activeForBranch++
			if claim.RunID != "run-successor" {
				t.Fatalf("active claim held by %q, want run-successor", claim.RunID)
			}
		}
	}
	if activeForBranch != 1 {
		t.Fatalf("active claims for branch %q = %d, want exactly 1", params.Branch, activeForBranch)
	}
}

// TestLeaseTransferRefusesConcurrentReclaimInFenceCheckWindow is the
// Transfer variant: a dead holder attempts to hand its (now TTL-expired)
// lease off to a chosen successor while a DIFFERENT successor reclaims the
// same resource through the ordinary TTL path in the window between
// Transfer's fence check and its append. Transfer must lose this race the
// same way Renew does.
func TestLeaseTransferRefusesConcurrentReclaimInFenceCheckWindow(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	original, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-dead", TTL: time.Minute})
	if err != nil {
		t.Fatalf("original Acquire: %v", err)
	}
	clock.Advance(2 * time.Minute) // past the 1-minute TTL

	paused, resume := make(chan struct{}), make(chan struct{})
	store.testFenceCheckSeam = pauseOnceAt("transfer", original.ID, paused, resume)

	transferDone := make(chan error, 1)
	go func() {
		_, err := store.Lease().Transfer(ctx, original.ID, original.Fence, "run-chosen-successor")
		transferDone <- err
	}()

	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("Transfer never reached the fence-check seam")
	}

	reclaimed, err := store.Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:p:repo:main", HolderRunID: "run-actual-reclaimer", TTL: time.Minute})
	if err != nil {
		t.Fatalf("concurrent reclaim Acquire: %v", err)
	}
	if !reclaimed.Reclaimed {
		t.Fatalf("concurrent Acquire did not reclaim: %+v", reclaimed)
	}

	close(resume)

	select {
	case err := <-transferDone:
		if !errors.Is(err, state.ErrLeaseFenced) {
			t.Fatalf("resumed dead-run Transfer after concurrent reclaim error = %v, want ErrLeaseFenced (double-writer if nil)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Transfer never returned after resume")
	}

	current, err := store.Lease().Get(ctx, "worktree:p:repo:main")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.HolderRunID != "run-actual-reclaimer" {
		t.Fatalf("current holder = %q, want run-actual-reclaimer (the resumed Transfer must not have overwritten it)", current.HolderRunID)
	}
}
