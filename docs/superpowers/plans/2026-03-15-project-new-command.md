# `synchestra project new` — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `synchestra project new` CLI command that creates a Synchestra project by linking spec, state, and code repos — resolving references, cloning missing repos, writing config YAML files, and committing/pushing changes.

**Architecture:** Three focused packages under `cli/`: `globalconfig` (reads `~/.synchestra.yaml`), `reporef` (parses and resolves repo references), and `project` (Cobra command group with `new` subcommand). Each package has clear boundaries and is independently testable. Git operations use `os/exec` calling the `git` binary directly.

**Tech Stack:** Go 1.26, Cobra CLI, `gopkg.in/yaml.v3`, `os/exec` for git

**Spec:** [`spec/features/cli/project/new/README.md`](../../spec/features/cli/project/new/README.md)

---

## Design Decisions

- **`synchestra-project.yaml` renamed to `synchestra-spec.yaml`** — each repo type gets a distinctly named config file. This plan includes updating all spec references.
- **Git operations via `os/exec`** — keeps it simple, avoids adding a git library dependency. The `ingitdb-cli` dependency provides different abstractions not suited to basic clone/commit/push.
- **No `internal/` directory** — packages live directly under `cli/` to stay close to the command wiring. Can be restructured later when more commands exist.
- **Config YAML types use plain value fields** — all fields are required strings or string slices; no need for pointer-based optionality.
- **Repo URLs are normalized to HTTPS** — regardless of whether the user provides an SSH URL, short path, or HTTPS URL, config files always store `https://{hosting}/{org}/{repo}`. This matches the spec's "Values are stored as origin URLs" and provides a canonical form for conflict comparison.

---

## File Structure

```
cli/
  main.go                          # Modify: wire up project command group
  globalconfig/
    globalconfig.go                # Read ~/.synchestra.yaml, apply defaults, expand ~
    globalconfig_test.go
  reporef/
    reporef.go                    # Parse repo references (URL/short), resolve to disk path, get origin URL
    reporef_test.go
  project/
    project.go                    # Cobra "project" command group
    new.go                        # "project new" command implementation
    new_test.go
    configfiles.go                # YAML struct types + write functions for spec/state/code configs
    configfiles_test.go
  gitops/
    gitops.go                     # Git operations: clone, is-git-repo, get-origin-url, commit-and-push
    gitops_test.go
```

---

## Chunk 1: Rename `synchestra-project.yaml` → `synchestra-spec.yaml` in Specs

### Task 1: Update all spec references from `synchestra-project.yaml` to `synchestra-spec.yaml`

**Files (modify all 21 files that reference `synchestra-project.yaml`):**
- `spec/features/project-definition/README.md`
- `spec/features/README.md`
- `spec/architecture/README.md`
- `spec/architecture/repository-types.md`
- `spec/features/cli/_args/path.md`
- `spec/features/cli/_args/project.md`
- `spec/features/cli/_args/README.md`
- `spec/features/cli/serve/README.md`
- `spec/features/cli/mcp/README.md`
- `spec/features/cli/server/project/add/_args/spec.md`
- `spec/features/cli/server/project/add/README.md`
- `spec/features/development-plan/README.md`
- `spec/features/ui/README.md`
- `spec/features/chat/README.md`
- `spec/features/chat/workflow/README.md`
- `spec/plans/chat-feature/README.md`
- `spec/plans/chat-workflow-engine/README.md`
- `docs/superpowers/specs/2026-03-13-serve-server-mcp-design.md`
- `docs/superpowers/plans/2026-03-13-serve-server-mcp.md`
- `README.md`
- `.github/copilot-instructions.md`

- [ ] **Step 1: Replace all occurrences of `synchestra-project.yaml` with `synchestra-spec.yaml`**

In every file listed above, replace the string `synchestra-project.yaml` with `synchestra-spec.yaml`. No other content changes.

- [ ] **Step 2: Verify no remaining references**

Run: `grep -r "synchestra-project\.yaml" --include="*.md" .`
Expected: No matches

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "spec: rename synchestra-project.yaml to synchestra-spec.yaml

The spec repo config file is now synchestra-spec.yaml, matching the
naming pattern of synchestra-state.yaml and synchestra-code.yaml."
```

---

## Chunk 2: Foundation — `globalconfig` Package

### Task 2: Implement `globalconfig` package

**Files:**
- Create: `cli/globalconfig/globalconfig.go`
- Create: `cli/globalconfig/globalconfig_test.go`

- [ ] **Step 1: Write failing tests for globalconfig**

Create `cli/globalconfig/globalconfig_test.go`:

```go
package globalconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FileNotExists(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), ".synchestra.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "" {
		t.Errorf("expected empty ReposDir, got %q", cfg.ReposDir)
	}
}

