package project

// Features implemented: project-definition, cli/project/new

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSpecConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := SpecConfig{
		Title:     "Acme Platform",
		StateRepo: "https://github.com/acme/acme-synchestra",
		Repos: []string{
			"https://github.com/acme/acme-api",
			"https://github.com/acme/acme-web",
		},
	}
	if err := WriteSpecConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-spec-repo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"title: Acme Platform",
		"state_repo: https://github.com/acme/acme-synchestra",
		"- https://github.com/acme/acme-api",
		"- https://github.com/acme/acme-web",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("spec config missing %q\ngot:\n%s", want, content)
		}
	}
}

func TestWriteStateConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := StateConfig{
		Title:    "Acme Platform",
		MainRepo: "https://github.com/acme/acme-spec",
		SpecRepos: []string{
			"https://github.com/acme/acme-spec",
			"https://github.com/acme/acme-rehearse",
		},
		CodeRepos: []string{
			"https://github.com/acme/acme-api",
		},
	}
	if err := WriteStateConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-state-repo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"title: Acme Platform",
		"main_repo: https://github.com/acme/acme-spec",
		"- https://github.com/acme/acme-spec",
		"- https://github.com/acme/acme-rehearse",
		"- https://github.com/acme/acme-api",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("state config missing %q\ngot:\n%s", want, content)
		}
	}
}

func TestWriteCodeConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := CodeConfig{
		SpecRepos: []string{
			"https://github.com/acme/acme-spec",
			"https://github.com/acme/acme-rehearse",
		},
	}
	if err := WriteCodeConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-code-repo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"- https://github.com/acme/acme-spec",
		"- https://github.com/acme/acme-rehearse",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("code config missing %q\ngot:\n%s", want, content)
		}
	}
}

func TestReadSpecConfig_NotExists(t *testing.T) {
	_, err := ReadSpecConfig(t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadSpecConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "title: Test\nstate_repo: https://github.com/org/state\nrepos:\n  - https://github.com/org/code\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Test" {
		t.Errorf("Title = %q, want Test", cfg.Title)
	}
}

func TestReadStateConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "title: Test\nmain_repo: https://github.com/org/spec\nspec_repos:\n  - https://github.com/org/spec\n  - https://github.com/org/rehearse\ncode_repos:\n  - https://github.com/org/api\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-state-repo.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadStateConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Test" {
		t.Errorf("Title = %q, want Test", cfg.Title)
	}
	if cfg.MainRepo != "https://github.com/org/spec" {
		t.Errorf("MainRepo = %q", cfg.MainRepo)
	}
	if len(cfg.SpecRepos) != 2 {
		t.Fatalf("expected 2 spec repos, got %d: %v", len(cfg.SpecRepos), cfg.SpecRepos)
	}
	if cfg.SpecRepos[0] != "https://github.com/org/spec" {
		t.Errorf("SpecRepos[0] = %q", cfg.SpecRepos[0])
	}
	if cfg.SpecRepos[1] != "https://github.com/org/rehearse" {
		t.Errorf("SpecRepos[1] = %q", cfg.SpecRepos[1])
	}
	if len(cfg.CodeRepos) != 1 || cfg.CodeRepos[0] != "https://github.com/org/api" {
		t.Errorf("CodeRepos = %v, want [https://github.com/org/api]", cfg.CodeRepos)
	}
}

func TestReadSpecConfig_PlanningWhatsNext(t *testing.T) {
	dir := t.TempDir()
	content := []byte("title: test\nplanning:\n  whats_next: incremental\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Planning == nil {
		t.Fatal("expected Planning to be non-nil")
	}
	if cfg.Planning.WhatsNext != "incremental" {
		t.Fatalf("expected WhatsNext=incremental, got %s", cfg.Planning.WhatsNext)
	}
}

func TestReadSpecConfig_PlanningWhatsNextDefault(t *testing.T) {
	dir := t.TempDir()
	content := []byte("title: test\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	whatsNext := cfg.WhatsNextMode()
	if whatsNext != "disabled" {
		t.Fatalf("expected default disabled, got %s", whatsNext)
	}
}

func TestReadCodeConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "spec_repos:\n  - https://github.com/org/spec\n  - https://github.com/org/rehearse\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-code-repo.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadCodeConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SpecRepos) != 2 {
		t.Fatalf("expected 2 spec repos, got %d: %v", len(cfg.SpecRepos), cfg.SpecRepos)
	}
	if cfg.SpecRepos[0] != "https://github.com/org/spec" {
		t.Errorf("SpecRepos[0] = %q", cfg.SpecRepos[0])
	}
}
