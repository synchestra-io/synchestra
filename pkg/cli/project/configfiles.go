package project

// Features implemented: project-definition, cli/project/new

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	SpecConfigFile  = "synchestra-spec-repo.yaml"
	StateConfigFile = "synchestra-state-repo.yaml"
	CodeConfigFile  = "synchestra-code-repo.yaml"
)

type SpecConfig struct {
	Title     string   `yaml:"title"`
	StateRepo string   `yaml:"state_repo"`
	Repos     []string `yaml:"repos"`
}

type StateConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
}

type CodeConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
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