func TestLoad_WithReposDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".synchestra.yaml")
	if err := os.WriteFile(cfgPath, []byte("repos_dir: /custom/repos\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "/custom/repos" {
		t.Errorf("expected /custom/repos, got %q", cfg.ReposDir)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".synchestra.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ReposDir != "" {
		t.Errorf("expected empty ReposDir, got %q", cfg.ReposDir)
	}
}

func TestResolveReposDir_Default(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("", homeDir)
	want := filepath.Join(homeDir, "synchestra", "repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveReposDir_TildeExpansion(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("~/my-repos", homeDir)
	want := filepath.Join(homeDir, "my-repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveReposDir_AbsolutePath(t *testing.T) {
	got := ResolveReposDir("/opt/repos", "/home/testuser")
	if got != "/opt/repos" {
		t.Errorf("got %q, want /opt/repos", got)
	}
}

func TestResolveReposDir_RelativePath(t *testing.T) {
	homeDir := "/home/testuser"
	got := ResolveReposDir("repos", homeDir)
	want := filepath.Join(homeDir, "repos")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/globalconfig/`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement globalconfig**

Create `cli/globalconfig/globalconfig.go`:

```go
package globalconfig

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents the user-level Synchestra configuration
// read from ~/.synchestra.yaml.
type GlobalConfig struct {
	ReposDir string `yaml:"repos_dir"`
}

// Load reads the global config from the given path.
// Returns a zero-value config (no error) if the file does not exist.
func Load(path string) (GlobalConfig, error) {
	var cfg GlobalConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// ResolveReposDir returns the effective repos directory.
// If reposDir is empty, returns {homeDir}/synchestra/repos.
// Expands ~ prefix and resolves relative paths against homeDir.
func ResolveReposDir(reposDir, homeDir string) string {
	if reposDir == "" {
		return filepath.Join(homeDir, "synchestra", "repos")
	}
	if strings.HasPrefix(reposDir, "~/") {
		return filepath.Join(homeDir, reposDir[2:])
	}
	if reposDir == "~" {
		return homeDir
	}
	if filepath.IsAbs(reposDir) {
		return reposDir
	}
	return filepath.Join(homeDir, reposDir)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/globalconfig/ -v`
Expected: PASS (all 7 tests)

- [ ] **Step 5: Commit**

```bash
git add cli/globalconfig/globalconfig.go cli/globalconfig/globalconfig_test.go
git commit -m "feat: add globalconfig package for ~/.synchestra.yaml"
```

---

## Chunk 3: Foundation — `reporef` Package

### Task 3: Implement `reporef` package

**Files:**
- Create: `cli/reporef/reporef.go`
- Create: `cli/reporef/reporef_test.go`

- [ ] **Step 1: Write failing tests for reporef**

Create `cli/reporef/reporef_test.go`:

```go
package reporef

import (
	"testing"
)

func TestParse_ShortPath(t *testing.T) {
	ref, err := Parse("github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_HTTPSURL(t *testing.T) {
	ref, err := Parse("https://github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_SSHURL(t *testing.T) {
	ref, err := Parse("git@github.com:acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_TrailingDotGit(t *testing.T) {
	ref, err := Parse("https://github.com/acme/acme-api.git")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api (should strip .git)", ref.Repo)
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := []string{
		"",
		"github.com",
		"github.com/acme",
		"just-a-name",
		"github.com/acme/api/extra/parts",
	}
	for _, input := range cases {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) should fail", input)
		}
	}
}

func TestRef_OriginURL(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.OriginURL()
	want := "https://github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_DiskPath(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.DiskPath("/home/user/synchestra/repos")
	want := "/home/user/synchestra/repos/github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_Identifier(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.Identifier()
	want := "github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/reporef/`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement reporef**

Create `cli/reporef/reporef.go`:

```go
package reporef

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Ref is a parsed repository reference.
type Ref struct {
	Hosting string // e.g., "github.com"
	Org     string // e.g., "acme"
	Repo    string // e.g., "acme-api"
}

// Parse parses a repo reference string into a Ref.
// Accepts:
//   - Short path: github.com/org/repo
//   - HTTPS URL: https://github.com/org/repo
//   - SSH URL: git@github.com:org/repo
func Parse(s string) (Ref, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Ref{}, fmt.Errorf("empty repo reference")
	}

	var hosting, org, repo string

	switch {
	case strings.HasPrefix(s, "git@"):
		// git@github.com:org/repo
		rest := strings.TrimPrefix(s, "git@")
		colonIdx := strings.Index(rest, ":")
		if colonIdx < 0 {
			return Ref{}, fmt.Errorf("invalid SSH repo reference: %q", s)
		}
		hosting = rest[:colonIdx]
		path := rest[colonIdx+1:]
		org, repo = splitOrgRepo(path)

	case strings.Contains(s, "://"):
		// https://github.com/org/repo
		u, err := url.Parse(s)
		if err != nil {
			return Ref{}, fmt.Errorf("invalid repo URL: %q: %w", s, err)
		}
		hosting = u.Host
		org, repo = splitOrgRepo(strings.TrimPrefix(u.Path, "/"))

	default:
		// github.com/org/repo
		parts := strings.Split(s, "/")
		if len(parts) != 3 {
			return Ref{}, fmt.Errorf("invalid repo reference %q: expected hosting/org/repo", s)
		}
		hosting, org, repo = parts[0], parts[1], parts[2]
	}

	repo = strings.TrimSuffix(repo, ".git")

	if hosting == "" || org == "" || repo == "" {
		return Ref{}, fmt.Errorf("invalid repo reference %q: missing hosting, org, or repo", s)
	}
	if strings.Contains(repo, "/") {
		return Ref{}, fmt.Errorf("invalid repo reference %q: too many path segments", s)
	}

	return Ref{Hosting: hosting, Org: org, Repo: repo}, nil
}

func splitOrgRepo(path string) (string, string) {
	path = strings.TrimSuffix(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return "", ""
	}
	if len(parts) > 2 {
		return "", "" // too many segments
	}
	return parts[0], parts[1]
}

// OriginURL returns the HTTPS URL for this repo.
func (r Ref) OriginURL() string {
	return "https://" + r.Hosting + "/" + r.Org + "/" + r.Repo
}

// DiskPath returns the local filesystem path for this repo under reposDir.
func (r Ref) DiskPath(reposDir string) string {
	return filepath.Join(reposDir, r.Hosting, r.Org, r.Repo)
}

// Identifier returns the short-form identifier: hosting/org/repo.
func (r Ref) Identifier() string {
	return r.Hosting + "/" + r.Org + "/" + r.Repo
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/reporef/ -v`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add cli/reporef/reporef.go cli/reporef/reporef_test.go
git commit -m "feat: add reporef package for repo reference parsing"
```

---

## Chunk 4: Git Operations — `gitops` Package

### Task 4: Implement `gitops` package

**Files:**
- Create: `cli/gitops/gitops.go`
- Create: `cli/gitops/gitops_test.go`

- [ ] **Step 1: Write failing tests for gitops**

Create `cli/gitops/gitops_test.go`:

```go
package gitops

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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/gitops/`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement gitops**

Create `cli/gitops/gitops.go`:

```go
package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsGitRepo returns true if dir is a git repository.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetOriginURL returns the URL of the "origin" remote.
func GetOriginURL(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting origin URL for %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Clone clones a git repository from url to dest.
func Clone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloning %s to %s: %w", url, dest, err)
	}
	return nil
}

// CommitAndPush stages the given files, commits with the message, and pushes.
// On push conflict, it pulls, re-stages, and retries once.
func CommitAndPush(dir string, files []string, message string) error {
	// Stage files
	args := []string{"-C", dir, "add"}
	args = append(args, files...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add in %s: %w\n%s", dir, err, out)
	}

	// Commit
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit in %s: %w\n%s", dir, err, out)
	}

	// Push with retry on conflict
	cmd = exec.Command("git", "-C", dir, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Pull and retry once
		if pullErr := Pull(dir); pullErr != nil {
			return fmt.Errorf("git push failed and pull also failed in %s: push: %w\n%s\npull: %v", dir, err, out, pullErr)
		}
		cmd = exec.Command("git", "-C", dir, "push")
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git push in %s failed after retry: %w\n%s", dir, err2, out2)
		}
	}

	return nil
}

// Pull performs a git pull in the given directory.
func Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull in %s: %w\n%s", dir, err, out)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/gitops/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cli/gitops/gitops.go cli/gitops/gitops_test.go
git commit -m "feat: add gitops package for git clone/commit/push operations"
```

---

## Chunk 5: Config File Types — `project/configfiles`

### Task 5: Implement config file YAML types and writers

**Files:**
- Create: `cli/project/configfiles.go`
- Create: `cli/project/configfiles_test.go`

- [ ] **Step 1: Write failing tests for config file types**

Create `cli/project/configfiles_test.go`:

```go
package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSpecConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := SpecConfig{
		Title:     "Acme Platform",
		StateRepo: "https://github.com/acme/acme-synchestra",
		Repos: []string{
			"https://github.com/acme/acme-api",
			"https://github.com/acme/acme-web",
		},
	}
	if err := WriteSpecConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-spec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"title: Acme Platform",
		"state_repo: https://github.com/acme/acme-synchestra",
		"- https://github.com/acme/acme-api",
		"- https://github.com/acme/acme-web",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("spec config missing %q\ngot:\n%s", want, content)
		}
	}
}

func TestWriteStateConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := StateConfig{
		SpecRepos: []string{"https://github.com/acme/acme-spec"},
	}
	if err := WriteStateConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-state.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "spec_repos:") {
		t.Errorf("state config missing spec_repos\ngot:\n%s", content)
	}
}

func TestWriteCodeConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := CodeConfig{
		SpecRepos: []string{"https://github.com/acme/acme-spec"},
	}
	if err := WriteCodeConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "synchestra-code.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "spec_repos:") {
		t.Errorf("code config missing spec_repos\ngot:\n%s", content)
	}
}

func TestReadSpecConfig_NotExists(t *testing.T) {
	_, err := ReadSpecConfig(t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadSpecConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "title: Test\nstate_repo: https://github.com/org/state\nrepos:\n  - https://github.com/org/code\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Test" {
		t.Errorf("Title = %q, want Test", cfg.Title)
	}
}

func TestReadStateConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "spec_repos:\n  - https://github.com/org/spec\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-state.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadStateConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SpecRepos) != 1 || cfg.SpecRepos[0] != "https://github.com/org/spec" {
		t.Errorf("SpecRepos = %v", cfg.SpecRepos)
	}
}

func TestReadCodeConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	content := "spec_repos:\n  - https://github.com/org/spec\n"
	if err := os.WriteFile(filepath.Join(dir, "synchestra-code.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadCodeConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SpecRepos) != 1 || cfg.SpecRepos[0] != "https://github.com/org/spec" {
		t.Errorf("SpecRepos = %v", cfg.SpecRepos)
	}
}

