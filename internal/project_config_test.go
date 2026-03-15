package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		name          string
		readme        string // content of README.md, empty if file should not exist
		specRef       *RepoRef
		providedTitle string
		expected      string
	}{
		{
			name:          "explicit title",
			readme:        "# Some Project\n",
			specRef:       &RepoRef{Repo: "my-repo"},
			providedTitle: "My Project",
			expected:      "My Project",
		},
		{
			name:          "from README H1",
			readme:        "# My Awesome Project\n\n## Overview\n",
			specRef:       &RepoRef{Repo: "my-repo"},
			providedTitle: "",
			expected:      "My Awesome Project",
		},
		{
			name:          "ignores H2 if H1 exists",
			readme:        "# Main Title\n## Subtitle\n",
			specRef:       &RepoRef{Repo: "my-repo"},
			providedTitle: "",
			expected:      "Main Title",
		},
		{
			name:          "falls back to repo identifier",
			readme:        "",
			specRef:       &RepoRef{Repo: "my-awesome-repo"},
			providedTitle: "",
			expected:      "my-awesome-repo",
		},
		{
			name:          "README without H1",
			readme:        "## Overview\nSome content\n",
			specRef:       &RepoRef{Repo: "my-repo"},
			providedTitle: "",
			expected:      "my-repo",
		},
		{
			name:          "H1 with no text",
			readme:        "#\n## Real Title\n",
			specRef:       &RepoRef{Repo: "my-repo"},
			providedTitle: "",
			expected:      "my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.readme != "" {
				readmePath := filepath.Join(tmpDir, "README.md")
				err := os.WriteFile(readmePath, []byte(tt.readme), 0644)
				if err != nil {
					t.Fatalf("failed to write README: %v", err)
				}
			}

			got := DeriveTitle(tmpDir, tt.specRef, tt.providedTitle)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWriteAndReadSpecConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SynchstraSpecYaml{
		Title:     "Test Project",
		StateRepo: "https://github.com/test/state",
		Repos:     []string{"https://github.com/test/api", "https://github.com/test/web"},
	}

	// Write config
	err := WriteSpecConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read it back
	read, err := ReadSpecConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if read == nil {
		t.Fatalf("read config is nil")
	}

	if read.Title != cfg.Title {
		t.Errorf("Title: got %q, want %q", read.Title, cfg.Title)
	}
	if read.StateRepo != cfg.StateRepo {
		t.Errorf("StateRepo: got %q, want %q", read.StateRepo, cfg.StateRepo)
	}
	if len(read.Repos) != len(cfg.Repos) {
		t.Errorf("Repos length: got %d, want %d", len(read.Repos), len(cfg.Repos))
	}
	for i, repo := range read.Repos {
		if repo != cfg.Repos[i] {
			t.Errorf("Repos[%d]: got %q, want %q", i, repo, cfg.Repos[i])
		}
	}
}

func TestWriteAndReadStateConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SynchstraStateYaml{
		SpecRepo: "https://github.com/test/spec",
	}

	// Write
	err := WriteStateConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read back
	read, err := ReadStateConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if read == nil {
		t.Fatalf("read config is nil")
	}

	if read.SpecRepo != cfg.SpecRepo {
		t.Errorf("SpecRepo: got %q, want %q", read.SpecRepo, cfg.SpecRepo)
	}
}

func TestReadNonexistentConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return nil, not error
	cfg, err := ReadSpecConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg != nil {
		t.Fatalf("expected nil config for nonexistent file")
	}
}
