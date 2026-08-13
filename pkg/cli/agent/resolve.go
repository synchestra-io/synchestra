package agent

// Features implemented: cli
// Features depended on:  state-store, state-store/backends/git, agent-coordination, global-config

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/reporef"
	"github.com/synchestra-io/synchestra/pkg/cli/resolve"
	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

// resolveStore constructs a state.Store the same way pkg/cli/task's
// resolveStore does (state repo path + --sync override), and additionally
// resolves the project identity Agent() requires (unlike Task()/Chat(),
// GitStoreOptions.ProjectID is mandatory for Agent() — see
// pkg/state/gitstore/agent.go), returning it alongside the store since
// every agent command needs to stamp it onto its domain params too.
// Commands interact only with state.Store — they are unaware of which
// backend is used.
func resolveStore(cmd *cobra.Command, projectFlag, syncFlag, runFlag string) (state.Store, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", exitcode.UnexpectedErrorf("getting working directory: %v", err)
	}
	repoPath, err := resolve.StateRepoPath(cwd)
	if err != nil {
		return nil, "", err
	}
	projectID, err := resolveProjectID(cwd, projectFlag)
	if err != nil {
		return nil, "", err
	}

	opts := gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{StateRepoPath: repoPath},
		ProjectID:    projectID,
		RunID:        runFlag,
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
		return nil, "", exitcode.InvalidArgsErrorf("invalid --sync value %q: must be remote or local", syncFlag)
	}

	store, err := gitstore.New(cmd.Context(), opts)
	if err != nil {
		return nil, "", err
	}
	return store, projectID, nil
}

// resolveProjectID returns explicit, canonicalized via reporef.Parse, if
// set; otherwise it best-effort derives the canonical hosting/org/repo
// identity from the current directory's "origin" Git remote, matching the
// same "hosting/org/repo" convention pkg/cli/reporef and this package's
// tests use throughout. Every agent-coordination record is scoped to a
// ProjectID (state-store/topology), so a command that cannot resolve one
// fails fast with a clear, actionable message rather than defaulting to an
// empty or ambiguous value.
func resolveProjectID(cwd, explicit string) (string, error) {
	if s := strings.TrimSpace(explicit); s != "" {
		ref, err := reporef.Parse(s)
		if err != nil {
			return "", exitcode.InvalidArgsErrorf("--project: %v", err)
		}
		return ref.Hosting + "/" + ref.Org + "/" + ref.Repo, nil
	}
	out, err := exec.Command("git", "-C", cwd, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", exitcode.InvalidArgsError("--project is required (no origin Git remote to detect it from)")
	}
	ref, err := reporef.Parse(strings.TrimSpace(string(out)))
	if err != nil {
		return "", exitcode.InvalidArgsErrorf("--project is required: could not parse origin remote %q: %v", strings.TrimSpace(string(out)), err)
	}
	return ref.Hosting + "/" + ref.Org + "/" + ref.Repo, nil
}

// closeStore calls store.Close and maps any failure to the CLI's
// unexpected-error exit code, so every command reports a Close failure
// consistently (state-store/journal-batching#ac:close-flushes-pending-batch:
// a caller must call Close before exit — this is the one place every agent
// command does so).
func closeStore(ctx context.Context, store state.Store) error {
	if err := store.Close(ctx); err != nil {
		return exitcode.UnexpectedErrorf("closing store: %v", err)
	}
	return nil
}

// mapStoreError converts state-layer errors to CLI exit codes, matching
// pkg/cli/task/resolve.go's mapStoreError exactly (kept as a separate copy
// per-package, the same way pkg/cli/task already does, rather than
// introducing a shared cross-command-group dependency for four lines).
func mapStoreError(err error) *exitcode.Error {
	switch {
	case errors.Is(err, state.ErrNotFound):
		return exitcode.NotFoundError(err.Error())
	case errors.Is(err, state.ErrConflict):
		return exitcode.ConflictError(err.Error())
	case errors.Is(err, state.ErrLeaseFenced):
		return exitcode.ConflictError(err.Error())
	case errors.Is(err, state.ErrInvalidTransition):
		return exitcode.InvalidStateError(err.Error())
	default:
		return exitcode.UnexpectedError(err.Error())
	}
}
