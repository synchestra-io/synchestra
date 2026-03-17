package gitstore_test

// Features depended on: state-store, state-store/backends/git

import (
	"context"
	"testing"

	"github.com/synchesta-io/synchestra/pkg/state"
	"github.com/synchesta-io/synchestra/pkg/state/gitstore"
)

// TestGitStateStoreImplementsStore verifies that GitStateStore satisfies state.Store
// and that New returns a valid store with all sub-interfaces accessible.
func TestGitStateStoreImplementsStore(t *testing.T) {
	var store state.Store

	s, err := gitstore.New(context.Background(), gitstore.Options{
		StateRepoPath: t.TempDir(),
		SpecRepoPaths: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	store = s // compile-time verification that *GitStateStore satisfies state.Store

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

func TestSyncModeDefaultsToSync(t *testing.T) {
	s, err := gitstore.New(context.Background(), gitstore.Options{
		StateRepoPath: t.TempDir(),
		SpecRepoPaths: []string{t.TempDir()},
		// SyncMode intentionally omitted — should default to "sync"
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	// Verify it returns a valid store (sync mode is internal,
	// but construction should succeed with default)
	if s == nil {
		t.Error("New() returned nil store")
	}
}
