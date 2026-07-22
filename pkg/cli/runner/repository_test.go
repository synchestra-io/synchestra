package runner

// Features implemented: cli/runner/dispatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestResolveRepositoryIncludesProjectSubdirectoryAndHubID(t *testing.T) {
	repoPath := newTestRepository(t, map[string]string{
		"README.md":                    "# Monorepo\n",
		"services/api/synchestra.yaml": "# Synchestra Repo Config Schema: https://synchestra.md/repo-config\n\nhub:\n  id: acme/api\n  endpoint: https://project.example.test\n",
	})
	projectPath := filepath.Join(repoPath, "services", "api")
	repository, err := resolveRepository(context.Background(), projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if repository.Root != repoPath || repository.ProjectRoot != projectPath || repository.Subdirectory != "services/api" {
		t.Fatalf("repository context = %+v", repository)
	}
	if repository.Snapshot.ProjectID != "acme/api" || repository.Snapshot.Subdirectory != "services/api" {
		t.Fatalf("snapshot = %+v", repository.Snapshot)
	}
}

func TestResolveRepositoryPreservesSCPStyleSSHCloneURL(t *testing.T) {
	repoPath := newTestRepository(t, map[string]string{"README.md": "# Repository\n"})
	runTestCommand(t, repoPath, "git", "remote", "set-url", "origin", "git@code.example.test:platform/service.git")

	repository, err := resolveRepository(context.Background(), repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if repository.Snapshot.CanonicalID != "code.example.test/platform/service" {
		t.Fatalf("canonical ID = %q", repository.Snapshot.CanonicalID)
	}
	if repository.Snapshot.CloneURL != "git@code.example.test:platform/service.git" {
		t.Fatalf("clone URL = %q", repository.Snapshot.CloneURL)
	}
}

func TestNormalizeRemoteProducesCredentialFreeCloneURL(t *testing.T) {
	tests := []struct {
		input         string
		wantCanonical string
		wantClone     string
	}{
		{input: "git@github.com:acme/repo.git", wantCanonical: "github.com/acme/repo", wantClone: "git@github.com:acme/repo.git"},
		{input: "ssh://git@gitlab.example.com/group/sub/repo.git", wantCanonical: "gitlab.example.com/group/sub/repo", wantClone: "ssh://git@gitlab.example.com/group/sub/repo.git"},
		{input: "ssh://deploy@gitlab.example.com:2222/group/sub/repo", wantCanonical: "gitlab.example.com:2222/group/sub/repo", wantClone: "ssh://deploy@gitlab.example.com:2222/group/sub/repo.git"},
		{input: "https://github.com/acme/repo.git", wantCanonical: "github.com/acme/repo", wantClone: "https://github.com/acme/repo.git"},
		{input: "http://localhost:8080/acme/repo", wantCanonical: "localhost:8080/acme/repo", wantClone: "http://localhost:8080/acme/repo.git"},
		{input: "git://localhost:9418/acme/repo", wantCanonical: "localhost:9418/acme/repo", wantClone: "git://localhost:9418/acme/repo.git"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			canonical, cloneURL, err := normalizeRemote(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if canonical != test.wantCanonical || cloneURL != test.wantClone {
				t.Fatalf("normalizeRemote(%q) = %q, %q", test.input, canonical, cloneURL)
			}
		})
	}
}

func TestNormalizeRemoteRejectsInlineCredentials(t *testing.T) {
	tests := []string{
		"https://token@github.com/acme/repo.git",
		"https://user:password@github.com/acme/repo.git",
		"ssh://git:password@github.com/acme/repo.git",
		"git:password@github.com:acme/repo.git",
	}
	for _, remote := range tests {
		t.Run(remote, func(t *testing.T) {
			if _, _, err := normalizeRemote(remote); err == nil {
				t.Fatal("credential-bearing Git origin accepted")
			}
		})
	}
}

func TestNormalizedSSHRemotePassesCanonicalContractWithoutPassword(t *testing.T) {
	canonical, cloneURL, err := normalizeRemote("ssh://git@git.example.test:2222/group/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	repository := dispatchcontract.RepositorySnapshot{
		CanonicalID:  canonical,
		CloneURL:     cloneURL,
		BaseRevision: "1111111111111111111111111111111111111111",
	}
	if err := repository.Validate(); err != nil {
		t.Fatalf("credential-free SSH snapshot rejected: %v", err)
	}
	repository.CloneURL = "ssh://git:password@git.example.test:2222/group/repo.git"
	if err := repository.Validate(); err == nil {
		t.Fatal("contract accepted an inline SSH password")
	}
}

func TestNormalizeRemotePreservesSCPTransportWithoutCredentialsInIdentity(t *testing.T) {
	canonical, cloneURL, err := normalizeRemote("git@code.example.test:platform/service.git")
	if err != nil {
		t.Fatal(err)
	}
	if canonical != "code.example.test/platform/service" {
		t.Fatalf("canonical ID = %q", canonical)
	}
	if cloneURL != "git@code.example.test:platform/service.git" {
		t.Fatalf("clone URL = %q", cloneURL)
	}
	if canonical == cloneURL || canonical == "git@code.example.test/platform/service" {
		t.Fatalf("canonical ID contains transport identity: %q", canonical)
	}
}

func TestClientConfigPrecedence(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	global := "hub:\n  endpoint: https://global.example.test\n  token: global-token\n  actor: global-actor\n"
	if err := os.WriteFile(filepath.Join(home, ".synchestra.yaml"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	projectConfig := "# Synchestra Repo Config Schema: https://synchestra.md/repo-config\n\nhub:\n  id: acme/project\n  endpoint: https://project.example.test\n"
	if err := os.WriteFile(filepath.Join(project, "synchestra.yaml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	environment := map[string]string{"SYNCHESTRA_TOKEN": "env-token", "SYNCHESTRA_ACTOR": "env-actor"}
	deps := normalizeDependencies(Dependencies{
		UserHomeDir: func() (string, error) { return home, nil },
		LookupEnv: func(key string) (string, bool) {
			value, ok := environment[key]
			return value, ok
		},
	})
	config, err := loadClientConfig(deps, project)
	if err != nil {
		t.Fatal(err)
	}
	if config.BaseURL != "https://project.example.test" || config.Token != "env-token" || config.Actor != "env-actor" {
		t.Fatalf("config = %+v", config)
	}
	environment["SYNCHESTRA_URL"] = "https://env.example.test"
	config, err = loadClientConfig(deps, project)
	if err != nil {
		t.Fatal(err)
	}
	if config.BaseURL != "https://env.example.test" {
		t.Fatalf("environment URL did not win: %+v", config)
	}
}

func TestCommandRegistersDispatchOperations(t *testing.T) {
	cmd := Command()
	if len(cmd.Commands()) != 1 || cmd.Commands()[0].Name() != "dispatch" {
		t.Fatalf("runner commands = %v", cmd.Commands())
	}
	dispatch := cmd.Commands()[0]
	want := []string{"cancel", "logs", "retry", "status"}
	for index, child := range dispatch.Commands() {
		if index >= len(want) || child.Name() != want[index] {
			t.Fatalf("dispatch commands = %v, want %v", dispatch.Commands(), want)
		}
	}
}
