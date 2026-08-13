package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

func TestWorktreeClaimIsUniquePerRepositoryAndBranch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	params := state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-a",
		WorktreePath: "/work/relay", Branch: "agent/run-a", TargetRef: "main", BaseSHA: "deadbeef",
	}
	claim, err := store.Worktree().Claim(ctx, params)
	if err != nil {
		t.Fatalf("first Claim: %v", err)
	}
	if claim.ID == "" || claim.Fence.IsZero() {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	competing := params
	competing.RunID = "run-b"
	competing.WorktreePath = "/work/relay-2"
	if _, err := store.Worktree().Claim(ctx, competing); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("competing Claim on same repo/branch error = %v, want ErrConflict", err)
	}

	// A different branch on the same repository is independent.
	other := params
	other.RunID = "run-c"
	other.Branch = "agent/run-c"
	if _, err := store.Worktree().Claim(ctx, other); err != nil {
		t.Fatalf("Claim on a different branch: %v", err)
	}

	got, err := store.Worktree().Get(ctx, claim.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RunID != "run-a" || got.Branch != "agent/run-a" {
		t.Fatalf("Get returned wrong claim: %+v", got)
	}

	all, err := store.Worktree().List(ctx, state.WorktreeFilter{RepositoryID: "relay", ActiveOnly: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("active claims = %d, want 2 (got %+v)", len(all), all)
	}
}

func TestWorktreeClaimUnderConcurrencyHasExactlyOneWinner(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	const racers = 10
	var wg sync.WaitGroup
	successes := 0
	conflicts := 0
	var mu sync.Mutex
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{
				ProjectID: "p", RepositoryID: "relay", RunID: "run", Branch: "main",
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, state.ErrConflict):
				conflicts++
			default:
				t.Errorf("unexpected Claim error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("successful claims = %d, want 1", successes)
	}
	if conflicts != racers-1 {
		t.Fatalf("conflicting claims = %d, want %d", conflicts, racers-1)
	}
}

func TestWorktreeRenewAndReleaseRespectFence(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	claim, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "p", RepositoryID: "relay", RunID: "run-a", Branch: "main",
	})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	if _, err := store.Worktree().Renew(ctx, claim.ID, state.LeaseFence{Epoch: claim.Fence.Epoch, Token: "wrong"}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Renew with wrong fence error = %v, want ErrLeaseFenced", err)
	}
	renewed, err := store.Worktree().Renew(ctx, claim.ID, claim.Fence)
	if err != nil {
		t.Fatalf("Renew: %v", err)
	}
	if !renewed.RenewedAt.After(claim.RenewedAt) {
		t.Fatalf("renewed.RenewedAt did not advance: got %v, want after %v", renewed.RenewedAt, claim.RenewedAt)
	}

	if err := store.Worktree().Release(ctx, claim.ID, state.LeaseFence{Epoch: claim.Fence.Epoch, Token: "wrong"}); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("Release with wrong fence error = %v, want ErrLeaseFenced", err)
	}
	if err := store.Worktree().Release(ctx, claim.ID, claim.Fence); err != nil {
		t.Fatalf("Release: %v", err)
	}
	released, err := store.Worktree().Get(ctx, claim.ID)
	if err != nil {
		t.Fatalf("Get after release: %v", err)
	}
	if released.ReleasedAt == nil {
		t.Fatalf("released claim has no ReleasedAt: %+v", released)
	}

	// The branch is claimable again after release.
	if _, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{ProjectID: "p", RepositoryID: "relay", RunID: "run-b", Branch: "main"}); err != nil {
		t.Fatalf("Claim after release: %v", err)
	}
}

func TestWorktreeClaimRejectsIncompleteParams(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Worktree().Claim(ctx, state.WorktreeClaimParams{ProjectID: "p"}); err == nil {
		t.Fatal("Claim with missing fields unexpectedly succeeded")
	}
}

func TestWorktreeGetMissingClaimReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Worktree().Get(ctx, "does-not-exist"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
}
