package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	SpecConfigFile   = "synchestra-spec.yaml"
	StateConfigFile  = "synchestra-state.yaml"
	TargetConfigFile = "synchestra-target.yaml"
)

type SpecConfig struct {
	Title     string   `yaml:"title"`
	StateRepo string   `yaml:"state_repo"`
	Repos     []string `yaml:"repos"`
}

type StateConfig struct {
	SpecRepo string `yaml:"spec_repo"`
}

type TargetConfig struct {
	SpecRepo string `yaml:"spec_repo"`
}

func WriteSpecConfig(dir string, cfg SpecConfig) error {
	return writeYAML(filepath.Join(dir, SpecConfigFile), cfg)
}

func WriteStateConfig(dir string, cfg StateConfig) error {
	return writeYAML(filepath.Join(dir, StateConfigFile), cfg)
}

func WriteTargetConfig(dir string, cfg TargetConfig) error {
	return writeYAML(filepath.Join(dir, TargetConfigFile), cfg)
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

func ReadTargetConfig(dir string) (TargetConfig, error) {
	var cfg TargetConfig
	data, err := os.ReadFile(filepath.Join(dir, TargetConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading target config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing target config: %w", err)
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
