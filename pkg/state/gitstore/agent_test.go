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

	claim, err := newStore().Agent().Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-a",
		WorktreePath: "/work/relay", Branch: "agent/run-a", TargetRef: "main",
	})
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claim.ID == "" || claim.Fence.IsZero() {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	// A competing claim on the same repository+branch is rejected...
	if _, err := newStore().Agent().Worktree().Claim(ctx, state.WorktreeClaimParams{
		ProjectID: "github.com/fair-split/relay", RepositoryID: "relay", RunID: "run-b", Branch: "agent/run-a",
	}); !errors.Is(err, state.ErrConflict) {
		t.Fatalf("competing Claim error = %v, want ErrConflict", err)
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
// version of this same proof).
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

	if _, err := newStoreAtEpoch(1).Agent().Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:relay:main", HolderRunID: "run-old"}); err != nil {
		t.Fatalf("pre-promotion Acquire at epoch 1: %v", err)
	}
	// Promotion: a store at epoch 2 writes to the same repository.
	if _, err := newStoreAtEpoch(2).Agent().Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:relay:other", HolderRunID: "run-new"}); err != nil {
		t.Fatalf("promotion write at epoch 2: %v", err)
	}
	// The former active (still epoch 1) is now fenced.
	if _, err := newStoreAtEpoch(1).Agent().Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:relay:another", HolderRunID: "run-old"}); !errors.Is(err, state.ErrLeaseFenced) {
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
