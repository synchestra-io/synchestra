package synchinit

// Features implemented: cli/init

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/project"
)

// initBareTestRepo creates a bare remote at $TMPDIR/<name>.git and returns
// its path. Mirrors project.initBareTestRepo's pattern (which is private).
func initBareTestRepo(t *testing.T, name string) string {
	t.Helper()
	bareDir := filepath.Join(t.TempDir(), name+".git")
	if out, err := exec.Command("git", "init", "--bare", bareDir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return bareDir
}

// seedBareRepo pushes one commit to the bare remote so it has a HEAD.
func seedBareRepo(t *testing.T, bareDir, readmeBody string) {
	t.Helper()
	seed := filepath.Join(t.TempDir(), "seed")
	if out, err := exec.Command("git", "clone", bareDir, seed).CombinedOutput(); err != nil {
		t.Fatalf("clone bare for seeding: %v\n%s", err, out)
	}
	gitCfg(t, seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte(readmeBody), 0o644); err != nil {
		t.Fatalf("seed README: %v", err)
	}
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = seed
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v in seed: %v\n%s", args, err, out)
		}
	}
	run("add", "README.md")
	run("commit", "-m", "seed")
	run("push", "origin", "HEAD:main")
}

// initTestRepo clones a freshly-seeded bare remote and returns the work dir.
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
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
}

// runInitCmd executes synchestra init with the given args inside workDir.
// Returns (stdout, stderr, error). chdir is restored on cleanup.
func runInitCmd(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestInit_Greenfield exercises AC: greenfield-embedded-init.
func TestInit_Greenfield(t *testing.T) {
	workDir, _ := initTestRepo(t, "greenfield")

	stdout, stderr, err := runInitCmd(t, workDir, "--no-push")
	if err != nil {
		t.Fatalf("init: %v\nstderr: %s", err, stderr)
	}

	// synchestra.yaml exists with canonical line 1.
	cfgPath := filepath.Join(workDir, SynchestraConfigFile)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading synchestra.yaml: %v", err)
	}
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) < 1 || lines[0] != synchestraSchemaHeader {
		t.Errorf("line 1 not canonical schema header; got %q", lines[0])
	}
	if !strings.Contains(string(data), "mode: embedded") {
		t.Errorf("synchestra.yaml missing state.mode=embedded; got: %s", data)
	}
	if !strings.Contains(string(data), "branch: synchestra-state") {
		t.Errorf("synchestra.yaml missing branch=synchestra-state; got: %s", data)
	}

	// Worktree exists at .synchestra/.
	wtPath := filepath.Join(workDir, ".synchestra")
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf(".synchestra directory missing or not a dir: %v", err)
	}

	// synchestra-state.yaml exists on the orphan branch (read via worktree).
	stateCfg, err := project.ReadEmbeddedStateConfig(wtPath)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if stateCfg.Mode != "embedded" {
		t.Errorf("state mode = %q, want embedded", stateCfg.Mode)
	}

	// .gitignore contains .synchestra.
	gi, err := os.ReadFile(filepath.Join(workDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), ".synchestra") {
		t.Errorf(".gitignore missing .synchestra entry; got: %s", gi)
	}

	// stdout summary contains expected substrings.
	if !strings.Contains(stdout, "Synchestra project initialized") {
		t.Errorf("stdout missing init banner; got: %s", stdout)
	}
	if !strings.Contains(stdout, "Next:") {
		t.Errorf("stdout missing next-step hint; got: %s", stdout)
	}
}

// TestInit_RejectsNonGitDir exercises AC: rejects-non-git-dir.
func TestInit_RejectsNonGitDir(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := runInitCmd(t, dir)
	if err == nil {
		t.Fatalf("expected error in non-git dir; stdout=%s stderr=%s", stdout, stderr)
	}
	var ec *exitcode.Error
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if ec.ExitCode() != exitcode.NotFound {
		t.Errorf("exit code = %d, want %d (NotFound)", ec.ExitCode(), exitcode.NotFound)
	}
	if !strings.Contains(err.Error(), "git init") {
		t.Errorf("error should instruct user to run `git init`; got: %v", err)
	}
	// No file should have been created.
	if _, err := os.Stat(filepath.Join(dir, SynchestraConfigFile)); !os.IsNotExist(err) {
		t.Errorf("synchestra.yaml unexpectedly exists in non-git dir")
	}
}

// TestInit_RejectsUnknownMode exercises AC: rejects-unknown-mode.
func TestInit_RejectsUnknownMode(t *testing.T) {
	workDir, _ := initTestRepo(t, "unknown-mode")
	_, _, err := runInitCmd(t, workDir, "--state-mode", "magical", "--no-push")
	if err == nil {
		t.Fatal("expected error for unknown --state-mode")
	}
	var ec *exitcode.Error
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitcode.Error, got %T", err)
	}
	if ec.ExitCode() != exitcode.InvalidArgs {
		t.Errorf("exit code = %d, want %d (InvalidArgs)", ec.ExitCode(), exitcode.InvalidArgs)
	}
	if !strings.Contains(err.Error(), "magical") || !strings.Contains(err.Error(), "embedded") {
		t.Errorf("error should name invalid value and accepted values; got: %v", err)
	}
	// No synchestra.yaml created.
	if _, err := os.Stat(filepath.Join(workDir, SynchestraConfigFile)); !os.IsNotExist(err) {
		t.Errorf("synchestra.yaml unexpectedly exists after rejected init")
	}
}

