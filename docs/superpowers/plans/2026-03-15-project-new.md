# `synchestra project new` Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `synchestra project new` — the CLI command that creates a Synchestra project by linking a spec repo, state repo, and one or more target repos, writing config files, committing, and pushing.

**Architecture:** The command is decomposed into focused internal packages (`exitcode`, `globalconfig`, `reporef`, `gitops`) each with a single responsibility, wired together in `cli/cmd/project/new.go`. Git operations use an injected `Runner` struct with function fields so tests can swap real implementations for fakes. The `cli/main.go` is updated to accept an `exit func(int)` parameter so commands can emit specific exit codes.

**Tech Stack:** Go 1.26, Cobra, gopkg.in/yaml.v3, os/exec (git), standard library

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `cli/internal/exitcode/exitcode.go` | Create | `Error` type wrapping a code + error |
| `cli/internal/globalconfig/globalconfig.go` | Create | Load `~/.synchestra.yaml`; default `repos_dir` |
| `cli/internal/globalconfig/globalconfig_test.go` | Create | Tests for config loading |
| `cli/internal/reporef/reporef.go` | Create | Parse repo references; resolve to local path + origin URL |
| `cli/internal/reporef/reporef_test.go` | Create | Tests for parsing and resolving |
| `cli/internal/gitops/gitops.go` | Create | `Runner` struct + real git implementations |
| `cli/internal/gitops/gitops_test.go` | Create | Integration tests using real git repos in temp dirs |
| `cli/cmd/project/project.go` | Create | `project` command group |
| `cli/cmd/project/new.go` | Create | `project new` command logic |
| `cli/cmd/project/new_test.go` | Create | End-to-end tests with injected fake git runner |
| `cli/main.go` | Modify | Add `exit func(int)` param; handle `exitcode.Error`; register `project` command |
| `main.go` | Modify | Pass `os.Exit` as explicit `exit` to `cli.Run` |

---

## Chunk 1: Foundation — Exit Codes, Global Config, Repo References

### Task 1: Exit code error type

**Files:**
- Create: `cli/internal/exitcode/exitcode.go`

- [ ] **Step 1: Write failing test**

```go
// cli/internal/exitcode/exitcode_test.go
package exitcode_test

import (
	"errors"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
)

func TestError_Error(t *testing.T) {
	e := exitcode.New(3, "repo %q not found", "foo")
	if e.Error() != `repo "foo" not found` {
		t.Errorf("Error() = %q", e.Error())
	}
	if e.Code != 3 {
		t.Errorf("Code = %d, want 3", e.Code)
	}
}

func TestError_Unwrap(t *testing.T) {
	e := exitcode.New(1, "conflict")
	var target *exitcode.Error
	if !errors.As(e, &target) {
		t.Error("errors.As should find *exitcode.Error")
	}
	if target.Code != 1 {
		t.Errorf("Code = %d, want 1", target.Code)
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./cli/internal/exitcode/... -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Create `exitcode.go`**

```go
// Package exitcode defines an error type that carries a process exit code.
package exitcode

import "fmt"

// Error is an error that carries a specific process exit code.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// New returns an *Error with the given exit code and formatted message.
func New(code int, format string, args ...any) *Error {
	return &Error{Code: code, Err: fmt.Errorf(format, args...)}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cli/internal/exitcode/... -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cli/internal/exitcode/
git commit -m "feat: add exitcode.Error for typed exit codes"
```

---

### Task 2: Global config loading

**Files:**
- Create: `cli/internal/globalconfig/globalconfig.go`
- Create: `cli/internal/globalconfig/globalconfig_test.go`

- [ ] **Step 1: Write failing test**

```go
// cli/internal/globalconfig/globalconfig_test.go
package globalconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/globalconfig"
)

func TestLoad_FileNotExist_ReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "synchestra", "repos")
	if cfg.ReposDir != want {
		t.Errorf("ReposDir = %q, want %q", cfg.ReposDir, want)
	}
}