```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/project/`
Expected: FAIL

- [ ] **Step 3: Implement config file types and writers**

Create `cli/project/configfiles.go`:

```go
package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	SpecConfigFile   = "synchestra-spec.yaml"
	StateConfigFile  = "synchestra-state.yaml"
	CodeConfigFile = "synchestra-code.yaml"
)

// SpecConfig is the project definition written to the spec repo.
type SpecConfig struct {
	Title     string   `yaml:"title"`
	StateRepo string   `yaml:"state_repo"`
	Repos     []string `yaml:"repos"`
}

// StateConfig is the back-reference written to the state repo.
type StateConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
}

// CodeConfig is the pointer written to each code repo.
type CodeConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
}

// WriteSpecConfig writes synchestra-spec.yaml to the given directory.
func WriteSpecConfig(dir string, cfg SpecConfig) error {
	return writeYAML(filepath.Join(dir, SpecConfigFile), cfg)
}

// WriteStateConfig writes synchestra-state.yaml to the given directory.
func WriteStateConfig(dir string, cfg StateConfig) error {
	return writeYAML(filepath.Join(dir, StateConfigFile), cfg)
}

// WriteCodeConfig writes synchestra-code.yaml to the given directory.
func WriteCodeConfig(dir string, cfg CodeConfig) error {
	return writeYAML(filepath.Join(dir, CodeConfigFile), cfg)
}

// ReadSpecConfig reads synchestra-spec.yaml from the given directory.
func ReadSpecConfig(dir string) (SpecConfig, error) {
	var cfg SpecConfig
	data, err := os.ReadFile(filepath.Join(dir, SpecConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading spec config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing spec config: %w", err)
	}
	return cfg, nil
}

// ReadStateConfig reads synchestra-state.yaml from the given directory.
func ReadStateConfig(dir string) (StateConfig, error) {
	var cfg StateConfig
	data, err := os.ReadFile(filepath.Join(dir, StateConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading state config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing state config: %w", err)
	}
	return cfg, nil
}

// ReadCodeConfig reads synchestra-code.yaml from the given directory.
func ReadCodeConfig(dir string) (CodeConfig, error) {
	var cfg CodeConfig
	data, err := os.ReadFile(filepath.Join(dir, CodeConfigFile))
	if err != nil {
		return cfg, fmt.Errorf("reading code config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing code config: %w", err)
	}
	return cfg, nil
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/project/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cli/project/configfiles.go cli/project/configfiles_test.go
git commit -m "feat: add config file types and writers for spec/state/code repos"
```