// TestInit_RejectsUnimplementedMode exercises AC: rejects-unimplemented-mode.
func TestInit_RejectsUnimplementedMode(t *testing.T) {
	workDir, _ := initTestRepo(t, "unimpl-mode")
	for _, mode := range []string{"separate-repo", "hub-managed"} {
		t.Run(mode, func(t *testing.T) {
			_, _, err := runInitCmd(t, workDir, "--state-mode", mode, "--no-push")
			if err == nil {
				t.Fatalf("expected error for --state-mode=%s", mode)
			}
			var ec *exitcode.Error
			if !errors.As(err, &ec) {
				t.Fatalf("expected *exitcode.Error, got %T", err)
			}
			if ec.ExitCode() != exitcode.InvalidArgs {
				t.Errorf("exit code = %d, want %d (InvalidArgs)", ec.ExitCode(), exitcode.InvalidArgs)
			}
			if !strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("error should say not yet implemented; got: %v", err)
			}
			if !strings.Contains(err.Error(), mode) {
				t.Errorf("error should name the requested mode; got: %v", err)
			}
		})
	}
}

// TestInit_IdempotentRerun exercises AC: idempotent-rerun.
func TestInit_IdempotentRerun(t *testing.T) {
	workDir, _ := initTestRepo(t, "idempotent")

	// First run.
	if _, stderr, err := runInitCmd(t, workDir, "--no-push"); err != nil {
		t.Fatalf("first init: %v\nstderr: %s", err, stderr)
	}

	cfgPath := filepath.Join(workDir, SynchestraConfigFile)
	stat1, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat after first init: %v", err)
	}
	body1, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Second run — must be no-op.
	stdout, stderr, err := runInitCmd(t, workDir, "--no-push")
	if err != nil {
		t.Fatalf("second init: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "already initialized") {
		t.Errorf("rerun stdout should indicate already initialized; got: %s", stdout)
	}

	stat2, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	body2, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// Byte-for-byte identical (rerun did not rewrite).
	if !bytes.Equal(body1, body2) {
		t.Errorf("rerun rewrote synchestra.yaml; before=%q after=%q", body1, body2)
	}
	// mtime unchanged (proves the file was not rewritten).
	if !stat1.ModTime().Equal(stat2.ModTime()) {
		t.Errorf("rerun changed mtime: before=%v after=%v", stat1.ModTime(), stat2.ModTime())
	}
}

// TestInit_RejectsModeMismatch exercises AC: rejects-mode-mismatch.
func TestInit_RejectsModeMismatch(t *testing.T) {
	workDir, _ := initTestRepo(t, "mismatch")

	// First run — embedded.
	if _, stderr, err := runInitCmd(t, workDir, "--no-push"); err != nil {
		t.Fatalf("first init: %v\nstderr: %s", err, stderr)
	}

	// Second run — request separate-repo. Because separate-repo is not
	// implemented in v1, the InvalidArgs path runs first and we exit 2,
	// not 4. To exercise the mode-mismatch path proper, manually rewrite
	// synchestra.yaml to declare a different mode and rerun.
	cfgPath := filepath.Join(workDir, SynchestraConfigFile)
	body := synchestraSchemaHeader + "\n\nstate:\n  mode: separate-repo\n  repo: https://example.com/state\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runInitCmd(t, workDir, "--no-push") // default mode = embedded
	if err == nil {
		t.Fatal("expected mode-mismatch error")
	}
	var ec *exitcode.Error
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitcode.Error, got %T", err)
	}
	if ec.ExitCode() != exitcode.InvalidState {
		t.Errorf("exit code = %d, want %d (InvalidState)", ec.ExitCode(), exitcode.InvalidState)
	}
	if !strings.Contains(err.Error(), "embedded") || !strings.Contains(err.Error(), "separate-repo") {
		t.Errorf("error should name both modes; got: %v", err)
	}
}

// TestInit_ProjectFlagResolves exercises AC: project-flag-resolves.
func TestInit_ProjectFlagResolves(t *testing.T) {
	workDir, _ := initTestRepo(t, "projflag")

	// Run from a different directory, pointing at workDir via --project.
	otherDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(otherDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--project", workDir, "--no-push"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init via --project: %v\nstderr: %s", err, stderr.String())
	}

	// synchestra.yaml at the --project root, NOT at the cwd.
	if _, err := os.Stat(filepath.Join(workDir, SynchestraConfigFile)); err != nil {
		t.Errorf("synchestra.yaml not at --project root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(otherDir, SynchestraConfigFile)); !os.IsNotExist(err) {
		t.Errorf("synchestra.yaml unexpectedly at cwd")
	}
}

// TestInit_RejectsNonExistentProject exercises the negative half of
// AC: project-flag-resolves.
func TestInit_RejectsNonExistentProject(t *testing.T) {
	bogus := "/nonexistent/path/that/should/never/be/there"
	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--project", bogus, "--no-push"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent --project")
	}
	var ec *exitcode.Error
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitcode.Error, got %T", err)
	}
	if ec.ExitCode() != exitcode.InvalidArgs {
		t.Errorf("exit code = %d, want %d (InvalidArgs)", ec.ExitCode(), exitcode.InvalidArgs)
	}
}

// TestInit_TopLevelHelp exercises AC: top-level-registered (parent-level
// registration is asserted in cli/main_test.go style integration; here we
// at least confirm Command() returns a registered cobra.Command with the
// expected Use string and key flags).
func TestInit_TopLevelHelp(t *testing.T) {
	cmd := Command()
	if cmd.Use != "init" {
		t.Errorf("Use = %q, want init", cmd.Use)
	}
	for _, flag := range []string{"state-mode", "branch", "project", "no-push"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing flag --%s", flag)
		}
	}
}
