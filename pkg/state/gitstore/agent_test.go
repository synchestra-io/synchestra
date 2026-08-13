package gitstore_test

// Features implemented: state-store/topology
// Features depended on:  state-store, agent-coordination

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// TestGitStateStoreAgentClaimsAWorktreeThroughRealGit proves the "wire the
// contracts into the Git store path" half of Task 1: GitStateStore.Agent()
// is not a stub — it persists through an actual inGitDB-backed Git
// repository, and a claim survives being read back through a second,
// independently-constructed GitStateStore pointed at the same repo path
// (the way a restarted CLI process would see it).
//
// This test deliberately does NOT use state.CloseAfter around Claim: Claim
// issues two SEQUENTIAL Appends internally (a lease acquire, then a
// worktree.claimed/reclaimed follow-up — see worktree.go and
// pkg/state/agentstore/README.md's Open Questions), and the second Append
// only starts once the first has already returned. Racing a single Close
// call against the whole two-Append sequence is unsafe: Close may flush and
// close the journal after the first Append lands but before the second one
// even enqueues, permanently failing it with ErrJournalClosed instead of
// speeding it up. CloseAfter's doc comment states this constraint;
// pkg/cli/agent's command wiring honors it by only using CloseAfter around
// genuinely single-Append operations (message send/ack) and calling
// Worktree() mutations plainly. See TestGitStateStoreAgentRespectsPromotion
// FenceThroughRealGit below for the real-Git, single-Append exit-flush
// proof, and pkg/state/agentstore/close_test.go for the journal-level one.
func TestGitStateStoreAgentClaimsAWorktreeThroughRealGit(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)
	ctx := context.Background()

	newStore := func() state.Store {
		s, err := gitstore.New(ctx, gitstore.GitStoreOptions{
			StoreOptions: state.StoreOptions{StateRepoPath: repo},
			ProjectID:    "github.com/fair-split/relay",
		})
		if err != nil {
			t.Fatalf("gitstore.New: %v", err)
		}
		return s
	}

	claimStore := newStore()
	claim, err := claimStore.Agent().Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-a",
		WorktreePath: "/work/relay", Branch: "agent/run-a", TargetRef: "main",
	})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if err := claimStore.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if claim.ID == "" || claim.Fence.IsZero() {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	// A competing claim on the same repository+branch is rejected...
	competingStore := newStore()
	_, err = competingStore.Agent().Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-b", Branch: "agent/run-a",
	})
	if !errors.Is(err, state.ErrConflict) {
		t.Fatalf("competing Claim error = %v, want ErrConflict", err)
	}
	if err := competingStore.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// ...and a fresh GitStateStore over the same on-disk repo (simulating a
	// restarted CLI process) reads back the same claim.
	got, err := newStore().Agent().Worktree().Get(ctx, claim.ID)
	if err != nil {
		t.Fatalf("Get from a fresh store instance: %v", err)
	}
	if got.RunID != "run-a" || got.Branch != "agent/run-a" {
		t.Fatalf("unexpected reloaded claim: %+v", got)
	}

	if out, err := exec.Command("git", "-C", repo, "log", "--oneline").CombinedOutput(); err != nil || len(out) == 0 {
		t.Fatalf("expected real Git commits in %s: err=%v out=%s", repo, err, out)
	}
}

// TestGitStateStoreAgentRespectsPromotionFenceThroughRealGit is the Git-
// store-path proof for state-store/topology#ac:promotion-fences-former-active's
// claim/lease half: once a store configured for a newer authority epoch has
// written to the shared repository, a store still configured for the old
// epoch is fenced on its very next write — through the real inGitDB-backed
// journal, not just the in-memory conformance harness (see
// pkg/state/agentstore/lease_test.go's
// TestPromotionFencesFormerActiveStoreOnNextWrite for the backend-neutral
// version of this same proof). This test is about fencing CORRECTNESS, not
// exit-flush latency, so it deliberately does not use state.CloseAfter: the
// fenced outcome here is only decided once the journal's actual commit
// validation runs (the local role/epoch precondition alone does not catch
// it — see CloseAfter's doc comment on the "at most one Append" contract),
// and racing a forced Close against that under real Git I/O (slower, and
// variable, under -race in particular) risks observing ErrJournalClosed
// instead of the fencing outcome this test asserts. See
// TestGitStateStoreAgentClaimsAWorktreeThroughRealGit's doc comment for the
// full reasoning and pkg/state/agentstore/close_test.go for the
// deterministic (in-memory journal) exit-flush proof.
func TestGitStateStoreAgentRespectsPromotionFenceThroughRealGit(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)
	ctx := context.Background()

	newStoreAtEpoch := func(epoch int64) state.Store {
		s, err := gitstore.New(ctx, gitstore.GitStoreOptions{
			StoreOptions:   state.StoreOptions{StateRepoPath: repo},
			ProjectID:      "github.com/fair-split/relay",
			AuthorityEpoch: epoch,
		})
		if err != nil {
			t.Fatalf("gitstore.New(epoch=%d): %v", epoch, err)
		}
		return s
	}
	acquire := func(store state.Store, resource, holder string) (state.AuthorityLease, error) {
		lease, err := store.Agent().Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: resource, HolderRunID: holder})
		if closeErr := store.Close(ctx); closeErr != nil {
			t.Fatalf("Close: %v", closeErr)
		}
		return lease, err
	}

	if _, err := acquire(newStoreAtEpoch(1), "worktree:relay:main", "run-old"); err != nil {
		t.Fatalf("pre-promotion Acquire at epoch 1: %v", err)
	}
	// Promotion: a store at epoch 2 writes to the same repository.
	if _, err := acquire(newStoreAtEpoch(2), "worktree:relay:other", "run-new"); err != nil {
		t.Fatalf("promotion write at epoch 2: %v", err)
	}
	// The former active (still epoch 1) is now fenced.
	if _, err := acquire(newStoreAtEpoch(1), "worktree:relay:another", "run-old"); !errors.Is(err, state.ErrLeaseFenced) {
		t.Fatalf("post-promotion Acquire at old epoch error = %v, want ErrLeaseFenced", err)
	}
}

// TestGitStateStoreAgentUnavailableWithoutProjectID proves Agent() is
// fail-closed rather than panicking or silently succeeding when the caller
// forgot to configure ProjectID.
func TestGitStateStoreAgentUnavailableWithoutProjectID(t *testing.T) {
	ctx := context.Background()
	s, err := gitstore.New(ctx, gitstore.GitStoreOptions{StoreOptions: state.StoreOptions{StateRepoPath: t.TempDir()}})
	if err != nil {
		t.Fatalf("gitstore.New: %v", err)
	}
	if _, err := s.Agent().Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "r", Title: "t"}); err == nil {
		t.Fatal("Effort().Create without ProjectID unexpectedly succeeded")
	}
	if _, err := s.Agent().Health().Report(ctx); err == nil {
		t.Fatal("Health().Report without ProjectID unexpectedly succeeded")
	}
}