---

## Chunk 6: Title Derivation

### Task 6: Implement title derivation logic

The title fallback chain: `--title` flag > first `# heading` in spec repo `README.md` > repo identifier.

**Files:**
- Create: `cli/project/title.go`
- Create: `cli/project/title_test.go`

- [ ] **Step 1: Write failing tests**

Create `cli/project/title_test.go`:

```go
package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveTitle_ExplicitFlag(t *testing.T) {
	got := DeriveTitle("My Project", t.TempDir(), "acme-spec")
	if got != "My Project" {
		t.Errorf("got %q, want My Project", got)
	}
}

func TestDeriveTitle_FromREADME(t *testing.T) {
	dir := t.TempDir()
	readme := "# Acme Platform\n\nSome description.\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatal(err)
	}
	got := DeriveTitle("", dir, "acme-spec")
	if got != "Acme Platform" {
		t.Errorf("got %q, want Acme Platform", got)
	}
}

func TestDeriveTitle_FromREADME_NoHeading(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("No heading here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := DeriveTitle("", dir, "acme-spec")
	if got != "acme-spec" {
		t.Errorf("got %q, want acme-spec", got)
	}
}

func TestDeriveTitle_NoREADME(t *testing.T) {
	got := DeriveTitle("", t.TempDir(), "acme-spec")
	if got != "acme-spec" {
		t.Errorf("got %q, want acme-spec", got)
	}
}

func TestExtractFirstHeading(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"# Hello World\n", "Hello World"},
		{"  # Trimmed  \n", "Trimmed"},
		{"## Not H1\n# Actual H1\n", "Actual H1"},
		{"no heading\n", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractFirstHeading([]byte(tc.input))
		if got != tc.want {
			t.Errorf("extractFirstHeading(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/project/ -run Title -v`
Expected: FAIL

- [ ] **Step 3: Implement title derivation**

Create `cli/project/title.go`:

```go
package project

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// DeriveTitle returns the project title using the fallback chain:
// 1. Explicit title (if non-empty)
// 2. First # heading in specRepoDir/README.md
// 3. repoIdentifier (e.g., "acme-spec")
func DeriveTitle(explicit, specRepoDir, repoIdentifier string) string {
	if explicit != "" {
		return explicit
	}
	data, err := os.ReadFile(filepath.Join(specRepoDir, "README.md"))
	if err == nil {
		if h := extractFirstHeading(data); h != "" {
			return h
		}
	}
	return repoIdentifier
}

// extractFirstHeading returns the text of the first # heading in markdown content.
func extractFirstHeading(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./cli/project/ -run Title -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cli/project/title.go cli/project/title_test.go
git commit -m "feat: add title derivation from flag, README heading, or repo name"
```

---

## Chunk 7: `project new` Command and Wiring

### Task 7: Implement the `project new` Cobra command

**Files:**
- Create: `cli/project/project.go`
- Create: `cli/project/new.go`
- Modify: `cli/main.go` — wire up project command group

- [ ] **Step 1: Create the project command group**

Create `cli/project/project.go`:

```go
package project

import (
	"github.com/spf13/cobra"
)

// Command returns the "project" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project creation and management",
	}
	cmd.AddCommand(newCommand())
	return cmd
}
```

- [ ] **Step 2: Implement `project new` command**

Create `cli/project/new.go`:

```go
package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/gitops"
	"github.com/synchesta-io/synchestra/cli/globalconfig"
	"github.com/synchesta-io/synchestra/cli/reporef"
	"gopkg.in/yaml.v3"
)

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Synchestra project",
		Long: `Creates a new Synchestra project by linking a spec repo, state repo, and
