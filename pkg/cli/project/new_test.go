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
	codeBare := initBareTestRepo(t, "code")

	seedBareRepo(t, specBare, "# My Test Project\n\nDescription.\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, codeBare, "# Code\n")

	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "test", "spec")
	stateDir := filepath.Join(reposDir, "local", "test", "state")
	codeDir := filepath.Join(reposDir, "local", "test", "code")

	cloneAndConfigure(t, specBare, specDir, "https://local/test/spec")
	cloneAndConfigure(t, stateBare, stateDir, "https://local/test/state")
	cloneAndConfigure(t, codeBare, codeDir, "https://local/test/code")

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
		"--code-repo", "local/test/code",
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
	if len(specCfg.Repos) != 1 || specCfg.Repos[0] != "https://local/test/code" {
		t.Errorf("Repos = %v", specCfg.Repos)
	}

	stateCfg, err := ReadStateConfig(stateDir)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if len(stateCfg.SpecRepos) != 1 || stateCfg.SpecRepos[0] != "https://local/test/spec" {
		t.Errorf("state SpecRepos = %v", stateCfg.SpecRepos)
	}

	codeCfg, err := ReadCodeConfig(codeDir)
	if err != nil {
		t.Fatalf("reading code config: %v", err)
	}
	if len(codeCfg.SpecRepos) != 1 || codeCfg.SpecRepos[0] != "https://local/test/spec" {
		t.Errorf("code SpecRepos = %v", codeCfg.SpecRepos)
	}
}

func TestRunNew_MultipleCodeRepos(t *testing.T) {
	specBare := initBareTestRepo(t, "spec2")
	stateBare := initBareTestRepo(t, "state2")
	code1Bare := initBareTestRepo(t, "code2a")
	code2Bare := initBareTestRepo(t, "code2b")

	seedBareRepo(t, specBare, "# Multi Code\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, code1Bare, "# C1\n")
	seedBareRepo(t, code2Bare, "# C2\n")

	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "mc", "spec")
	stateDir := filepath.Join(reposDir, "local", "mc", "state")
	code1Dir := filepath.Join(reposDir, "local", "mc", "c1")
	code2Dir := filepath.Join(reposDir, "local", "mc", "c2")

	cloneAndConfigure(t, specBare, specDir, "https://local/mc/spec")
	cloneAndConfigure(t, stateBare, stateDir, "https://local/mc/state")
	cloneAndConfigure(t, code1Bare, code1Dir, "https://local/mc/c1")
	cloneAndConfigure(t, code2Bare, code2Dir, "https://local/mc/c2")

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
		"--spec-repo", "local/mc/spec",
		"--state-repo", "local/mc/state",
		"--code-repo", "local/mc/c1",
		"--code-repo", "local/mc/c2",
		"--title", "Multi Code Project",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	specCfg, err := ReadSpecConfig(specDir)
	if err != nil {
		t.Fatal(err)
	}
	if specCfg.Title != "Multi Code Project" {
		t.Errorf("Title = %q, want Multi Code Project", specCfg.Title)
	}
	if len(specCfg.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d: %v", len(specCfg.Repos), specCfg.Repos)
	}

	for _, cd := range []string{code1Dir, code2Dir} {
		cfg, err := ReadCodeConfig(cd)
		if err != nil {
			t.Fatalf("reading code config from %s: %v", cd, err)
		}
		if len(cfg.SpecRepos) != 1 || cfg.SpecRepos[0] != "https://local/mc/spec" {
			t.Errorf("code SpecRepos = %v", cfg.SpecRepos)
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
			args: []string{"new", "--state-repo", "local/test/state", "--code-repo", "local/test/code"},
			want: "--spec-repo is required",
		},
		{
			name: "missing state repo",
			args: []string{"new", "--spec-repo", "local/test/spec", "--code-repo", "local/test/code"},
			want: "--state-repo is required",
		},
		{
			name: "missing code repo",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state"},
			want: "at least one --code-repo is required",
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
			args: []string{"new", "--spec-repo", "local/test/shared", "--state-repo", "local/test/shared", "--code-repo", "local/test/code"},
		},
		{
			name: "code equals spec",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state", "--code-repo", "local/test/spec"},
		},
		{
			name: "duplicate code repos",
			args: []string{"new", "--spec-repo", "local/test/spec", "--state-repo", "local/test/state", "--code-repo", "local/test/code", "--code-repo", "local/test/code"},
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
