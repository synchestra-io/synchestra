package project_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/cmd/project"
	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
	"gopkg.in/yaml.v3"
)

// testEnv holds temp dirs and a fake git runner for a test scenario.
type testEnv struct {
	homeDir   string
	reposDir  string
	specDir   string
	stateDir  string
	targetDir string
	runner    gitops.Runner
}

// setupTestEnv creates a home dir, repos dir, and three pre-initialized
// git repos (spec, state, target) with a fake git runner.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	home := t.TempDir()
	reposDir := filepath.Join(home, "synchestra", "repos")

	specDir := filepath.Join(reposDir, "github.com", "acme", "acme-spec")
	stateDir := filepath.Join(reposDir, "github.com", "acme", "acme-state")
	targetDir := filepath.Join(reposDir, "github.com", "acme", "acme-api")

	for _, dir := range []string{specDir, stateDir, targetDir} {
		mustInitGitRepo(t, dir)
	}

	runner := gitops.Runner{
		IsRepo: func(dir string) (bool, error) {
			cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
			return cmd.Run() == nil, nil
		},
		Clone: func(url, dir string) error {
			return nil // repos are pre-created, no clone needed
		},
		OriginURL: func(dir string) (string, error) {
			// Map local path back to canonical HTTPS URL
			rel, _ := filepath.Rel(reposDir, dir)
			return "https://" + filepath.ToSlash(rel), nil
		},
		CommitAndPush: func(dir string, files []string, msg string) error {
			// Actually commit so we can verify files were written
			for _, f := range files {
				run(t, dir, "git", "add", f)
			}
			run(t, dir, "git", "commit", "-m", msg)
			return nil
		},
		Push: func(dir string) error { return nil },
		Pull: func(dir string) error { return nil },
	}

	return &testEnv{
		homeDir:   home,
		reposDir:  reposDir,
		specDir:   specDir,
		stateDir:  stateDir,
		targetDir: targetDir,
		runner:    runner,
	}
}

func mustInitGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v in %s: %v\n%s", name, args, dir, err, out)
	}
}

func readYAML(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}

func TestProjectNew_WritesConfigFiles(t *testing.T) {
	env := setupTestEnv(t)

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
		"--title", "Acme Platform",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify synchestra-spec.yaml
	var spec struct {
		Title     string   `yaml:"title"`
		StateRepo string   `yaml:"state_repo"`
		Repos     []string `yaml:"repos"`
	}
	readYAML(t, filepath.Join(env.specDir, "synchestra-spec.yaml"), &spec)
	if spec.Title != "Acme Platform" {
		t.Errorf("spec.title = %q, want %q", spec.Title, "Acme Platform")
	}
	if spec.StateRepo != "https://github.com/acme/acme-state" {
		t.Errorf("spec.state_repo = %q", spec.StateRepo)
	}
	if len(spec.Repos) != 1 || spec.Repos[0] != "https://github.com/acme/acme-api" {
		t.Errorf("spec.repos = %v", spec.Repos)
	}

	// Verify synchestra-state.yaml
	var state struct {
		SpecRepo string `yaml:"spec_repo"`
	}
	readYAML(t, filepath.Join(env.stateDir, "synchestra-state.yaml"), &state)
	if state.SpecRepo != "https://github.com/acme/acme-spec" {
		t.Errorf("state.spec_repo = %q", state.SpecRepo)
	}

	// Verify synchestra-target.yaml
	var target struct {
		SpecRepo string `yaml:"spec_repo"`
	}
	readYAML(t, filepath.Join(env.targetDir, "synchestra-target.yaml"), &target)
	if target.SpecRepo != "https://github.com/acme/acme-spec" {
		t.Errorf("target.spec_repo = %q", target.SpecRepo)
	}
}

func TestProjectNew_TitleDerivedFromREADME(t *testing.T) {
	env := setupTestEnv(t)

	// Write README.md with a heading to the spec repo
	readme := filepath.Join(env.specDir, "README.md")
	if err := os.WriteFile(readme, []byte("# My Project\n\nSome description.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var spec struct {
		Title string `yaml:"title"`
	}
	readYAML(t, filepath.Join(env.specDir, "synchestra-spec.yaml"), &spec)
	if spec.Title != "My Project" {
		t.Errorf("spec.title = %q, want %q", spec.Title, "My Project")
	}
}

func TestProjectNew_TitleDerivedFromRepoName(t *testing.T) {
	env := setupTestEnv(t)

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var spec struct {
		Title string `yaml:"title"`
	}
	readYAML(t, filepath.Join(env.specDir, "synchestra-spec.yaml"), &spec)
	// No README.md, so title falls back to repo name
	if spec.Title != "acme-spec" {
		t.Errorf("spec.title = %q, want %q", spec.Title, "acme-spec")
	}
}

func TestProjectNew_ConflictExistingConfigDifferentProject(t *testing.T) {
	env := setupTestEnv(t)

	// Write a state config pointing to a different spec
	existing := "spec_repo: https://github.com/other/other-spec\n"
	if err := os.WriteFile(
		filepath.Join(env.stateDir, "synchestra-state.yaml"),
		[]byte(existing), 0644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var exitErr *exitcode.Error
	if !asExitError(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected exitcode 1, got: %v", err)
	}
}

func TestProjectNew_MissingRequiredFlags(t *testing.T) {
	env := setupTestEnv(t)
	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		// missing --state-repo and --target-repo
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProjectNew_MultipleTargetRepos(t *testing.T) {
	env := setupTestEnv(t)

	// Create second target dir
	target2Dir := filepath.Join(env.reposDir, "github.com", "acme", "acme-web")
	mustInitGitRepo(t, target2Dir)

	env.runner.OriginURL = func(dir string) (string, error) {
		rel, _ := filepath.Rel(env.reposDir, dir)
		return "https://" + filepath.ToSlash(rel), nil
	}

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
		"--target-repo", "github.com/acme/acme-web",
		"--title", "Multi-target",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var spec struct {
		Repos []string `yaml:"repos"`
	}
	readYAML(t, filepath.Join(env.specDir, "synchestra-spec.yaml"), &spec)
	if len(spec.Repos) != 2 {
		t.Errorf("spec.repos length = %d, want 2; repos = %v", len(spec.Repos), spec.Repos)
	}

	// Verify both targets got synchestra-target.yaml
	for _, dir := range []string{env.targetDir, target2Dir} {
		if _, err := os.Stat(filepath.Join(dir, "synchestra-target.yaml")); err != nil {
			t.Errorf("synchestra-target.yaml missing in %s: %v", dir, err)
		}
	}
}

func TestProjectNew_PushConflictRetrySucceeds(t *testing.T) {
	env := setupTestEnv(t)

	// First CommitAndPush fails (simulates push conflict), then Push succeeds.
	env.runner.CommitAndPush = func(dir string, files []string, msg string) error {
		for _, f := range files {
			run(t, dir, "git", "add", f)
		}
		run(t, dir, "git", "commit", "-m", msg)
		return fmt.Errorf("push rejected: non-fast-forward") // simulate conflict
	}
	env.runner.Push = func(dir string) error { return nil }

	cmd := project.NewCommand(func() (string, error) { return env.homeDir, nil }, env.runner)
	cmd.SetArgs([]string{
		"--spec-repo", "github.com/acme/acme-spec",
		"--state-repo", "github.com/acme/acme-state",
		"--target-repo", "github.com/acme/acme-api",
		"--title", "Retry Test",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
}

// asExitError checks if err (or any wrapped error) is an *exitcode.Error.
func asExitError(err error, target **exitcode.Error) bool {
	return errors.As(err, target)
}
