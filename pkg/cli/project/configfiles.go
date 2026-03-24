package project

// Features implemented: project-definition, cli/project/new, development-plan

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	SpecConfigFile     = "synchestra-spec-repo.yaml"
	StateConfigFile    = "synchestra-state-repo.yaml"
	CodeConfigFile     = "synchestra-code-repo.yaml"
	EmbeddedConfigFile = "synchestra.yaml"
	EmbeddedStateFile  = "synchestra-state.yaml"
)

// PlanningConfig holds planning-related settings from synchestra-spec-repo.yaml.
type PlanningConfig struct {
	WhatsNext string `yaml:"whats_next"`
}

type SpecConfig struct {
	Title     string          `yaml:"title"`
	StateRepo string          `yaml:"state_repo"`
	Repos     []string        `yaml:"repos"`
	Planning  *PlanningConfig `yaml:"planning,omitempty"`
}

// WhatsNextMode returns the effective whats_next mode, defaulting to "disabled".
func (c SpecConfig) WhatsNextMode() string {
	if c.Planning != nil && c.Planning.WhatsNext != "" {
		return c.Planning.WhatsNext
	}
	return "disabled"
}

type StateConfig struct {
	Title     string   `yaml:"title"`
	MainRepo  string   `yaml:"main_repo"`
	SpecRepos []string `yaml:"spec_repos"`
	CodeRepos []string `yaml:"code_repos,omitempty"`
}

type CodeConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
}

// EmbeddedConfig is the marker file written to the host repo root (on the main branch).
type EmbeddedConfig struct {
	State       string `yaml:"state"`        // "embedded"
	StateBranch string `yaml:"state_branch"` // orphan branch name, e.g. "synchestra-state"
}

// EmbeddedStateConfig lives on the orphan branch (inside the worktree).
type EmbeddedStateConfig struct {
	Title        string           `yaml:"title"`
	Mode         string           `yaml:"mode"`          // "embedded"
	SourceBranch string           `yaml:"source_branch"` // e.g. "main"
	Sync         *EmbeddedSyncCfg `yaml:"sync,omitempty"`
}

// EmbeddedSyncCfg controls sync policy for embedded state.
type EmbeddedSyncCfg struct {
	Pull string `yaml:"pull"`
	Push string `yaml:"push"`
}

func WriteSpecConfig(dir string, cfg SpecConfig) error {
	return writeYAML(filepath.Join(dir, SpecConfigFile), cfg)
}

func WriteStateConfig(dir string, cfg StateConfig) error {
	return writeYAML(filepath.Join(dir, StateConfigFile), cfg)
}

func WriteCodeConfig(dir string, cfg CodeConfig) error {
	return writeYAML(filepath.Join(dir, CodeConfigFile), cfg)
}

func ReadSpecConfig(dir string) (SpecConfig, error) {
	var cfg SpecConfig
	data, err := os.ReadFile(filepath.Join(dir, SpecConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading spec config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing spec config: %w", err)
	}
	return cfg, nil
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

func ReadCodeConfig(dir string) (CodeConfig, error) {
	var cfg CodeConfig
	data, err := os.ReadFile(filepath.Join(dir, CodeConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading code config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing code config: %w", err)
	}
	return cfg, nil
}

func WriteEmbeddedConfig(dir string, cfg EmbeddedConfig) error {
	return writeYAML(filepath.Join(dir, EmbeddedConfigFile), cfg)
}

func ReadEmbeddedConfig(dir string) (EmbeddedConfig, error) {
	var cfg EmbeddedConfig
	data, err := os.ReadFile(filepath.Join(dir, EmbeddedConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading embedded config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing embedded config: %w", err)
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
