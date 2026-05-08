package synchinit

// Features implemented: state-repo-config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EmbeddedStateFile is the canonical filename for the state-side
// self-identifier per the state-repo-config Feature.
const EmbeddedStateFile = "synchestra-state.yaml"

// EmbeddedStateConfig is the deserialized form of synchestra-state.yaml
// when written on the orphan-branch worktree (embedded mode). Per the
// state-repo-config Feature.
type EmbeddedStateConfig struct {
	Title        string           `yaml:"title,omitempty"`
	Mode         string           `yaml:"mode"`
	SourceBranch string           `yaml:"source_branch,omitempty"`
	Sync         *EmbeddedSyncCfg `yaml:"sync,omitempty"`
	SpecRepos    []string         `yaml:"spec_repos,omitempty"`
}

// EmbeddedSyncCfg controls sync policy for embedded state.
type EmbeddedSyncCfg struct {
	Pull string `yaml:"pull"`
	Push string `yaml:"push"`
}

// WriteEmbeddedStateConfig writes synchestra-state.yaml to dir.
func WriteEmbeddedStateConfig(dir string, cfg EmbeddedStateConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling embedded state config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, EmbeddedStateFile), data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", EmbeddedStateFile, err)
	}
	return nil
}

// ReadEmbeddedStateConfig reads dir/synchestra-state.yaml.
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
