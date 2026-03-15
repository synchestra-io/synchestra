package internal

// Features depended on: cli/project/new, repository-types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SynchstraSpecYaml represents synchestra-spec.yaml.
type SynchstraSpecYaml struct {
	Title     string   `yaml:"title"`
	StateRepo string   `yaml:"state_repo"`
	Repos     []string `yaml:"repos"`
}

// SynchstraStateYaml represents synchestra-state.yaml.
type SynchstraStateYaml struct {
	SpecRepo string `yaml:"spec_repo"`
}

// SynchstraTargetYaml represents synchestra-target.yaml.
type SynchstraTargetYaml struct {
	SpecRepo string `yaml:"spec_repo"`
}

// ReadSpecConfig reads synchestra-spec.yaml from the given repo path.
// Returns nil if the file doesn't exist.
func ReadSpecConfig(repoPath string) (*SynchstraSpecYaml, error) {
	filePath := filepath.Join(repoPath, "synchestra-spec.yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read synchestra-spec.yaml: %w", err)
	}

	var cfg SynchstraSpecYaml
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse synchestra-spec.yaml: %w", err)
	}
	return &cfg, nil
}

// ReadStateConfig reads synchestra-state.yaml from the given repo path.
// Returns nil if the file doesn't exist.
func ReadStateConfig(repoPath string) (*SynchstraStateYaml, error) {
	filePath := filepath.Join(repoPath, "synchestra-state.yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read synchestra-state.yaml: %w", err)
	}

	var cfg SynchstraStateYaml
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse synchestra-state.yaml: %w", err)
	}
	return &cfg, nil
}

// ReadTargetConfig reads synchestra-target.yaml from the given repo path.
// Returns nil if the file doesn't exist.
func ReadTargetConfig(repoPath string) (*SynchstraTargetYaml, error) {
	filePath := filepath.Join(repoPath, "synchestra-target.yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read synchestra-target.yaml: %w", err)
	}

	var cfg SynchstraTargetYaml
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse synchestra-target.yaml: %w", err)
	}
	return &cfg, nil
}

// WriteSpecConfig writes synchestra-spec.yaml to the given repo path.
func WriteSpecConfig(repoPath string, cfg *SynchstraSpecYaml) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal synchestra-spec.yaml: %w", err)
	}

	filePath := filepath.Join(repoPath, "synchestra-spec.yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("could not write synchestra-spec.yaml: %w", err)
	}
	return nil
}

// WriteStateConfig writes synchestra-state.yaml to the given repo path.
func WriteStateConfig(repoPath string, cfg *SynchstraStateYaml) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal synchestra-state.yaml: %w", err)
	}

	filePath := filepath.Join(repoPath, "synchestra-state.yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("could not write synchestra-state.yaml: %w", err)
	}
	return nil
}

// WriteTargetConfig writes synchestra-target.yaml to the given repo path.
func WriteTargetConfig(repoPath string, cfg *SynchstraTargetYaml) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal synchestra-target.yaml: %w", err)
	}

	filePath := filepath.Join(repoPath, "synchestra-target.yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("could not write synchestra-target.yaml: %w", err)
	}
	return nil
}

// DeriveTitle derives the project title from the spec repo README or repo identifier.
func DeriveTitle(specRepoPath string, specRepoRef *RepoRef, providedTitle string) string {
	if providedTitle != "" {
		return providedTitle
	}

	// Try to read README.md and find first H1 (# ) heading
	readmePath := filepath.Join(specRepoPath, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			// Only match H1: exactly "# " followed by text
			if strings.HasPrefix(trimmed, "# ") {
				heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
				if heading != "" {
					return heading
				}
			}
		}
	}

	// Fall back to repo identifier
	return specRepoRef.Repo
}
