package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_DefaultValues(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/testuser", nil
	}

	cfg, err := LoadGlobalConfig(mockHomeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/home/testuser/synchestra/repos"
	if cfg.ReposDir != expected {
		t.Errorf("ReposDir: got %q, want %q", cfg.ReposDir, expected)
	}
}

func TestExpandHome_Tilde(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		homeDir  string
		expected string
	}{
		{
			name:     "tilde only",
			path:     "~",
			homeDir:  "/home/user",
			expected: "/home/user",
		},
		{
			name:     "tilde slash path",
			path:     "~/repos",
			homeDir:  "/home/user",
			expected: "/home/user/repos",
		},
		{
			name:     "tilde slash nested",
			path:     "~/a/b/c",
			homeDir:  "/home/user",
			expected: "/home/user/a/b/c",
		},
		{
			name:     "no tilde",
			path:     "/absolute/path",
			homeDir:  "/home/user",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			path:     "relative/path",
			homeDir:  "/home/user",
			expected: "relative/path",
		},
		{
			name:     "reject tilde username",
			path:     "~otheruser/path",
			homeDir:  "/home/user",
			expected: "~otheruser/path", // Not expanded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.path, tt.homeDir)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadGlobalConfig_FileRead(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".synchestra.yaml")
	err := os.WriteFile(configFile, []byte("repos_dir: /custom/repos\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mockHomeDir := func() (string, error) {
		return tmpDir, nil
	}

	cfg, err := LoadGlobalConfig(mockHomeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/custom/repos"
	if cfg.ReposDir != expected {
		t.Errorf("ReposDir: got %q, want %q", cfg.ReposDir, expected)
	}
}
