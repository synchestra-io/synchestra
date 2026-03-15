package globalconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FileNotExists(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), ".synchestra.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "" {
		t.Errorf("expected empty ReposDir, got %q", cfg.ReposDir)
	}
}

func TestLoad_WithReposDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".synchestra.yaml")
	if err := os.WriteFile(cfgPath, []byte("repos_dir: /custom/repos\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "/custom/repos" {
		t.Errorf("expected /custom/repos, got %q", cfg.ReposDir)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".synchestra.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "" {
		t.Errorf("expected empty ReposDir, got %q", cfg.ReposDir)
	}
}

func TestResolveReposDir_Default(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("", homeDir)
	want := filepath.Join(homeDir, "synchestra", "repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveReposDir_TildeExpansion(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("~/my-repos", homeDir)
	want := filepath.Join(homeDir, "my-repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveReposDir_AbsolutePath(t *testing.T) {
	got := ResolveReposDir("/opt/repos", "/home/testuser")
	if got != "/opt/repos" {
		t.Errorf("got %q, want /opt/repos", got)
	}
}

func TestResolveReposDir_RelativePath(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("repos", homeDir)
	want := filepath.Join(homeDir, "repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
