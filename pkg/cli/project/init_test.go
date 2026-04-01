package project

// Features implemented: cli/project/init, embedded-state

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// initTestRepo creates a git repo with an initial commit and a bare remote,
// returning (workDir, bareDir). The working copy has origin pointing to bare.
func initTestRepo(t *testing.T, name string) (string, string) {
	t.Helper()

	bare := initBareTestRepo(t, name)
	seedBareRepo(t, bare, "# "+name+"\n\nA test project.\n")

	workDir := filepath.Join(t.TempDir(), name)
	if out, err := exec.Command("git", "clone", bare, workDir).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	gitCfg(t, workDir)
	return workDir, bare
}

func gitCfg(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
}

func TestInit_NewProject(t *testing.T) {
	workDir, _ := initTestRepo(t, "myproject")

	// Change to the work directory for the init command.
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"init", "--no-push"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Embedded state initialized") {
		t.Errorf("unexpected output: %s", output)
	}

	// Verify worktree directory exists.
	wtPath := filepath.Join(workDir, ".synchestra")
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf(".synchestra directory missing or not a dir")
	}

	// Verify synchestra-state.yaml on the orphan branch (via worktree).
	stateCfg, err := ReadEmbeddedStateConfig(wtPath)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if stateCfg.Mode != "embedded" {
		t.Errorf("mode = %q, want embedded", stateCfg.Mode)
	}
	if stateCfg.Title != "myproject" {
		t.Errorf("title = %q, want myproject", stateCfg.Title)
	}

	// Verify tasks/README.md exists with board header.
	boardData, err := os.ReadFile(filepath.Join(wtPath, "tasks", "README.md"))
	if err != nil {
		t.Fatalf("reading board: %v", err)
	}
	if !strings.Contains(string(boardData), "| Task | Status |") {
		t.Errorf("board missing table header")
	}

	// Verify synchestra-spec-repo.yaml on main branch.
	specCfg, err := ReadSpecConfig(workDir)
	if err != nil {
		t.Fatalf("reading spec config: %v", err)
	}
	if specCfg.StateRepo != "worktree://synchestra-state" {
		t.Errorf("state_repo = %q, want worktree://synchestra-state", specCfg.StateRepo)
	}

	// Verify .gitignore contains .synchestra.
	gitignore, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), ".synchestra") {
		t.Errorf(".gitignore missing .synchestra entry")
	}
}

func TestInit_Idempotent(t *testing.T) {
	workDir, _ := initTestRepo(t, "idempotent")

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// First init.
	cmd := Command()
	var stdout1 bytes.Buffer
	cmd.SetOut(&stdout1)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", "--no-push"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init — should succeed (idempotent).
	cmd2 := Command()
	var stdout2 bytes.Buffer
	cmd2.SetOut(&stdout2)
	cmd2.SetErr(&bytes.Buffer{})
	cmd2.SetArgs([]string{"init", "--no-push"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	if !strings.Contains(stdout2.String(), "Already initialized") {
		t.Errorf("expected idempotent message, got: %s", stdout2.String())
	}
}

func TestInit_CustomTitle(t *testing.T) {
	workDir, _ := initTestRepo(t, "titled")

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", "--title", "My Custom Title", "--no-push"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	wtPath := filepath.Join(workDir, ".synchestra")
	stateCfg, err := ReadEmbeddedStateConfig(wtPath)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if stateCfg.Title != "My Custom Title" {
		t.Errorf("title = %q, want My Custom Title", stateCfg.Title)
	}
}

func TestInit_CustomBranch(t *testing.T) {
	workDir, _ := initTestRepo(t, "custombranch")

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", "--branch", "my-state", "--no-push"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	specCfg, err := ReadSpecConfig(workDir)
	if err != nil {
		t.Fatalf("reading spec config: %v", err)
	}
	if specCfg.StateRepo != "worktree://my-state" {
		t.Errorf("state_repo = %q, want worktree://my-state", specCfg.StateRepo)
	}
}

func TestInit_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitcode.Error, got %T", err)
	}
	if ee.ExitCode() != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d", ee.ExitCode(), exitcode.NotFound)
	}
}

func TestInit_ConflictsWithDedicatedProject(t *testing.T) {
	workDir, _ := initTestRepo(t, "conflict")

	// Write a spec config to simulate existing dedicated project.
	if err := WriteSpecConfig(workDir, SpecConfig{Title: "Existing", StateRepo: "https://example.com/state"}); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", "--no-push"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitcode.Error, got %T", err)
	}
	if ee.ExitCode() != exitcode.Conflict {
		t.Errorf("exit code = %d, want %d", ee.ExitCode(), exitcode.Conflict)
	}
}

func TestInit_DerivesTitle_FromReadme(t *testing.T) {
	workDir, _ := initTestRepo(t, "readme-title")

	// The seedBareRepo writes "# readme-title\n\n..." which should be picked up.
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", "--no-push"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	wtPath := filepath.Join(workDir, ".synchestra")
	stateCfg, err := ReadEmbeddedStateConfig(wtPath)
	if err != nil {
		t.Fatal(err)
	}
	if stateCfg.Title != "readme-title" {
		t.Errorf("title = %q, want readme-title", stateCfg.Title)
	}
}
