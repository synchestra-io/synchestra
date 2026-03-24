package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"

	"github.com/synchestra-io/synchestra/pkg/state"
)

type gitStateSync struct {
	store *GitStateStore
}

func (s *gitStateSync) Pull(_ context.Context) error { return errNotImplemented }
func (s *gitStateSync) Push(_ context.Context) error { return errNotImplemented }
func (s *gitStateSync) Sync(_ context.Context) error { return errNotImplemented }

// Compile-time interface check.
var _ state.StateSync = (*gitStateSync)(nil)