one or more code repos. Resolves all repo references, clones any that are
not already on disk, validates they are git repos, writes config files to
each, and commits and pushes the changes.`,
		RunE: runNew,
	}
	cmd.Flags().String("spec-repo", "", "spec repository reference (required)")
	cmd.Flags().String("state-repo", "", "state repository reference (required)")
	cmd.Flags().StringArray("code-repo", nil, "code repository reference (repeatable, at least one required)")
	cmd.Flags().String("title", "", "project title (default: derived from spec repo README)")
	_ = cmd.MarkFlagRequired("spec-repo")
	_ = cmd.MarkFlagRequired("state-repo")
	_ = cmd.MarkFlagRequired("code-repo")
	return cmd
}

func runNew(cmd *cobra.Command, _ []string) error {
	specRepoStr, _ := cmd.Flags().GetString("spec-repo")
	stateRepoStr, _ := cmd.Flags().GetString("state-repo")
	codeRepoStrs, _ := cmd.Flags().GetStringArray("code-repo")
	titleFlag, _ := cmd.Flags().GetString("title")

	if len(codeRepoStrs) == 0 {
		return &exitError{code: 2, msg: "at least one --code-repo is required"}
	}

	// Parse repo references
	specRef, err := reporef.Parse(specRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --spec-repo: %v", err)}
	}
	stateRef, err := reporef.Parse(stateRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --state-repo: %v", err)}
	}
	var codeRefs []reporef.Ref
	for _, s := range codeRepoStrs {
		ref, err := reporef.Parse(s)
		if err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("invalid --code-repo %q: %v", s, err)}
		}
		codeRefs = append(codeRefs, ref)
	}

	// Load global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("cannot determine home directory: %v", err)}
	}
	cfg, err := globalconfig.Load(filepath.Join(homeDir, ".synchestra.yaml"))
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("reading global config: %v", err)}
	}
	reposDir := globalconfig.ResolveReposDir(cfg.ReposDir, homeDir)

	// Resolve disk paths
	allRefs := append([]reporef.Ref{specRef, stateRef}, codeRefs...)
	allPaths := make([]string, len(allRefs))
	for i, ref := range allRefs {
		allPaths[i] = ref.DiskPath(reposDir)
	}
	specPath, statePath := allPaths[0], allPaths[1]
	codePaths := allPaths[2:]

	// Clone repos that don't exist on disk
	for i, ref := range allRefs {
		p := allPaths[i]
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Cloning %s...\n", ref.Identifier())
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("creating directory for %s: %v", ref.Identifier(), err)}
			}
			if err := gitops.Clone(ref.OriginURL(), p); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("cloning %s: %v", ref.Identifier(), err)}
			}
		}
	}

	// Validate all are git repos
	for i, ref := range allRefs {
		if !gitops.IsGitRepo(allPaths[i]) {
			return &exitError{code: 3, msg: fmt.Sprintf("%s is not a git repository", ref.Identifier())}
		}
	}

	// Check for existing config files pointing to a different project
	if err := checkSpecConflict(specPath, stateRef.OriginURL()); err != nil {
		return err
	}
	if err := checkBackrefConflict(statePath, StateConfigFile, specRef.OriginURL()); err != nil {
		return err
	}
	for _, cp := range codePaths {
		if err := checkBackrefConflict(cp, CodeConfigFile, specRef.OriginURL()); err != nil {
			return err
		}
	}

	// Derive title
	title := DeriveTitle(titleFlag, specPath, specRef.Repo)

	// Collect code origin URLs
	codeOriginURLs := make([]string, len(codeRefs))
	for i, ref := range codeRefs {
		codeOriginURLs[i] = ref.OriginURL()
	}

	// Write config files
	specCfg := SpecConfig{
		Title:     title,
		StateRepo: stateRef.OriginURL(),
		Repos:     codeOriginURLs,
	}
	if err := WriteSpecConfig(specPath, specCfg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("writing spec config: %v", err)}
	}

	stateCfg := StateConfig{SpecRepos: []string{specRef.OriginURL()}}
	if err := WriteStateConfig(statePath, stateCfg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("writing state config: %v", err)}
	}

	for _, cp := range codePaths {
		codeCfg := CodeConfig{SpecRepos: []string{specRef.OriginURL()}}
		if err := WriteCodeConfig(cp, codeCfg); err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("writing code config: %v", err)}
		}
	}

	// Commit and push
	commitMsg := fmt.Sprintf("synchestra: initialize project %q", title)

	if err := gitops.CommitAndPush(specPath, []string{SpecConfigFile}, commitMsg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("committing spec repo: %v", err)}
	}
	if err := gitops.CommitAndPush(statePath, []string{StateConfigFile}, commitMsg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("committing state repo: %v", err)}
	}
	for i, cp := range codePaths {
		if err := gitops.CommitAndPush(cp, []string{CodeConfigFile}, commitMsg); err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("committing code repo %s: %v", codeRefs[i].Identifier(), err)}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Project %q created successfully.\n", title)
	return nil
}

// checkSpecConflict checks if synchestra-spec.yaml exists and points to a
// different state repo (i.e., belongs to a different project).
func checkSpecConflict(dir, expectedStateRepo string) error {
	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return &exitError{code: 10, msg: fmt.Sprintf("reading existing spec config: %v", err)}
	}
	if cfg.StateRepo != "" && cfg.StateRepo != expectedStateRepo {
		return &exitError{
			code: 1,
			msg:  fmt.Sprintf("conflict: %s in %s already points to state repo %s", SpecConfigFile, dir, cfg.StateRepo),
		}
	}
	return nil
}

// checkBackrefConflict checks if a state or code config file exists and
// its spec_repo field points to a different spec repo.
func checkBackrefConflict(dir, filename, expectedSpecRepo string) error {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return &exitError{code: 10, msg: fmt.Sprintf("reading %s: %v", path, err)}
	}
	var backref struct {
		SpecRepo string `yaml:"spec_repo"`
	}
	if err := yaml.Unmarshal(data, &backref); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("parsing %s: %v", path, err)}
	}
	if backref.SpecRepo != "" && backref.SpecRepo != expectedSpecRepo {
		return &exitError{
			code: 1,
			msg:  fmt.Sprintf("conflict: %s in %s already points to spec repo %s", filename, dir, backref.SpecRepo),
		}
	}
	return nil
}

// exitError is an error that carries an exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// ExitCode returns the exit code for the error.
func (e *exitError) ExitCode() int { return e.code }
```

