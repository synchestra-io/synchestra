package gitops

// Features depended on: cli/project/new

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bare.git")
	cmd := exec.Command("git", "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	// Create initial commit so HEAD exists
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")
}

func TestIsGitRepo_True(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if !IsGitRepo(dir) {
		t.Error("expected true for git repo")
	}
}

func TestIsGitRepo_False(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("expected false for non-git dir")
	}
}

func TestIsGitRepo_NotExists(t *testing.T) {
	if IsGitRepo("/nonexistent/path") {
		t.Error("expected false for nonexistent path")
	}
}

func TestGetOriginURL(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/acme/acme-api")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}
	url, err := GetOriginURL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://github.com/acme/acme-api" {
		t.Errorf("got %q, want https://github.com/acme/acme-api", url)
	}
}

func TestClone(t *testing.T) {
	bare := initBareRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")
	if err := Clone(bare, dest); err != nil {
		t.Fatal(err)
	}
	if !IsGitRepo(dest) {
		t.Error("cloned dir should be a git repo")
	}
}

func TestCommitAndPush(t *testing.T) {
	// Set up bare remote + working clone
	bare := initBareRepo(t)

	// Create initial commit in bare repo via a temp clone
	setupDir := filepath.Join(t.TempDir(), "setup")
	if err := Clone(bare, setupDir); err != nil {
		t.Fatal(err)
	}
	setupRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = setupDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	setupRun("config", "user.email", "test@test.com")
	setupRun("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(setupDir, "README.md"), []byte("init\n"), 0644); err != nil {
		t.Fatal(err)
	}
	setupRun("add", ".")
	setupRun("commit", "-m", "init")
	setupRun("push", "origin", "HEAD")

	// Now clone fresh and test CommitAndPush
	workDir := filepath.Join(t.TempDir(), "work")
	if err := Clone(bare, workDir); err != nil {
		t.Fatal(err)
	}
	workRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	workRun("config", "user.email", "test@test.com")
	workRun("config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CommitAndPush(workDir, []string{"test.txt"}, "test commit"); err != nil {
		t.Fatal(err)
	}
}
