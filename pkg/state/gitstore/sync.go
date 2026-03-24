package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"

	"github.com/synchestra-io/synchestra/pkg/cli/gitops"
	"github.com/synchestra-io/synchestra/pkg/state"
)

type gitStateSync struct {
	store *GitStateStore
}

func (s *gitStateSync) Pull(_ context.Context) error {
	return gitops.Pull(s.store.stateRepoPath)
}

func (s *gitStateSync) Push(_ context.Context) error {
	return gitops.Push(s.store.stateRepoPath)
}

func (s *gitStateSync) Sync(ctx context.Context) error {
	if err := s.Pull(ctx); err != nil {
		return err
	}
	return s.Push(ctx)
}

// Compile-time interface check.
var _ state.StateSync = (*gitStateSync)(nil)