- [ ] **Step 3: Wire up the project command in cli/main.go**

Modify `cli/main.go` — add the project command to the root:

Add import: `"github.com/synchesta-io/synchestra/cli/project"`

Add to `rootCmd.AddCommand(...)`:
```go
project.Command(),
```

- [ ] **Step 4: Update root `main.go` to handle custom exit codes**

The root `main.go` (not `cli/main.go`) currently always exits with code 1. Commands need custom exit codes per spec (0, 1, 2, 3, 10+).

In root `main.go`, replace the `fatal` function:

```go
fatal := func(err error) {
    _, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
    type exitCoder interface{ ExitCode() int }
    if ec, ok := err.(exitCoder); ok {
        exit(ec.ExitCode())
        return
    }
    exit(1)
}
```

Note: `fang.Execute` wraps Cobra's `Execute()`. Verify that `fang.Execute` preserves the `RunE` error type through the call chain. If `fang` strips or re-wraps errors such that `errors.As` with `exitCoder` fails, the exit code check must be placed inside `cli.Run` using `errors.As` instead of a type assertion:

```go
if err := fang.Execute(context.Background(), rootCmd); err != nil {
    type exitCoder interface{ ExitCode() int }
    var ec exitCoder
    if errors.As(err, &ec) {
        fatal(err) // fatal will extract the code
    } else {
        fatal(err)
    }
}
```

The `errors.As` approach handles any level of wrapping, so the `fatal` function with `errors.As` (not a type assertion) is the safe choice:

```go
fatal := func(err error) {
    _, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
    type exitCoder interface{ ExitCode() int }
    var ec exitCoder
    if errors.As(err, &ec) {
        exit(ec.ExitCode())
        return
    }
    exit(1)
}
```

- [ ] **Step 5: Update go.mod — ensure `gopkg.in/yaml.v3` is direct**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go mod tidy`

- [ ] **Step 6: Build and verify compilation**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 7: Run all tests**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 8: Verify CLI recognizes the command**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go run . project new --help`
Expected: Shows usage with `--spec-repo`, `--state-repo`, `--code-repo`, `--title` flags

- [ ] **Step 9: Commit**

```bash
git add cli/project/project.go cli/project/new.go cli/main.go main.go go.mod go.sum
git commit -m "feat: implement synchestra project new command

Creates a project by resolving repo references, cloning missing repos,
writing synchestra-spec.yaml / synchestra-state.yaml / synchestra-code.yaml,
and committing + pushing changes to all repos."
```

---

## Chunk 8: Integration Test

### Task 8: Add integration test for `project new`

**Files:**
- Create: `cli/project/new_test.go`

- [ ] **Step 1: Write integration test using bare git repos**

Create `cli/project/new_test.go`:

