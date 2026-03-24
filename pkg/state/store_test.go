package state_test

// Features depended on: state-store

import (
	"context"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// Compile-time check that Store requires State() accessor.
type mockStore struct{}

func (m *mockStore) Task() state.TaskStore       { return nil }
func (m *mockStore) Chat() state.ChatStore       { return nil }
func (m *mockStore) Project() state.ProjectStore { return nil }
func (m *mockStore) State() state.StateSync      { return nil }

var _ state.Store = (*mockStore)(nil)

func TestStateSyncInterface(t *testing.T) {
	// Verify StateSync interface has all three methods via compile-time check.
	var _ state.StateSync = (*mockStateSync)(nil)
}

type mockStateSync struct{}

func (m *mockStateSync) Pull(_ context.Context) error { return nil }
func (m *mockStateSync) Push(_ context.Context) error { return nil }
func (m *mockStateSync) Sync(_ context.Context) error { return nil }
