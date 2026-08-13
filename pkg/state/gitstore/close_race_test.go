package gitstore_test

// Features implemented: state-store/backends/git

import (
	"context"
	"sync"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

// TestGitStateStoreCloseRacesAgentConstruction proves GitStateStore.Close
// can be called concurrently with a GitStateStore's very first Agent() call
// without triggering a data race: Close reads agentCore/agentErr
// (gitstore.go) under the same agentMu guard Agent() (agent.go) writes them
// under, rather than reading the field directly and relying only on
// sync.Once's happens-before guarantee -- which covers goroutines that
// themselves call Do, but NOT a goroutine (Close) that reads the field
// without ever calling Do. Run with -race; without the fix this test
// reliably reports a race on the first concurrent invocation. This is the
// concurrent-caller scenario the type's own doc comment anticipates ("a
// single CLI invocation's or long-running server's actual lifecycle").
func TestGitStateStoreCloseRacesAgentConstruction(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)
	ctx := context.Background()

	s, err := gitstore.New(ctx, gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{StateRepoPath: repo},
		ProjectID:    "github.com/fair-split/relay",
	})
	if err != nil {
		t.Fatalf("gitstore.New: %v", err)
	}

	const racers = 8
	var wg sync.WaitGroup
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Agent() // first construction races Close below.
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.Close(ctx); err != nil {
			t.Errorf("concurrent Close: %v", err)
		}
	}()
	wg.Wait()

	// The store is left in a coherent, still-usable state: Agent() succeeds
	// and a write goes through, whether or not this particular Close raced
	// ahead of construction finishing.
	if _, err := s.Agent().Lease().Acquire(ctx, state.LeaseAcquireParams{Resource: "worktree:relay:main", HolderRunID: "run-a"}); err != nil {
		t.Fatalf("Acquire after the concurrent Agent()/Close race: %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("final Close: %v", err)
	}
}
