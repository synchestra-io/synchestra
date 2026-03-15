package internal

// Features depended on: global-config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents the user-level Synchestra configuration
// read from ~/.synchestra.yaml.
type GlobalConfig struct {
	ReposDir string `yaml:"repos_dir"`
}

// LoadGlobalConfig reads ~/.synchestra.yaml and returns the config.
// If the file doesn't exist, returns a config with default values.
// osUserHomeDir is injected for testability.
func LoadGlobalConfig(osUserHomeDir func() (string, error)) (*GlobalConfig, error) {
	homeDir, err := osUserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".synchestra.yaml")

	cfg := &GlobalConfig{
		ReposDir: filepath.Join(homeDir, "synchestra", "repos"),
	}

	// Try to read the file; if it doesn't exist, use defaults
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return default config
		}
		return nil, fmt.Errorf("could not read %s: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", configPath, err)
	}

	// Fill in defaults for empty fields
	if cfg.ReposDir == "" {
		cfg.ReposDir = filepath.Join(homeDir, "synchestra", "repos")
	} else {
		// Expand ~ to home directory
		cfg.ReposDir = expandHome(cfg.ReposDir, homeDir)
	}

	return cfg, nil
}

// expandHome replaces ~ with the home directory.
// Only handles `~` (current user home) and `~/...` patterns.
func expandHome(path string, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