func TestLoad_WithReposDir(t *testing.T) {
	home := t.TempDir()
	content := "repos_dir: /custom/path\n"
	if err := os.WriteFile(filepath.Join(home, ".synchestra.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReposDir != "/custom/path" {
		t.Errorf("ReposDir = %q, want /custom/path", cfg.ReposDir)
	}
}

func TestLoad_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	content := "repos_dir: ~/my/repos\n"
	if err := os.WriteFile(filepath.Join(home, ".synchestra.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := globalconfig.Load(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "my", "repos")
	if cfg.ReposDir != want {
		t.Errorf("ReposDir = %q, want %q", cfg.ReposDir, want)
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./cli/internal/globalconfig/... -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement globalconfig.go**

```go
// Package globalconfig loads ~/.synchestra.yaml.
package globalconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig is the user-level Synchestra configuration from ~/.synchestra.yaml.
type GlobalConfig struct {
	ReposDir string `yaml:"repos_dir"`
}

// Load reads ~/.synchestra.yaml from homeDir and returns the config.
// Missing file returns defaults; invalid YAML returns an error.
func Load(homeDir string) (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	data, err := os.ReadFile(filepath.Join(homeDir, ".synchestra.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		cfg.ReposDir = filepath.Join(homeDir, "synchestra", "repos")
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.ReposDir = expandTilde(cfg.ReposDir, homeDir)
	if cfg.ReposDir == "" {
		cfg.ReposDir = filepath.Join(homeDir, "synchestra", "repos")
	}
	return cfg, nil
}

func expandTilde(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cli/internal/globalconfig/... -v
```
Expected: all PASS

- [ ] **Step 5: Tidy module dependencies**

`gopkg.in/yaml.v3` is currently an indirect dep; importing it directly requires promoting it.

```bash
go mod tidy
```
Expected: `go.mod` updated (yaml.v3 moves to direct), `go.sum` unchanged or updated

- [ ] **Step 6: Commit**

```bash
git add cli/internal/globalconfig/ go.mod go.sum
git commit -m "feat: add globalconfig.Load for ~/.synchestra.yaml"
```

---

### Task 3: Repo reference parsing and resolution

**Files:**
- Create: `cli/internal/reporef/reporef.go`
- Create: `cli/internal/reporef/reporef_test.go`

- [ ] **Step 1: Write failing tests**

```go
// cli/internal/reporef/reporef_test.go
package reporef_test

import (
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/reporef"
)

func TestParse(t *testing.T) {
	cases := []struct {
		input       string
		wantHosting string
		wantOrg     string
		wantRepo    string
		wantErr     bool
	}{
		{
			input:       "github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		{
			input:       "https://github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		{
			input:       "git@github.com:acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// .git suffix stripped
		{
			input:       "https://github.com/acme/acme-spec.git",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// http:// (non-TLS)
		{
			input:       "http://github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// sub-paths are invalid
		{input: "github.com/org/repo/extra", wantErr: true},
		{input: "notaref", wantErr: true},
		{input: "github.com/only-one-segment", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := reporef.Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Hosting != tc.wantHosting {
				t.Errorf("Hosting = %q, want %q", got.Hosting, tc.wantHosting)
			}
			if got.Org != tc.wantOrg {
				t.Errorf("Org = %q, want %q", got.Org, tc.wantOrg)
			}
			if got.Repo != tc.wantRepo {
				t.Errorf("Repo = %q, want %q", got.Repo, tc.wantRepo)
			}
		})
	}
}

func TestRef_LocalPath(t *testing.T) {
	ref, err := reporef.Parse("github.com/acme/acme-spec")
	if err != nil {
		t.Fatal(err)
	}
	got := ref.LocalPath("/home/user/synchestra/repos")
	want := filepath.Join("/home/user/synchestra/repos", "github.com", "acme", "acme-spec")
	if got != want {
		t.Errorf("LocalPath = %q, want %q", got, want)
	}
}

func TestRef_OriginURL(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"github.com/acme/acme-spec", "https://github.com/acme/acme-spec"},
		{"https://github.com/acme/acme-spec", "https://github.com/acme/acme-spec"},
		{"git@github.com:acme/acme-spec", "https://github.com/acme/acme-spec"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			ref, err := reporef.Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got := ref.OriginURL(); got != tc.want {
				t.Errorf("OriginURL = %q, want %q", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./cli/internal/reporef/... -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement reporef.go**

```go
// Package reporef parses and resolves Synchestra repository references.
package reporef

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Ref is a parsed repository reference.
type Ref struct {
	Hosting string // e.g. "github.com"
	Org     string // e.g. "acme"
	Repo    string // e.g. "acme-spec"
}

// Parse parses a repo reference in any supported format:
//   - Short form:  "github.com/org/repo"
//   - HTTPS URL:   "https://github.com/org/repo" or "https://github.com/org/repo.git"
//   - SSH URL:     "git@github.com:org/repo" or "git@github.com:org/repo.git"
func Parse(ref string) (Ref, error) {
	var path string
	switch {
	case strings.HasPrefix(ref, "https://"):
		path = strings.TrimPrefix(ref, "https://")
	case strings.HasPrefix(ref, "http://"):
		path = strings.TrimPrefix(ref, "http://")
	case strings.HasPrefix(ref, "git@"):
		// git@github.com:org/repo -> github.com/org/repo
		rest := strings.TrimPrefix(ref, "git@")
		path = strings.Replace(rest, ":", "/", 1)
	default:
		path = ref
	}

	// Strip trailing .git
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Ref{}, fmt.Errorf("invalid repo reference %q: expected hosting/org/repo", ref)
	}
	return Ref{Hosting: parts[0], Org: parts[1], Repo: parts[2]}, nil
}

// LocalPath returns the absolute local filesystem path for this repo under reposDir.
func (r Ref) LocalPath(reposDir string) string {
	return filepath.Join(reposDir, r.Hosting, r.Org, r.Repo)
}

// OriginURL returns the canonical HTTPS origin URL for this repo.
func (r Ref) OriginURL() string {
	return fmt.Sprintf("https://%s/%s/%s", r.Hosting, r.Org, r.Repo)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cli/internal/reporef/... -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cli/internal/reporef/
git commit -m "feat: add reporef parsing and resolution"
```

---

## Chunk 2: Git Operations

### Task 4: Git runner interface and real implementation

**Files:**
- Create: `cli/internal/gitops/gitops.go`
- Create: `cli/internal/gitops/gitops_test.go`

- [ ] **Step 1: Write failing tests**

The tests use real git repos in temp dirs. They test `IsRepo`, `Clone`, `OriginURL`, and `CommitAndPush` / `Pull` against real git operations.

```go
// cli/internal/gitops/gitops_test.go
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./cli/internal/gitops/... -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement gitops.go**

```go
// Package gitops provides git operations used by Synchestra CLI commands.
package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner holds git operation implementations.
// Each field is a function so tests can substitute fakes.
type Runner struct {
	IsRepo        func(dir string) (bool, error)
	Clone         func(url, dir string) error
	OriginURL     func(dir string) (string, error)
	CommitAndPush func(dir string, files []string, msg string) error
	Push          func(dir string) error
	Pull          func(dir string) error
}

// NewRunner returns a Runner backed by real git operations.
func NewRunner() Runner {
	return Runner{
		IsRepo:        isRepo,
		Clone:         cloneRepo,
		OriginURL:     originURL,
		CommitAndPush: commitAndPush,
		Push:          push,
		Pull:          pull,
	}
}

func isRepo(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func cloneRepo(url, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	out, err := exec.Command("git", "clone", url, dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %q into %q: %w\n%s", url, dir, err, out)
	}
	return nil
}

func originURL(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func commitAndPush(dir string, files []string, msg string) error {
	addArgs := append([]string{"-C", dir, "add", "--"}, files...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}
	out, err := exec.Command("git", "-C", dir, "commit", "-m", msg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	out, err = exec.Command("git", "-C", dir, "push", "--set-upstream", "origin", "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %w\n%s", err, out)
	}
	return nil
}

func push(dir string) error {
	out, err := exec.Command("git", "-C", dir, "push", "--set-upstream", "origin", "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %w\n%s", err, out)
	}
	return nil
}

func pull(dir string) error {
	out, err := exec.Command("git", "-C", dir, "pull").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull: %w\n%s", err, out)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cli/internal/gitops/... -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cli/internal/gitops/
git commit -m "feat: add gitops.Runner with real git operations"
```

---

## Chunk 3: `project new` Command

### Task 5: `project` command group and `project new` logic

**Files:**
- Create: `cli/cmd/project/project.go`
- Create: `cli/cmd/project/new.go`
- Create: `cli/cmd/project/new_test.go`

- [ ] **Step 1: Write failing tests for `project new`**

Tests use a fake git runner backed by real temp dirs (pre-initialized as git repos). The command is tested end-to-end via its cobra `RunE`.

```go
// cli/cmd/project/new_test.go
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

	origOriginURL := env.runner.OriginURL
	env.runner.OriginURL = func(dir string) (string, error) {
		rel, _ := filepath.Rel(env.reposDir, dir)
		return "https://" + filepath.ToSlash(rel), nil
	}
	_ = origOriginURL

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
	callCount := 0
	env.runner.CommitAndPush = func(dir string, files []string, msg string) error {
		callCount++
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

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./cli/cmd/project/... -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Create `project.go`**

```go
// Package project implements the `synchestra project` command group.
package project

import "github.com/spf13/cobra"

// GroupCommand returns the `project` command group.
func GroupCommand(homeDir func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create and manage Synchestra projects",
		Long: `Commands for creating and managing Synchestra projects — setting up spec,
state, and target repositories, viewing project configuration, and
modifying project settings.`,
	}
	return cmd
}
```

- [ ] **Step 4: Create `new.go`**

```go
package project

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
	"github.com/synchesta-io/synchestra/cli/internal/globalconfig"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
	"github.com/synchesta-io/synchestra/cli/internal/reporef"
	"gopkg.in/yaml.v3"
)

// NewCommand returns the `project new` cobra command.
func NewCommand(homeDir func() (string, error), git gitops.Runner) *cobra.Command {
	var (
		specRepoRef   string
		stateRepoRef  string
		targetRepRefs []string
		title         string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Synchestra project",
		Long: `Creates a new Synchestra project by linking a spec repo, state repo, and
one or more target repos. Resolves all repo references, clones any that
are not already on disk, writes config files to each, commits and pushes.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProjectNew(cmd, homeDir, git, specRepoRef, stateRepoRef, targetRepRefs, title)
		},
	}

	cmd.Flags().StringVar(&specRepoRef, "spec-repo", "", "Spec repository reference (required)")
	cmd.Flags().StringVar(&stateRepoRef, "state-repo", "", "State repository reference (required)")
	cmd.Flags().StringArrayVar(&targetRepRefs, "target-repo", nil, "Target repository reference (repeatable, at least one required)")
	cmd.Flags().StringVar(&title, "title", "", "Project title (derived from README.md or repo name if omitted)")

	_ = cmd.MarkFlagRequired("spec-repo")
	_ = cmd.MarkFlagRequired("state-repo")

	return cmd
}

func runProjectNew(
	cmd *cobra.Command,
	homeDirFn func() (string, error),
	git gitops.Runner,
	specRepoRef, stateRepoRef string,
	targetRepoRefs []string,
	title string,
) error {
	if len(targetRepoRefs) == 0 {
		return exitcode.New(2, "--target-repo is required (at least one)")
	}

	homeDir, err := homeDirFn()
	if err != nil {
		return exitcode.New(10, "get home directory: %v", err)
	}

	cfg, err := globalconfig.Load(homeDir)
	if err != nil {
		return exitcode.New(10, "load global config: %v", err)
	}

	// Parse all repo references
	specRef, err := reporef.Parse(specRepoRef)
	if err != nil {
		return exitcode.New(2, "invalid --spec-repo: %v", err)
	}
	stateRef, err := reporef.Parse(stateRepoRef)
	if err != nil {
		return exitcode.New(2, "invalid --state-repo: %v", err)
	}

	targetRefs := make([]reporef.Ref, len(targetRepoRefs))
	for i, tr := range targetRepoRefs {
		targetRefs[i], err = reporef.Parse(tr)
		if err != nil {
			return exitcode.New(2, "invalid --target-repo %q: %v", tr, err)
		}
	}

	// Collect all refs: spec, state, targets
	allRefs := append([]reporef.Ref{specRef, stateRef}, targetRefs...)

	// Resolve local paths and clone if needed
	localPaths := make(map[string]string) // originURL -> localPath
	for _, ref := range allRefs {
		local := ref.LocalPath(cfg.ReposDir)
		origin := ref.OriginURL()

		exists, err := git.IsRepo(local)
		if err != nil {
			return exitcode.New(10, "check repo %s: %v", local, err)
		}
		if !exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "cloning %s...\n", origin)
			if err := git.Clone(origin, local); err != nil {
				return exitcode.New(3, "clone %s: %v", origin, err)
			}
			isRepo, err := git.IsRepo(local)
			if err != nil || !isRepo {
				return exitcode.New(3, "clone of %s did not produce a git repo", origin)
			}
		}
		localPaths[origin] = local
	}

	specLocal := specRef.LocalPath(cfg.ReposDir)
	stateLocal := stateRef.LocalPath(cfg.ReposDir)

	// Check for conflicts: existing config files pointing to a different project
	specOrigin := specRef.OriginURL()
	if err := checkNoConflict(stateLocal, "synchestra-state.yaml", "spec_repo", specOrigin); err != nil {
		return err
	}
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		if err := checkNoConflict(targetLocal, "synchestra-target.yaml", "spec_repo", specOrigin); err != nil {
			return err
		}
	}

	// Derive title
	if title == "" {
		title = deriveTitle(specLocal, specRef.Repo)
	}

	// Get origin URLs from git remote (authoritative)
	stateOrigin, err := git.OriginURL(stateLocal)
	if err != nil {
		return exitcode.New(10, "get origin URL for state repo: %v", err)
	}

	targetOrigins := make([]string, len(targetRefs))
	for i, tr := range targetRefs {
		targetOrigins[i], err = git.OriginURL(tr.LocalPath(cfg.ReposDir))
		if err != nil {
			return exitcode.New(10, "get origin URL for target repo %s: %v", tr.Repo, err)
		}
	}

	// Write synchestra-spec.yaml
	specConfig := map[string]any{
		"title":      title,
		"state_repo": stateOrigin,
		"repos":      targetOrigins,
	}
	if err := writeYAML(filepath.Join(specLocal, "synchestra-spec.yaml"), specConfig); err != nil {
		return exitcode.New(10, "write synchestra-spec.yaml: %v", err)
	}

	// Write synchestra-state.yaml
	stateConfig := map[string]any{"spec_repo": specOrigin}
	if err := writeYAML(filepath.Join(stateLocal, "synchestra-state.yaml"), stateConfig); err != nil {
		return exitcode.New(10, "write synchestra-state.yaml: %v", err)
	}

	// Write synchestra-target.yaml to each target
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		targetConfig := map[string]any{"spec_repo": specOrigin}
		if err := writeYAML(filepath.Join(targetLocal, "synchestra-target.yaml"), targetConfig); err != nil {
			return exitcode.New(10, "write synchestra-target.yaml to %s: %v", tr.Repo, err)
		}
	}

	// Commit and push all repos (with pull-retry on push conflict)
	commitMsg := fmt.Sprintf("chore: initialize Synchestra project %q", title)

	if err := commitPushWithRetry(git, specLocal, []string{"synchestra-spec.yaml"}, commitMsg, func() error {
		return checkNoConflict(stateLocal, "synchestra-state.yaml", "spec_repo", specOrigin)
	}); err != nil {
		return exitcode.New(10, "commit spec repo: %v", err)
	}
	if err := commitPushWithRetry(git, stateLocal, []string{"synchestra-state.yaml"}, commitMsg, func() error {
		return checkNoConflict(stateLocal, "synchestra-state.yaml", "spec_repo", specOrigin)
	}); err != nil {
		return exitcode.New(10, "commit state repo: %v", err)
	}
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		if err := commitPushWithRetry(git, targetLocal, []string{"synchestra-target.yaml"}, commitMsg, func() error {
			return checkNoConflict(targetLocal, "synchestra-target.yaml", "spec_repo", specOrigin)
		}); err != nil {
			return exitcode.New(10, "commit target repo %s: %v", tr.Repo, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Project %q created.\n", title)
	return nil
}

// commitPushWithRetry runs CommitAndPush; on push failure it pulls, re-checks for
// conflicts, then retries with Push only. This implements the spec's "on push conflict:
// pull, re-check, retry or fail" requirement.
func commitPushWithRetry(git gitops.Runner, dir string, files []string, msg string, conflictCheck func() error) error {
	if err := git.CommitAndPush(dir, files, msg); err == nil {
		return nil
	}
	// Push failed — pull to get remote changes
	if pullErr := git.Pull(dir); pullErr != nil {
		return fmt.Errorf("pull after push failure: %w", pullErr)
	}
	// Re-check: if a concurrent writer set a conflicting config, return exit code 1
	if checkErr := conflictCheck(); checkErr != nil {
		return checkErr
	}
	// Retry push (commit already recorded locally)
	return git.Push(dir)
}

// checkNoConflict returns an exitcode.Error (code 1) if the given config file
// exists and its field does not equal expectedValue.
func checkNoConflict(dir, filename, field, expectedValue string) error {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return exitcode.New(10, "read %s: %v", path, err)
	}
	var m map[string]string
	if err := yaml.Unmarshal(data, &m); err != nil {
		return exitcode.New(10, "parse %s: %v", path, err)
	}
	existing, ok := m[field]
	if ok && existing != expectedValue {
		return exitcode.New(1, "%s already configured for a different project (%s: %q)", filename, field, existing)
	}
	return nil
}

// deriveTitle extracts the first `# Heading` from README.md, or falls back to repoName.
func deriveTitle(repoDir, repoName string) string {
	data, err := os.ReadFile(filepath.Join(repoDir, "README.md"))
	if err != nil {
		return repoName
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return repoName
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./cli/cmd/project/... -v
```
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cli/cmd/project/
git commit -m "feat: implement project new command"
```

---

## Chunk 4: Wire Into CLI and Update main.go

### Task 6: Register `project` command in `cli/main.go` and update exit code handling

**Files:**
- Modify: `cli/main.go`
- Modify: `main.go`

- [ ] **Step 1: Update `cli/main.go`**

Add `exit func(int)` parameter and register `project` subcommand. Handle `exitcode.Error` returned from commands.

```go
package cli

import (
	"context"
	"errors"
	"os"

	"charm.land/fang/v2"
	"github.com/ingitdb/ingitdb-cli/cmd/ingitdb/commands"
	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/cmd/project"
	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Run(
	args []string,
	osUserHomeDir func() (string, error),
	osGetwd func() (string, error),
	fatal func(error),
	logf func(...any),
	exit func(int),
) {
	_ = osGetwd
	_ = logf
	rootCmd := &cobra.Command{
		Use:           "synchestra",
		Short:         "Synchestra CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return errors.New("not implemented yet")
		},
	}
	rootCmd.Flags().String("path", "", "path to the database directory (default: current directory)")
	rootCmd.SetErr(os.Stderr)

	projectGroup := project.GroupCommand(osUserHomeDir)
	projectGroup.AddCommand(
		project.NewCommand(osUserHomeDir, gitops.NewRunner()),
	)

	rootCmd.AddCommand(
		commands.Version(version, commit, date),
		commands.Pull(),
		commands.Setup(),
		commands.Resolve(),
		commands.Watch(),
		commands.Find(),
		commands.Migrate(),
		projectGroup,
	)

	rootCmd.SetArgs(args[1:])
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		var exitErr *exitcode.Error
		if errors.As(err, &exitErr) {
			fatal(exitErr.Err)
			exit(exitErr.Code)
			return
		}
		fatal(err)
		exit(1)
	}
}
```

- [ ] **Step 2: Update `main.go`**

`fatal` no longer calls `exit(1)` itself — `Run` now owns all exit calls. Pass `os.Exit` explicitly:

```go
package main

import (
	"fmt"
	"os"

	"github.com/synchesta-io/synchestra/cli"
)

var (
	exit = os.Exit
)

func main() {
	fatal := func(err error) {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	logf := func(args ...any) {
		_, _ = fmt.Fprintln(os.Stderr, args...)
	}
	cli.Run(os.Args, os.UserHomeDir, os.Getwd, fatal, logf, exit)
}
```

- [ ] **Step 3: Build to verify compilation**

```bash
go build ./...
```
Expected: no errors

- [ ] **Step 4: Smoke test**

```bash
go run . project new --help
```
Expected: prints usage for `project new` with `--spec-repo`, `--state-repo`, `--target-repo`, `--title` flags

- [ ] **Step 5: Run all tests**

```bash
go test ./...
```
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cli/main.go main.go
git commit -m "feat: wire project new into CLI root; propagate exit codes"
```

---

## Final Verification

- [ ] Run full test suite: `go test ./... -v`
- [ ] Build: `go build ./...`
- [ ] Smoke: `go run . project --help` and `go run . project new --help`
- [ ] Check all spec requirements are covered:
  - [x] Reads `~/.synchestra.yaml` for `repos_dir`
  - [x] Resolves refs to `{repos_dir}/{hosting}/{org}/{repo}`
  - [x] Clones missing repos; exit 3 on failure
  - [x] Validates git repos
  - [x] Conflict detection; exit 1
  - [x] Title derivation: flag > README > repo name
  - [x] Writes `synchestra-spec.yaml`, `synchestra-state.yaml`, `synchestra-target.yaml`
  - [x] Commits and pushes all repos
  - [x] Push conflict: pull, re-check, retry or fail
  - [x] Exit codes: 0 success, 1 conflict, 2 invalid args, 3 repo not found, 10+ unexpected
