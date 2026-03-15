package gitops_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/gitops"
)

// setupBareRepo creates a bare git repo seeded with one empty commit (simulates a remote).
// Seeding ensures git push works reliably across all git versions.
func setupBareRepo(t *testing.T) string {
	t.Helper()
	// Init a scratch repo, make an initial commit, then push to a bare repo.
	scratch := t.TempDir()
	run(t, scratch, "git", "init")
	run(t, scratch, "git", "config", "user.email", "test@test.com")
	run(t, scratch, "git", "config", "user.name", "Test")
	run(t, scratch, "git", "commit", "--allow-empty", "-m", "init")

	bare := t.TempDir()
	run(t, bare, "git", "init", "--bare")
	run(t, scratch, "git", "remote", "add", "origin", bare)
	run(t, scratch, "git", "push", "-u", "origin", "HEAD")
	return bare
}

// cloneRepo clones a bare repo into a new temp dir and returns the clone path.
func cloneRepo(t *testing.T, remoteDir string) string {
	t.Helper()
	dest := t.TempDir()
	run(t, dest, "git", "clone", remoteDir, ".")
	run(t, dest, "git", "config", "user.email", "test@test.com")
	run(t, dest, "git", "config", "user.name", "Test")
	return dest
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestIsRepo(t *testing.T) {
	runner := gitops.NewRunner()

	// Real git repo
	bare := setupBareRepo(t)
	clone := cloneRepo(t, bare)
	ok, err := runner.IsRepo(clone)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for git repo")
	}

	// Non-git directory
	notGit := t.TempDir()
	ok, err = runner.IsRepo(notGit)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for non-git dir")
	}

	// Non-existent directory returns false, not error
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	ok, err = runner.IsRepo(nonExistent)
	if err != nil {
		t.Fatalf("expected no error for non-existent dir, got: %v", err)
	}
	if ok {
		t.Error("expected false for non-existent dir")
	}
}

func TestOriginURL(t *testing.T) {
	bare := setupBareRepo(t)
	clone := cloneRepo(t, bare)
	runner := gitops.NewRunner()

	url, err := runner.OriginURL(clone)
	if err != nil {
		t.Fatal(err)
	}
	if url != bare {
		t.Errorf("OriginURL = %q, want %q", url, bare)
	}
}

func TestClone(t *testing.T) {
	bare := setupBareRepo(t)
	runner := gitops.NewRunner()

	dest := filepath.Join(t.TempDir(), "cloned")
	if err := runner.Clone(bare, dest); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	ok, err := runner.IsRepo(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("cloned directory is not a git repo")
	}
}

func TestCommitAndPush(t *testing.T) {
	bare := setupBareRepo(t)
	clone := cloneRepo(t, bare)
	runner := gitops.NewRunner()

	// Write a file
	file := filepath.Join(clone, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runner.CommitAndPush(clone, []string{"test.txt"}, "test commit"); err != nil {
		t.Fatalf("CommitAndPush failed: %v", err)
	}

	// Verify it was pushed by cloning again
	dest := filepath.Join(t.TempDir(), "verify")
	if err := runner.Clone(bare, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", data, "hello")
	}
}

func TestPull(t *testing.T) {
	bare := setupBareRepo(t)
	clone1 := cloneRepo(t, bare)
	clone2 := cloneRepo(t, bare)
	runner := gitops.NewRunner()

	// Push a commit from clone1
	if err := os.WriteFile(filepath.Join(clone1, "shared.txt"), []byte("shared"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runner.CommitAndPush(clone1, []string{"shared.txt"}, "add shared"); err != nil {
		t.Fatalf("CommitAndPush failed: %v", err)
	}

	// Pull in clone2
	if err := runner.Pull(clone2); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(clone2, "shared.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "shared" {
		t.Errorf("file content = %q, want %q", data, "shared")
	}
}

func TestPush(t *testing.T) {
	bare := setupBareRepo(t)
	cloneDir := cloneRepo(t, bare)
	runner := gitops.NewRunner()

	// Write a file and commit it locally WITHOUT pushing
	if err := os.WriteFile(filepath.Join(cloneDir, "push.txt"), []byte("pushed"), 0644); err != nil {
		t.Fatal(err)
	}
	run(t, cloneDir, "git", "add", "push.txt")
	run(t, cloneDir, "git", "commit", "-m", "local commit")

	// Push the local commit to the remote
	if err := runner.Push(cloneDir); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify the commit appears in the bare repo by cloning a fresh copy
	dest := filepath.Join(t.TempDir(), "verify")
	if err := runner.Clone(bare, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "push.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "pushed" {
		t.Errorf("file content = %q, want %q", data, "pushed")
	}
}
