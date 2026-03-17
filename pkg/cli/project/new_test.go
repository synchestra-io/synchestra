package project

// Features implemented: cli/project/new
// Features depended on:  global-config, project-definition

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/pkg/cli/gitops"
	"github.com/synchesta-io/synchestra/pkg/cli/reporef"
)

func initBareTestRepo(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name+".git")
	if out, err := exec.Command("git", "init", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare %s: %v\n%s", name, err, out)
	}
	return dir
}

func seedBareRepo(t *testing.T, bareDir, readmeContent string) {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "seed")
	if err := gitops.Clone(bareDir, tmp); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte(readmeContent), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	run("push", "origin", "HEAD")
}

func cloneAndConfigure(t *testing.T, bare, dest, fetchURL string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Clone(bare, dest); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dest
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("remote", "set-url", "origin", fetchURL)
	run("remote", "set-url", "--push", "origin", bare)
}

func TestRunNew_ViaCobra(t *testing.T) {
	specBare := initBareTestRepo(t, "spec")
	stateBare := initBareTestRepo(t, "state")
	targetBare := initBareTestRepo(t, "target")

	seedBareRepo(t, specBare, "# My Test Project\n\nDescription.\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, targetBare, "# Target\n")

	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "test", "spec")
	stateDir := filepath.Join(reposDir, "local", "test", "state")
	targetDir := filepath.Join(reposDir, "local", "test", "target")

	cloneAndConfigure(t, specBare, specDir, "https://local/test/spec")
	cloneAndConfigure(t, stateBare, stateDir, "https://local/test/state")
	cloneAndConfigure(t, targetBare, targetDir, "https://local/test/target")

	homeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeDir, ".synchestra.yaml"), []byte("repos_dir: "+reposDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)

	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"new",
		"--spec-repo", "local/test/spec",
		"--state-repo", "local/test/state",
		"--target-repo", "local/test/target",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specCfg, err := ReadSpecConfig(specDir)
	if err != nil {
		t.Fatalf("reading spec config: %v", err)
	}
	if specCfg.Title != "My Test Project" {
		t.Errorf("Title = %q, want My Test Project (derived from README heading)", specCfg.Title)
	}
	if specCfg.StateRepo != "https://local/test/state" {
		t.Errorf("StateRepo = %q", specCfg.StateRepo)
	}
	if len(specCfg.Repos) != 1 || specCfg.Repos[0] != "https://local/test/target" {
		t.Errorf("Repos = %v", specCfg.Repos)
	}

	stateCfg, err := ReadStateConfig(stateDir)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if stateCfg.SpecRepo != "https://local/test/spec" {
		t.Errorf("state SpecRepo = %q", stateCfg.SpecRepo)
	}

	targetCfg, err := ReadTargetConfig(targetDir)
	if err != nil {
		t.Fatalf("reading target config: %v", err)
	}
	if targetCfg.SpecRepo != "https://local/test/spec" {
		t.Errorf("target SpecRepo = %q", targetCfg.SpecRepo)
	}
}

func TestRunNew_MultipleTargets(t *testing.T) {
	specBare := initBareTestRepo(t, "spec2")
	stateBare := initBareTestRepo(t, "state2")
	target1Bare := initBareTestRepo(t, "target2a")
	target2Bare := initBareTestRepo(t, "target2b")

	seedBareRepo(t, specBare, "# Multi Target\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, target1Bare, "# T1\n")
	seedBareRepo(t, target2Bare, "# T2\n")

	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "mt", "spec")
	stateDir := filepath.Join(reposDir, "local", "mt", "state")
	target1Dir := filepath.Join(reposDir, "local", "mt", "t1")
	target2Dir := filepath.Join(reposDir, "local", "mt", "t2")

	cloneAndConfigure(t, specBare, specDir, "https://local/mt/spec")
	cloneAndConfigure(t, stateBare, stateDir, "https://local/mt/state")
	cloneAndConfigure(t, target1Bare, target1Dir, "https://local/mt/t1")
	cloneAndConfigure(t, target2Bare, target2Dir, "https://local/mt/t2")

	homeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeDir, ".synchestra.yaml"), []byte("repos_dir: "+reposDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)

	cmd := Command()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"new",
		"--spec-repo", "local/mt/spec",
		"--state-repo", "local/mt/state",
		"--target-repo", "local/mt/t1",
		"--target-repo", "local/mt/t2",
		"--title", "Multi Target Project",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specCfg, err := ReadSpecConfig(specDir)
	if err != nil {
		t.Fatal(err)
	}
	if specCfg.Title != "Multi Target Project" {
		t.Errorf("Title = %q, want Multi Target Project", specCfg.Title)
	}
	if len(specCfg.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d: %v", len(specCfg.Repos), specCfg.Repos)
	}

	for _, td := range []string{target1Dir, target2Dir} {
		cfg, err := ReadTargetConfig(td)
		if err != nil {
			t.Fatalf("reading target config from %s: %v", td, err)
		}
		if cfg.SpecRepo != "https://local/mt/spec" {
			t.Errorf("target SpecRepo = %q", cfg.SpecRepo)
		}
	}
}

