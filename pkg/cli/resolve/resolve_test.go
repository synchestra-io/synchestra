package resolve

// Features implemented: project-definition/state-repo, embedded-state

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// initGitRepo initialises a bare-minimum git repository in dir so that
// findGitRoot can discover it. It returns dir for convenience.
func initGitRepo(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

// TestStateRepoPath_SpecRepoWithWorktree verifies that a directory containing
// specscore-spec-repo.yaml with a worktree:// state_repo and a .synchestra/
// directory returns the worktree path.
func TestStateRepoPath_SpecRepoWithWorktree(t *testing.T) {
	dir := t.TempDir()

	specYAML := []byte("state_repo: worktree://synchestra-state\n")
	if err := os.WriteFile(filepath.Join(dir, "specscore-spec-repo.yaml"), specYAML, 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, "specscore-spec-repo.yaml"), specYAML, 0o644); err != nil {
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

// TestStateRepoPath_ConfigLess_WorktreeExists verifies config-less mode:
// a git repo with no synchestra config files but a .synchestra/ directory
// at the repo root returns the worktree path.
func TestStateRepoPath_ConfigLess_WorktreeExists(t *testing.T) {
	root := initGitRepo(t, t.TempDir())

	wtDir := filepath.Join(root, ".synchestra")
	if err := os.Mkdir(wtDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Start from a subdirectory to exercise the walk-up logic.
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := StateRepoPath(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wtDir {
		t.Errorf("got %q, want %q", got, wtDir)
	}
}

// TestStateRepoPath_ConfigLess_NoWorktree verifies config-less mode when the
// git repo has no .synchestra/ directory: returns a NotFound error whose
// message mentions "synchestra project init".
func TestStateRepoPath_ConfigLess_NoWorktree(t *testing.T) {
	root := initGitRepo(t, t.TempDir())

	// Start from a subdirectory.
	sub := filepath.Join(root, "c")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := StateRepoPath(sub)
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
	if !strings.Contains(err.Error(), "synchestra project init") {
		t.Errorf("error message should mention 'synchestra project init', got: %s", err.Error())
	}
}
