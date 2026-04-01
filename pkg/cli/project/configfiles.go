package project

// Features implemented: project-definition, cli/project/new, development-plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/synchestra-io/specscore/pkg/projectdef"
	"gopkg.in/yaml.v3"
)

// Re-export specscore constants for convenience.
const (
	SpecConfigFile = projectdef.SpecConfigFile // "specscore-spec-repo.yaml"
	CodeConfigFile = projectdef.CodeConfigFile // "specscore-code-repo.yaml"
)

// Synchestra-owned config file constants.
const (
	StateConfigFile   = "synchestra-state-repo.yaml"
	EmbeddedStateFile = "synchestra-state.yaml"
)

// StateConfig represents the contents of synchestra-state-repo.yaml.
type StateConfig struct {
	Title     string   `yaml:"title"`
	MainRepo  string   `yaml:"main_repo"`
	SpecRepos []string `yaml:"spec_repos"`
	CodeRepos []string `yaml:"code_repos,omitempty"`
}

// EmbeddedStateConfig lives on the orphan branch (inside the worktree).
type EmbeddedStateConfig struct {
	Title        string           `yaml:"title"`
	Mode         string           `yaml:"mode"`
	SourceBranch string           `yaml:"source_branch"`
	Sync         *EmbeddedSyncCfg `yaml:"sync,omitempty"`
}

// EmbeddedSyncCfg controls sync policy for embedded state.
type EmbeddedSyncCfg struct {
	Pull string `yaml:"pull"`
	Push string `yaml:"push"`
}

func WriteStateConfig(dir string, cfg StateConfig) error {
	return writeYAML(filepath.Join(dir, StateConfigFile), cfg)
}

func ReadStateConfig(dir string) (StateConfig, error) {
	var cfg StateConfig
	data, err := os.ReadFile(filepath.Join(dir, StateConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading state config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing state config: %w", err)
	}
	return cfg, nil
}

func WriteEmbeddedStateConfig(dir string, cfg EmbeddedStateConfig) error {
	return writeYAML(filepath.Join(dir, EmbeddedStateFile), cfg)
}

func ReadEmbeddedStateConfig(dir string) (EmbeddedStateConfig, error) {
	var cfg EmbeddedStateConfig
	data, err := os.ReadFile(filepath.Join(dir, EmbeddedStateFile))
	if err != nil {
		return cfg, fmt.Errorf("reading embedded state config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing embedded state config: %w", err)
	}
	return cfg, nil
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
