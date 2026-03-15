// Package globalconfig loads ~/.synchestra.yaml.
package globalconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig is the user-level Synchestra configuration from ~/.synchestra.yaml.
type GlobalConfig struct {
	ReposDir string `yaml:"repos_dir"`
}

// Load reads ~/.synchestra.yaml from homeDir and returns the config.
// Missing file returns defaults; invalid YAML returns an error.
func Load(homeDir string) (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	data, err := os.ReadFile(filepath.Join(homeDir, ".synchestra.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		cfg.ReposDir = filepath.Join(homeDir, "synchestra", "repos")
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.ReposDir = expandTilde(cfg.ReposDir, homeDir)
	if cfg.ReposDir == "" {
		cfg.ReposDir = filepath.Join(homeDir, "synchestra", "repos")
	}
	return cfg, nil
}

func expandTilde(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
