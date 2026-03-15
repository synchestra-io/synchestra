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
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-spec.yaml"))
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
		SpecRepo: "https://github.com/acme/acme-spec",
	}
	if err := WriteStateConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-state.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "spec_repo: https://github.com/acme/acme-spec") {
		t.Errorf("state config missing spec_repo\ngot:\n%s", content)
	}
}

func TestWriteTargetConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := TargetConfig{
		SpecRepo: "https://github.com/acme/acme-spec",
	}
	if err := WriteTargetConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-target.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "spec_repo: https://github.com/acme/acme-spec") {
		t.Errorf("target config missing spec_repo\ngot:\n%s", content)
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
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec.yaml"), []byte(content), 0644); err != nil {
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
	content := "spec_repo: https://github.com/org/spec\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-state.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadStateConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SpecRepo != "https://github.com/org/spec" {
		t.Errorf("SpecRepo = %q", cfg.SpecRepo)
	}
}

func TestReadTargetConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "spec_repo: https://github.com/org/spec\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-target.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadTargetConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SpecRepo != "https://github.com/org/spec" {
		t.Errorf("SpecRepo = %q", cfg.SpecRepo)
	}
}