func TestRunNew_MissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing spec repo",
			args: []string{"new", "--state-repo", "local/test/state", "--target-repo", "local/test/target"},
			want: "--spec-repo is required",
		},
		{
			name: "missing state repo",
			args: []string{"new", "--spec-repo", "local/test/spec", "--target-repo", "local/test/target"},
			want: "--state-repo is required",
		},
		{
			name: "missing target repo",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state"},
			want: "at least one --target-repo is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			var ee *exitError
			if !errors.As(err, &ee) {
				t.Fatalf("expected exitError, got %T (%v)", err, err)
			}
			if ee.code != 2 {
				t.Fatalf("exit code = %d, want 2", ee.code)
			}
			if ee.msg != tt.want {
				t.Fatalf("message = %q, want %q", ee.msg, tt.want)
			}
		})
	}
}

func TestRunNew_RejectsOverlappingRepos(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "state equals spec",
			args: []string{"new", "--spec-repo", "local/test/shared", "--state-repo", "local/test/shared", "--target-repo", "local/test/target"},
		},
		{
			name: "target equals spec",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state", "--target-repo", "local/test/spec"},
		},
		{
			name: "duplicate targets",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state", "--target-repo", "local/test/target", "--target-repo", "local/test/target"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			var ee *exitError
			if !errors.As(err, &ee) {
				t.Fatalf("expected exitError, got %T (%v)", err, err)
			}
			if ee.code != 2 {
				t.Fatalf("exit code = %d, want 2", ee.code)
			}
		})
	}
}

func TestValidateResolvedRepoPath_RejectsTraversal(t *testing.T) {
	reposDir := filepath.Join(t.TempDir(), "repos")
	err := validateResolvedRepoPath(reposDir, filepath.Join(reposDir, "..", "elsewhere", "repo"), "local/test/repo")
	if err == nil {
		t.Fatal("expected traversal error")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitError, got %T (%v)", err, err)
	}
	if ee.code != 2 {
		t.Fatalf("exit code = %d, want 2", ee.code)
	}
}

func TestValidateResolvedRepoPath_RejectsSymlink(t *testing.T) {
	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "test", "spec")
	if err := os.MkdirAll(filepath.Dir(specDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), specDir); err != nil {
		t.Fatal(err)
	}
	err := validateResolvedRepoPath(reposDir, specDir, "local/test/spec")
	if err == nil {
		t.Fatal("expected symlink error")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitError, got %T (%v)", err, err)
	}
	if ee.code != 1 {
		t.Fatalf("exit code = %d, want 1", ee.code)
	}
}

func TestEnsureCheckoutMatchesRef_AcceptsEquivalentOriginForms(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	setOriginRemote(t, dir, "git@github.com:acme/acme-api")

	ref, err := reporef.Parse("https://github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureCheckoutMatchesRef(dir, ref); err != nil {
		t.Fatalf("expected origin match, got %v", err)
	}
}

func TestEnsureCheckoutMatchesRef_RejectsMismatchedOrigin(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	setOriginRemote(t, dir, "https://github.com/acme/other-api")

	ref, err := reporef.Parse("https://github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	err = ensureCheckoutMatchesRef(dir, ref)
	if err == nil {
		t.Fatal("expected origin mismatch error")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitError, got %T (%v)", err, err)
	}
	if ee.code != 1 {
		t.Fatalf("exit code = %d, want 1", ee.code)
	}
}

func initGitRepo(t *testing.T, dir string) {
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
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")
}

func setOriginRemote(t *testing.T, dir, originURL string) {
	t.Helper()
	cmd := exec.Command("git", "remote", "add", "origin", originURL)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add origin: %v\n%s", err, out)
	}
}

func TestCheckSpecConflict_NoFile(t *testing.T) {
	err := checkSpecConflict(t.TempDir(), "https://example.com/state")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckSpecConflict_SameProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSpecConfig(dir, SpecConfig{Title: "Test", StateRepo: "https://example.com/state"}); err != nil {
		t.Fatal(err)
	}
	err := checkSpecConflict(dir, "https://example.com/state")
	if err != nil {
		t.Errorf("same project should not conflict, got %v", err)
	}
}

func TestCheckSpecConflict_DifferentProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSpecConfig(dir, SpecConfig{Title: "Other", StateRepo: "https://example.com/other-state"}); err != nil {
		t.Fatal(err)
	}
	err := checkSpecConflict(dir, "https://example.com/state")
	if err == nil {
		t.Error("different project should conflict")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 1 {
		t.Errorf("expected exit code 1, got %v", err)
	}
}

func TestCheckBackrefConflict_NoFile(t *testing.T) {
	err := checkBackrefConflict(t.TempDir(), StateConfigFile, "https://example.com/spec")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckBackrefConflict_SameProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteStateConfig(dir, StateConfig{SpecRepo: "https://example.com/spec"}); err != nil {
		t.Fatal(err)
	}
	err := checkBackrefConflict(dir, StateConfigFile, "https://example.com/spec")
	if err != nil {
		t.Errorf("same project should not conflict, got %v", err)
	}
}

func TestCheckBackrefConflict_DifferentProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteStateConfig(dir, StateConfig{SpecRepo: "https://example.com/other-spec"}); err != nil {
		t.Fatal(err)
	}
	err := checkBackrefConflict(dir, StateConfigFile, "https://example.com/spec")
	if err == nil {
		t.Error("different project should conflict")
	}
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 1 {
		t.Errorf("expected exit code 1, got %v", err)
	}
}