```go
package project

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/gitops"
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
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	run("push", "origin", "HEAD")
}

func cloneAndConfigure(t *testing.T, bare, dest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Clone(bare, dest); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dest
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config: %v\n%s", err, out)
		}
	}
}

func TestRunNew_ViaCobra(t *testing.T) {
	// Create bare repos to act as remotes
	specBare := initBareTestRepo(t, "spec")
	stateBare := initBareTestRepo(t, "state")
	codeBare := initBareTestRepo(t, "code")

	// Seed each with an initial commit (spec gets a README with heading)
	seedBareRepo(t, specBare, "# My Test Project\n\nDescription.\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, codeBare, "# Code\n")

	// Set up repos_dir structure with clones
	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "test", "spec")
	stateDir := filepath.Join(reposDir, "local", "test", "state")
	codeDir := filepath.Join(reposDir, "local", "test", "code")

	cloneAndConfigure(t, specBare, specDir)
	cloneAndConfigure(t, stateBare, stateDir)
	cloneAndConfigure(t, codeBare, codeDir)

	// Write global config pointing to our reposDir
	homeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeDir, ".synchestra.yaml"),
		[]byte("repos_dir: "+reposDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Override os.UserHomeDir for the test by setting HOME
	t.Setenv("HOME", homeDir)

	// Execute the command via Cobra
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

	// Verify spec config
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

	// Verify state config
	stateCfg, err := ReadStateConfig(stateDir)
	if err != nil {
		t.Fatalf("reading state config: %v", err)
	}
	if len(stateCfg.SpecRepos) != 1 || stateCfg.SpecRepos[0] != "https://local/test/spec" {
		t.Errorf("state SpecRepos = %v", stateCfg.SpecRepos)
	}

	// Verify code config
	codeCfg, err := ReadCodeConfig(codeDir)
	if err != nil {
		t.Fatalf("reading code config: %v", err)
	}
	if len(codeCfg.SpecRepos) != 1 || codeCfg.SpecRepos[0] != "https://local/test/spec" {
		t.Errorf("code SpecRepos = %v", codeCfg.SpecRepos)
	}
}

func TestRunNew_MultipleCodeRepos(t *testing.T) {
	// Create bare repos
	specBare := initBareTestRepo(t, "spec2")
	stateBare := initBareTestRepo(t, "state2")
	code1Bare := initBareTestRepo(t, "code2a")
	code2Bare := initBareTestRepo(t, "code2b")

	seedBareRepo(t, specBare, "# Multi Code\n")
	seedBareRepo(t, stateBare, "# State\n")
	seedBareRepo(t, code1Bare, "# C1\n")
	seedBareRepo(t, code2Bare, "# C2\n")

	reposDir := filepath.Join(t.TempDir(), "repos")
	specDir := filepath.Join(reposDir, "local", "mt", "spec")
	stateDir := filepath.Join(reposDir, "local", "mt", "state")
	code1Dir := filepath.Join(reposDir, "local", "mt", "c1")
	code2Dir := filepath.Join(reposDir, "local", "mt", "c2")

	cloneAndConfigure(t, specBare, specDir)
	cloneAndConfigure(t, stateBare, stateDir)
	cloneAndConfigure(t, code1Bare, code1Dir)
	cloneAndConfigure(t, code2Bare, code2Dir)

	homeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(homeDir, ".synchestra.yaml"),
		[]byte("repos_dir: "+reposDir+"\n"), 0644); err != nil {
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
		"--code-repo", "local/mt/c1",
		"--code-repo", "local/mt/c2",
		"--title", "Multi Code Project",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Verify spec config has both code repos
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

	// Verify both code repos have config
	for _, cd := range []string{code1Dir, code2Dir} {
		cfg, err := ReadCodeConfig(cd)
		if err != nil {
			t.Fatalf("reading code config from %s: %v", cd, err)
		}
		if len(cfg.SpecRepos) != 1 || cfg.SpecRepos[0] != "https://local/mt/spec" {
			t.Errorf("code SpecRepos = %v", cfg.SpecRepos)
		}
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
	if err := WriteSpecConfig(dir, SpecConfig{
		Title: "Test", StateRepo: "https://example.com/state",
	}); err != nil {
		t.Fatal(err)
	}
	err := checkSpecConflict(dir, "https://example.com/state")
	if err != nil {
		t.Errorf("same project should not conflict, got %v", err)
	}
}

func TestCheckSpecConflict_DifferentProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSpecConfig(dir, SpecConfig{
		Title: "Other", StateRepo: "https://example.com/other-state",
	}); err != nil {
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
	if err := WriteStateConfig(dir, StateConfig{SpecRepos: []string{"https://example.com/spec"}}); err != nil {
		t.Fatal(err)
	}
	err := checkBackrefConflict(dir, StateConfigFile, "https://example.com/spec")
	if err != nil {
		t.Errorf("same project should not conflict, got %v", err)
	}
}

func TestCheckBackrefConflict_DifferentProject(t *testing.T) {
	dir := t.TempDir()
	if err := WriteStateConfig(dir, StateConfig{SpecRepos: []string{"https://example.com/other-spec"}}); err != nil {
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
```

- [ ] **Step 2: Run all tests**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/.claude/worktrees/config-opus && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add cli/project/new_test.go
git commit -m "test: add integration tests for project new command"
```
