package gitops

// Features depended on: cli/project/new, state-store/backends/git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestCurrentBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

func TestCreateBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := CreateBranch(dir, "feature-x"); err != nil {
		t.Fatal(err)
	}
	if !BranchExists(dir, "feature-x") {
		t.Error("branch feature-x should exist after creation")
	}
}

func TestCheckoutBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := CreateBranch(dir, "feature-y"); err != nil {
		t.Fatal(err)
	}
	if err := CheckoutBranch(dir, "feature-y"); err != nil {
		t.Fatal(err)
	}
	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature-y" {
		t.Errorf("expected feature-y, got %q", branch)
	}
}

func TestCreateAndCheckoutBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := CreateAndCheckoutBranch(dir, "feature-z"); err != nil {
		t.Fatal(err)
	}
	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature-z" {
		t.Errorf("expected feature-z, got %q", branch)
	}
}

func TestMergeBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	mainBranch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := CreateAndCheckoutBranch(dir, "merge-src"); err != nil {
		t.Fatal(err)
	}
	mergefile := filepath.Join(dir, "merged.txt")
	if err := os.WriteFile(mergefile, []byte("merged\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, []string{"merged.txt"}, "add merged file"); err != nil {
		t.Fatal(err)
	}

	if err := CheckoutBranch(dir, mainBranch); err != nil {
		t.Fatal(err)
	}
	if err := MergeBranch(dir, "merge-src"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(mergefile); err != nil {
		t.Error("merged.txt should exist after merge")
	}
}

func TestDeleteBranch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := CreateBranch(dir, "to-delete"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBranch(dir, "to-delete"); err != nil {
		t.Fatal(err)
	}
	if BranchExists(dir, "to-delete") {
		t.Error("branch to-delete should not exist after deletion")
	}
}

func TestBranchExists(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	mainBranch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !BranchExists(dir, mainBranch) {
		t.Errorf("expected %s to exist", mainBranch)
	}
	if BranchExists(dir, "nonexistent-branch") {
		t.Error("expected nonexistent-branch to not exist")
	}
}

func TestCommit(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	f := filepath.Join(dir, "committed.txt")
	if err := os.WriteFile(f, []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, []string{"committed.txt"}, "add committed file"); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "-C", dir, "log", "--oneline")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "add committed file") {
		t.Error("commit message not found in log")
	}
}

func TestPush(t *testing.T) {
	bare := initBareRepo(t)

	setupDir := filepath.Join(t.TempDir(), "setup-push")
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

	workDir := filepath.Join(t.TempDir(), "work-push")
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

	if err := os.WriteFile(filepath.Join(workDir, "pushed.txt"), []byte("pushed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Commit(workDir, []string{"pushed.txt"}, "push test"); err != nil {
		t.Fatal(err)
	}
	if err := Push(workDir); err != nil {
		t.Fatal(err)
	}

	// Verify the commit arrived in bare repo
	cmd := exec.Command("git", "-C", bare, "log", "--oneline")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "push test") {
		t.Error("push test commit not found in bare repo log")
	}
}
