package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"gopkg.in/yaml.v3"
)

// specRepoConfig is the minimal structure of synchestra-spec-repo.yaml
// needed to resolve the state repo path.
type specRepoConfig struct {
	StateRepo string `yaml:"state_repo"`
}

// StateRepoPath finds the state repo path for the current project.
// It walks up from startDir looking for:
//   - synchestra-spec-repo.yaml (reads state_repo field; worktree:// for embedded state)
//   - synchestra-state-repo.yaml (direct detection)
func StateRepoPath(startDir string) (string, error) {
	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		// Check for spec repo config (spec repo -> state repo via state_repo field)
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
				return "", exitcode.NotFoundErrorf("no state_repo field in %s", specPath)
			}

			// Check for worktree:// scheme (embedded state).
			if strings.HasPrefix(cfg.StateRepo, "worktree://") {
				worktreePath := filepath.Join(current, ".synchestra")
				if info, statErr := os.Stat(worktreePath); statErr == nil && info.IsDir() {
					return worktreePath, nil
				}
				return "", exitcode.NotFoundErrorf("embedded state configured in %s but .synchestra/ worktree is missing; run 'synchestra project init' to set up", specPath)
			}

			// TODO: Resolve state_repo URL to local path using repos_dir convention
			return cfg.StateRepo, nil
		}

		// Check for state repo config (direct detection)
		statePath := filepath.Join(current, "synchestra-state-repo.yaml")
		if _, err := os.Stat(statePath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Config-less fallback: look for .synchestra/ at the git repo root.
			return configLessFallback(startDir)
		}
		current = parent
	}
}

// configLessFallback implements config-less mode resolution.
// It finds the git repository root and checks for a .synchestra/ directory.
// Features implemented: embedded-state
func configLessFallback(startDir string) (string, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	gitRoot, err := findGitRoot(absStart)
	if err != nil {
		return "", exitcode.NotFoundError("project not found: no synchestra-spec-repo.yaml or synchestra-state-repo.yaml in any parent directory")
	}

	worktreePath := filepath.Join(gitRoot, ".synchestra")
	if info, statErr := os.Stat(worktreePath); statErr == nil && info.IsDir() {
		return worktreePath, nil
	}

	return "", exitcode.NotFoundErrorf("no Synchestra project found in git repo %s; run 'synchestra project init' to set up", gitRoot)
}

// findGitRoot walks up from dir looking for a .git directory or file.
// It returns the directory containing .git, or an error if none is found.
func findGitRoot(dir string) (string, error) {
	current := dir
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a git repository: no .git found above %s", dir)
		}
		current = parent
	}
}
