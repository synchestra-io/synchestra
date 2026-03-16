package gitstore_test

// Features depended on: state-store, state-store/backends/git

import (
	"context"
	"testing"

	"github.com/synchesta-io/synchestra/pkg/state"
	"github.com/synchesta-io/synchestra/pkg/state/gitstore"
)

// TestGitStateStoreImplementsStore verifies that GitStateStore satisfies state.Store
// and that New can be used as a state.StoreFactory.
func TestGitStateStoreImplementsStore(t *testing.T) {
	// Verify New satisfies StoreFactory signature
	var factory state.StoreFactory = gitstore.New

	store, err := factory(context.Background(), state.StoreOptions{
		StateRepoPath: t.TempDir(),
		SpecRepoPath:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	// Verify sub-interface accessors return non-nil
	if store.Task() == nil {
		t.Error("Task() returned nil")
	}
	if store.Chat() == nil {
		t.Error("Chat() returned nil")
	}
	if store.Project() == nil {
		t.Error("Project() returned nil")
	}

	// Verify nested accessors return non-nil
	if store.Task().Board() == nil {
		t.Error("Task().Board() returned nil")
	}
	if store.Task().Artifact(context.Background(), "test-task") == nil {
		t.Error("Task().Artifact() returned nil")
	}
	if store.Chat().Artifact(context.Background(), "test-chat") == nil {
		t.Error("Chat().Artifact() returned nil")
	}
}
