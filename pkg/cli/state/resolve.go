package state

// Features implemented: cli/state
// Features depended on:  project-definition

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// specRepoConfig is the minimal structure of synchestra-spec-repo.yaml
// needed to resolve the state repo path.
type specRepoConfig struct {
	StateRepo string `yaml:"state_repo"`
}

// TODO: Remove once pull/push/sync commands call resolveStateRepoPath.
var _ = resolveStateRepoPath

// resolveStateRepoPath finds the state repo path for the current project.
// It walks up from startDir looking for synchestra-spec-repo.yaml (reads
// state_repo field) or synchestra-state-repo.yaml (direct detection).
//
// TODO: Add stateRepoConfig struct for parsing synchestra-state-repo.yaml
// when direct state repo detection needs config fields beyond file existence.
func resolveStateRepoPath(startDir string) (string, error) {
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
				return "", &exitError{code: 3, msg: fmt.Sprintf("no state_repo field in %s", specPath)}
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
			return "", &exitError{code: 3, msg: "project not found: no synchestra-spec-repo.yaml or synchestra-state-repo.yaml in any parent directory"}
		}
		current = parent
	}
}
