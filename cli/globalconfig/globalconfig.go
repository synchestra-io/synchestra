package globalconfig

import (
	"errors"
	"io/fs"
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

// Load reads the global config from the given path.
// Returns a zero-value config (no error) if the file does not exist.
func Load(path string) (GlobalConfig, error) {
	var cfg GlobalConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// ResolveReposDir returns the effective repos directory.
// If reposDir is empty, returns {homeDir}/synchestra/repos.
// Expands ~ prefix and resolves relative paths against homeDir.
func ResolveReposDir(reposDir, homeDir string) string {
	if reposDir == "" {
		return filepath.Join(homeDir, "synchestra", "repos")
	}
	if strings.HasPrefix(reposDir, "~/") {
		return filepath.Join(homeDir, reposDir[2:])
	}
	if reposDir == "~" {
		return homeDir
	}
	if filepath.IsAbs(reposDir) {
		return reposDir
	}
	return filepath.Join(homeDir, reposDir)
}
