package task

// Features implemented: cli/task
// Features depended on:  project-definition, state-store, state-store/backends/git

import (
	"context"
	"errors"
	"os"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/resolve"
	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

// resolveStore constructs a state.Store by resolving the project and applying
// the --sync override. The commands interact only with state.Store — they are
// unaware of which backend is used.
func resolveStore(syncFlag string) (state.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, exitcode.UnexpectedErrorf("getting working directory: %v", err)
	}
	repoPath, err := resolve.StateRepoPath(cwd)
	if err != nil {
		return nil, err
	}

	opts := gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{
			StateRepoPath: repoPath,
		},
	}

	switch syncFlag {
	case "remote":
		opts.Sync.Pull = state.SyncOnCommit
		opts.Sync.Push = state.SyncOnCommit
	case "local":
		opts.Sync.Pull = state.SyncManual
		opts.Sync.Push = state.SyncManual
	case "":
		// use defaults (on_commit)
	default:
		return nil, exitcode.InvalidArgsErrorf("invalid --sync value %q: must be remote or local", syncFlag)
	}

	return gitstore.New(context.Background(), opts)
}

// mapStoreError converts state-layer errors to CLI exit codes.
func mapStoreError(err error) *exitcode.Error {
	switch {
	case errors.Is(err, state.ErrNotFound):
		return exitcode.NotFoundError(err.Error())
	case errors.Is(err, state.ErrConflict):
		return exitcode.ConflictError(err.Error())
	case errors.Is(err, state.ErrInvalidTransition):
		return exitcode.InvalidStateError(err.Error())
	default:
		return exitcode.UnexpectedError(err.Error())
	}
}

