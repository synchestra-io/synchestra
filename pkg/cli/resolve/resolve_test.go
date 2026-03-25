package resolve

// Features implemented: project-definition/state-repo, embedded-state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

// TestStateRepoPath_SpecRepoWithWorktree verifies that a directory containing
// synchestra-spec-repo.yaml with a worktree:// state_repo and a .synchestra/
// directory returns the worktree path.
func TestStateRepoPath_SpecRepoWithWorktree(t *testing.T) {
	dir := t.TempDir()

	specYAML := []byte("state_repo: worktree://synchestra-state\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), specYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	wtDir := filepath.Join(dir, ".synchestra")
	if err := os.Mkdir(wtDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := StateRepoPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wtDir {
		t.Errorf("got %q, want %q", got, wtDir)
	}
}

// TestStateRepoPath_SpecRepoWorktreeMissing verifies that a worktree://
// config without the .synchestra/ directory returns a NotFound error.
func TestStateRepoPath_SpecRepoWorktreeMissing(t *testing.T) {
	dir := t.TempDir()

	specYAML := []byte("state_repo: worktree://synchestra-state\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), specYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := StateRepoPath(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ecErr *exitcode.Error
	if !errors.As(err, &ecErr) {
		t.Fatalf("expected exitcode.Error, got %T: %v", err, err)
	}
	if ecErr.ExitCode() != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d", ecErr.ExitCode(), exitcode.NotFound)
	}
}

// TestStateRepoPath_StateRepoYAML verifies that a directory containing
// synchestra-state-repo.yaml is returned directly as the state repo path.
func TestStateRepoPath_StateRepoYAML(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "synchestra-state-repo.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := StateRepoPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// filepath.Abs may resolve symlinks in TempDir; compare cleaned paths.
	wantAbs, _ := filepath.Abs(dir)
	if got != wantAbs {
		t.Errorf("got %q, want %q", got, wantAbs)
	}
}

// TestStateRepoPath_NoConfig verifies that an empty directory with no config
// files returns a NotFound error.
func TestStateRepoPath_NoConfig(t *testing.T) {
	dir := t.TempDir()

	_, err := StateRepoPath(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ecErr *exitcode.Error
	if !errors.As(err, &ecErr) {
		t.Fatalf("expected exitcode.Error, got %T: %v", err, err)
	}
	if ecErr.ExitCode() != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d", ecErr.ExitCode(), exitcode.NotFound)
	}
}
