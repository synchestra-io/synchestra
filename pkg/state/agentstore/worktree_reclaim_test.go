package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// TestWorktreeClaimReclaimsAbandonedClaim is the worktree-layer proof for
// agent-coordination#ac:abandoned-run-is-resumable: an agent dies holding a
// worktree claim (defaultWorktreeLeaseTTL, one hour); once that TTL elapses,
// a successor's Claim succeeds, the abandoned claim is visibly released (not
// left dangling as a phantom "active" record), and the dead run's fence is
// refused.
func TestWorktreeClaimReclaimsAbandonedClaim(t *testing.T) {
	ctx := context.Background()
	clock := newManualClock(time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	store, err := New(newTestJournal(t, 1, "server"), Options{ProjectID: "github.com/fair-split/relay", ActorID: "server", AuthorityEpoch: 1, Now: clock.Now})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	params := state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-dead",
		WorktreePath: "/work/relay", Branch: "agent/run-dead", TargetRef: "main", BaseSHA: "deadbeef",
	}
	original, err := store.Worktree().Claim(ctx, params)
	if err != nil {
		t.Fatalf("original Claim: %v", err)
	}

	// Before the TTL elapses, a competitor still conflicts.
	clock.Advance(30 * time.Minute)
	competing := params
	competing.RunID = "run-successor"
	if _, err := store.Worktree().Claim(ctx, competing); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("Claim before expiry error = %v, want ErrConflict", err)
	}

	// run-dead never renews. Past the one-hour TTL, a successor reclaims.
	clock.Advance(45 * time.Minute) // total 75 minutes
	reclaimed, err := store.Worktree().Claim(ctx, competing)
	if err != nil {
		t.Fatalf("Claim after expiry: %v", err)
	}
	if reclaimed.ID == original.ID {
		t.Fatalf("reclaimed claim reused the original claim ID")
	}
	if reclaimed.RunID != "run-successor" {
		t.Fatalf("reclaimed claim run = %q, want run-successor", reclaimed.RunID)
	}

	// The abandoned claim is now visibly released, not a dangling "active"
	// record — this is what keeps List(ActiveOnly) and Get honest after a
	// takeover.
	previous, err := store.Worktree().Get(ctx, original.ID)
	if err != nil {
		t.Fatalf("Get(original.ID): %v", err)
	}
	if previous.ReleasedAt == nil {
		t.Fatalf("abandoned claim %+v was not marked released after reclaim", previous)
	}

	// The dead run's fence no longer works.
	if _, err := store.Worktree().Renew(ctx, original.ID, original.Fence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("dead run Renew error = %v, want ErrLeaseFenced", err)
	}

	// Exactly one active claim exists for this repository/branch key.
	active, err := store.Worktree().List(ctx, state.WorktreeFilter{RepositoryID: "relay", ActiveOnly: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	activeForBranch := 0
	for _, claim := range active {
		if claim.Branch == params.Branch {
			activeForBranch++
		}
	}
	if activeForBranch != 1 {
		t.Fatalf("active claims for branch %q = %d, want exactly 1 (got %+v)", params.Branch, activeForBranch, active)
	}
}

// TestWorktreeHandoffMovesOwnershipAndFencesOldToken proves explicit
// sequential handoff (agent-coordination's "Sequential cooperation is
// supported through explicit handoff"): the SAME claim ID moves to the
// incoming run under a new fence, and the outgoing run's old fence stops
// working immediately — agent-coordination#ac:one-writer-claim-is-fenced
// holds across a voluntary handoff, not only a forced reclaim.
func TestWorktreeHandoffMovesOwnershipAndFencesOldToken(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	claim, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "p", RepositoryID: "relay", RunID: "run-outgoing", Branch: "main",
	})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	if _, err := store.Worktree().Handoff(ctx, claim.ID, state.WorktreeHandoffParams{
		Fence: state.LeaseFence{Epoch: claim.Fence.Epoch, Token: "wrong"}, ToRunID: "run-incoming",
	}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Handoff with wrong fence error = %v, want ErrLeaseFenced", err)
	}

	handedOff, err := store.Worktree().Handoff(ctx, claim.ID, state.WorktreeHandoffParams{
		Fence: claim.Fence, ToRunID: "run-incoming", Reason: "outgoing run checkpointed and is handing off",
	})
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if handedOff.ID != claim.ID {
		t.Fatalf("Handoff changed the claim ID: got %q, want %q", handedOff.ID, claim.ID)
	}
	if handedOff.RunID != "run-incoming" {
		t.Fatalf("handed-off claim run = %q, want run-incoming", handedOff.RunID)
	}
	if handedOff.Fence == claim.Fence {
		t.Fatalf("Handoff reused the outgoing run's fence token")
	}

	// The outgoing run's old fence is refused...
	if _, err := store.Worktree().Renew(ctx, claim.ID, claim.Fence); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("outgoing run Renew error = %v, want ErrLeaseFenced", err)
	}
	// ...but the incoming run's new fence works, and the claim is still the
	// sole writer (no competing Claim for the same branch can succeed).
	if _, err := store.Worktree().Renew(ctx, claim.ID, handedOff.Fence); err != nil {
		t.Fatalf("incoming run Renew: %v", err)
	}
	if _, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{ProjectID: "p", RepositoryID: "relay", RunID: "run-third", Branch: "main"}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("competing Claim after handoff error = %v, want ErrConflict", err)
	}
}

// TestWorktreeHandoffOnReleasedClaimIsRefused proves a handoff cannot revive
// an already-released claim.
func TestWorktreeHandoffOnReleasedClaimIsRefused(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	claim, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{ProjectID: "p", RepositoryID: "relay", RunID: "run-a", Branch: "main"})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := store.Worktree().Release(ctx, claim.ID, claim.Fence); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := store.Worktree().Handoff(ctx, claim.ID, state.WorktreeHandoffParams{Fence: claim.Fence, ToRunID: "run-b"}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Handoff on released claim error = %v, want ErrLeaseFenced", err)
	}
}
