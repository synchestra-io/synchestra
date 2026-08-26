package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  cli/auth, global-config, repo-config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
	"gopkg.in/yaml.v3"
)

const defaultAPIURL = "https://api.synchestra.io"

type hubFileConfig struct {
	ID       string `yaml:"id"`
	Endpoint string `yaml:"endpoint"`
	Token    string `yaml:"token"`
	Actor    string `yaml:"actor"`
}

type dispatchFileConfig struct {
	URL   string         `yaml:"url"`
	Token string         `yaml:"token"`
	Actor string         `yaml:"actor"`
	Hub   *hubFileConfig `yaml:"hub"`
}

type clientConfig struct {
	BaseURL string
	Token   string
	Actor   string
}

func loadClientConfig(deps Dependencies, _ string) (clientConfig, error) {
	homeDir, err := deps.UserHomeDir()
	if err != nil {
		return clientConfig{}, unexpected("resolve user home directory", err)
	}

	globalConfig, err := readGlobalDispatchConfig(deps, homeDir)
	if err != nil {
		return clientConfig{}, err
	}

	baseURL := firstNonEmpty(
		lookupEnv(deps, "SYNCHESTRA_URL"),
		hubEndpoint(globalConfig),
		globalConfig.URL,
		defaultAPIURL,
	)
	token := firstNonEmpty(
		lookupEnv(deps, "SYNCHESTRA_TOKEN"),
		hubToken(globalConfig),
		globalConfig.Token,
	)
	actor := firstNonEmpty(
		lookupEnv(deps, "SYNCHESTRA_ACTOR"),
		hubActor(globalConfig),
		globalConfig.Actor,
		"synchestra-cli",
	)

	if token == "" {
		return clientConfig{}, newCommandError(
			exitUnauthenticated,
			dispatchcontract.CodeUnauthenticated,
			"not authenticated; set SYNCHESTRA_TOKEN or run 'synchestra auth login'",
		)
	}
	return clientConfig{BaseURL: baseURL, Token: token, Actor: actor}, nil
}

func readGlobalDispatchConfig(deps Dependencies, homeDir string) (dispatchFileConfig, error) {
	if configuredPath := lookupEnv(deps, "SYNCHESTRA_CONFIG"); configuredPath != "" {
		cfg, err := readDispatchConfig(expandHome(configuredPath, homeDir), true)
		if err != nil {
			return dispatchFileConfig{}, unexpected("read SYNCHESTRA_CONFIG", err)
		}
		return cfg, nil
	}

	paths := []string{
		filepath.Join(homeDir, ".synchestra.yaml"),
		filepath.Join(homeDir, ".synchestra", "config.yaml"),
	}
	for _, path := range paths {
		cfg, err := readDispatchConfig(path, false)
		if err != nil {
			return dispatchFileConfig{}, unexpected("read global Synchestra configuration", err)
		}
		if cfg.URL != "" || cfg.Token != "" || cfg.Actor != "" || cfg.Hub != nil {
			return cfg, nil
		}
	}
	return dispatchFileConfig{}, nil
}

func readDispatchConfig(path string, required bool) (dispatchFileConfig, error) {
	var cfg dispatchFileConfig
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !required {
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

func hubEndpoint(cfg dispatchFileConfig) string {
	if cfg.Hub == nil {
		return ""
	}
	return cfg.Hub.Endpoint
}

func hubToken(cfg dispatchFileConfig) string {
	if cfg.Hub == nil {
		return ""
	}
	return cfg.Hub.Token
}

func hubActor(cfg dispatchFileConfig) string {
	if cfg.Hub == nil {
		return ""
	}
	return cfg.Hub.Actor
}

func lookupEnv(deps Dependencies, key string) string {
	value, ok := deps.LookupEnv(key)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func expandHome(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	}
	return path
}
