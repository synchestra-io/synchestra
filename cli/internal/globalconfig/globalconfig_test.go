package globalconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/globalconfig"
)

func TestLoad_FileNotExist_ReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "synchestra", "repos")
	if cfg.ReposDir != want {
		t.Errorf("ReposDir = %q, want %q", cfg.ReposDir, want)
	}
}

func TestLoad_WithReposDir(t *testing.T) {
	home := t.TempDir()
	content := "repos_dir: /custom/path\n"
	if err := os.WriteFile(filepath.Join(home, ".synchestra.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReposDir != "/custom/path" {
		t.Errorf("ReposDir = %q, want /custom/path", cfg.ReposDir)
	}
}

func TestLoad_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	content := "repos_dir: ~/my/repos\n"
	if err := os.WriteFile(filepath.Join(home, ".synchestra.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "my", "repos")
	if cfg.ReposDir != want {
		t.Errorf("ReposDir = %q, want %q", cfg.ReposDir, want)
	}
}
