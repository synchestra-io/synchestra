package task

// Features implemented: cli/task
// Features depended on:  project-definition, state-store, state-store/backends/git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
	"gopkg.in/yaml.v3"
)

// specRepoConfig is the minimal structure of synchestra-spec-repo.yaml
// needed to resolve the state repo path.
type specRepoConfig struct {
	StateRepo string `yaml:"state_repo"`
}

// resolveStore constructs a state.Store by resolving the project and applying
// the --sync override. The commands interact only with state.Store — they are
// unaware of which backend is used.
func resolveStore(syncFlag string) (state.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, &exitError{code: 10, msg: fmt.Sprintf("getting working directory: %v", err)}
	}
	repoPath, err := resolveStateRepoPath(cwd)
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
		return nil, &exitError{code: 2, msg: fmt.Sprintf("invalid --sync value %q: must be remote or local", syncFlag)}
	}

	return gitstore.New(context.Background(), opts)
}

// mapStoreError converts state-layer errors to CLI exit codes.
func mapStoreError(err error) *exitError {
	switch {
	case errors.Is(err, state.ErrNotFound):
		return &exitError{code: 3, msg: err.Error()}
	case errors.Is(err, state.ErrConflict):
		return &exitError{code: 1, msg: err.Error()}
	case errors.Is(err, state.ErrInvalidTransition):
		return &exitError{code: 4, msg: err.Error()}
	default:
		return &exitError{code: 10, msg: err.Error()}
	}
}

// resolveStateRepoPath finds the state repo path for the current project.
// It walks up from startDir looking for synchestra-spec-repo.yaml (reads
// state_repo field) or synchestra-state-repo.yaml (direct detection).
func resolveStateRepoPath(startDir string) (string, error) {
	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		specPath := filepath.Join(current, "synchestra-spec-repo.yaml")
		if _, err := os.Stat(specPath); err == nil {
			data, err := os.ReadFile(specPath)
			if err != nil {
				return "", fmt.Errorf("reading %s: %w", specPath, err)
			}
			var cfg specRepoConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return "", fmt.Errorf("parsing %s: %w", specPath, err)
			}
			if cfg.StateRepo == "" {
				return "", &exitError{code: 3, msg: fmt.Sprintf("no state_repo field in %s", specPath)}
			}
			return cfg.StateRepo, nil
		}

		statePath := filepath.Join(current, "synchestra-state-repo.yaml")
		if _, err := os.Stat(statePath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", &exitError{code: 3, msg: "project not found: no synchestra-spec-repo.yaml or synchestra-state-repo.yaml in any parent directory"}
		}
		current = parent
	}
}
