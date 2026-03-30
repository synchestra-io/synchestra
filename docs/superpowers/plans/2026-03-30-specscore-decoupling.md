# SpecScore Decoupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract spec-format-aware Go code from synchestra into specscore as reusable library packages with a standalone CLI, then rewire synchestra to import specscore.

**Architecture:** specscore gets `pkg/` (public library API — exitcode, feature, lint, sourceref, projectdef), `internal/cli/` (thin cobra wrappers), and `cmd/specscore/main.go`. synchestra imports `github.com/synchestra-io/specscore/pkg/*` and keeps only thin CLI wrappers + orchestration code.

**Tech Stack:** Go 1.26+, cobra, gopkg.in/yaml.v3, goreleaser, GitHub Actions (strongo/go-ci-action)

**Repos:**
- specscore: `/Users/alexandertrakhimenok/projects/synchestra-io/specscore`
- synchestra: `/Users/alexandertrakhimenok/projects/synchestra-io/synchestra`

---

## File Structure

### New files in specscore

```
specscore/
├── go.mod
├── go.sum
├── AGENTS.md
├── CLAUDE.md
├── .goreleaser.yml
├── .github/
│   └── workflows/
│       ├── go-ci.yml
│       └── release.yml
├── cmd/
│   └── specscore/
│       └── main.go
├── internal/
│   └── cli/
│       ├── root.go
│       ├── feature.go
│       ├── spec.go
│       └── code.go
├── pkg/
│   ├── exitcode/
│   │   └── exitcode.go
│   ├── sourceref/
│   │   ├── sourceref.go
│   │   └── scan.go
│   ├── feature/
│   │   ├── discover.go
│   │   ├── info.go
│   │   ├── deps.go
│   │   ├── tree.go
│   │   ├── fields.go
│   │   ├── slug.go
│   │   ├── template.go
│   │   ├── newfeature.go
│   │   └── transitive.go
│   ├── lint/
│   │   ├── lint.go
│   │   ├── linter.go
│   │   ├── checkers_extended.go
│   │   ├── readme_exists.go
│   │   ├── oq_section.go
│   │   ├── index_entries.go
│   │   ├── plan_hierarchy.go
│   │   └── plan_roi.go
│   └── projectdef/
│       └── projectdef.go
```

### Modified files in synchestra (Task T9)

```
synchestra/
├── go.mod                          # Add specscore dependency + replace directive
├── pkg/cli/main.go                 # Update imports
├── pkg/cli/feature/                # Rewrite as thin wrappers around specscore/pkg/feature
├── pkg/cli/spec/                   # Rewrite as thin wrapper around specscore/pkg/lint
├── pkg/cli/code/                   # Rewrite as thin wrapper around specscore/pkg/sourceref
├── pkg/cli/exitcode/               # Delete (imported from specscore)
├── pkg/sourceref/                  # Delete (imported from specscore)
├── pkg/cli/project/configfiles.go  # Update to import specscore/pkg/projectdef
```

---

## Task T1: Bootstrap specscore Go module

**Repo:** specscore
**Model:** Sonnet

**Files:**
- Create: `go.mod`
- Create: `cmd/specscore/main.go`
- Create: `internal/cli/root.go`
- Create: `pkg/exitcode/exitcode.go`
- Create: `AGENTS.md`
- Create: `CLAUDE.md`

- [ ] **Step 1: Initialize go.mod**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go mod init github.com/synchestra-io/specscore
```

Then edit `go.mod` to set Go version:

```
module github.com/synchestra-io/specscore

go 1.26.1
```

- [ ] **Step 2: Create pkg/exitcode/exitcode.go**

Create `pkg/exitcode/exitcode.go` with the following content (copied from synchestra, updated package doc):

```go
// Package exitcode defines the shared exit code constants and error type
// used by all SpecScore CLI commands and library consumers.
package exitcode

import "fmt"

// Standard exit codes shared by every CLI command.
const (
	Success      = 0  // Operation completed successfully.
	Conflict     = 1  // Concurrent-modification conflict.
	InvalidArgs  = 2  // Missing or invalid command arguments/flags.
	NotFound     = 3  // Requested resource does not exist.
	InvalidState = 4  // State transition is not allowed.
	Unexpected   = 10 // Catch-all for unexpected runtime errors.
)

// Error carries a machine-readable exit code alongside a human-readable
// message. It satisfies both the error interface and the ExitCode()
// convention checked by the top-level CLI runner.
type Error struct {
	code int
	msg  string
}

func (e *Error) Error() string { return e.msg }

// ExitCode returns the numeric exit code for this error.
func (e *Error) ExitCode() int { return e.code }

// New creates an Error with the given exit code and message.
func New(code int, msg string) *Error {
	return &Error{code: code, msg: msg}
}

// Newf creates an Error with the given exit code and formatted message.
func Newf(code int, format string, args ...any) *Error {
	return &Error{code: code, msg: fmt.Sprintf(format, args...)}
}

// --- Convenience constructors for standard exit codes ---

// ConflictError returns an exit-code-1 error.
func ConflictError(msg string) *Error { return &Error{code: Conflict, msg: msg} }

// ConflictErrorf returns an exit-code-1 error with a formatted message.
func ConflictErrorf(format string, args ...any) *Error {
	return Newf(Conflict, format, args...)
}

// InvalidArgsError returns an exit-code-2 error.
func InvalidArgsError(msg string) *Error { return &Error{code: InvalidArgs, msg: msg} }

// InvalidArgsErrorf returns an exit-code-2 error with a formatted message.
func InvalidArgsErrorf(format string, args ...any) *Error {
	return Newf(InvalidArgs, format, args...)
}

// NotFoundError returns an exit-code-3 error.
func NotFoundError(msg string) *Error { return &Error{code: NotFound, msg: msg} }

// NotFoundErrorf returns an exit-code-3 error with a formatted message.
func NotFoundErrorf(format string, args ...any) *Error {
	return Newf(NotFound, format, args...)
}

// InvalidStateError returns an exit-code-4 error.
func InvalidStateError(msg string) *Error { return &Error{code: InvalidState, msg: msg} }

// InvalidStateErrorf returns an exit-code-4 error with a formatted message.
func InvalidStateErrorf(format string, args ...any) *Error {
	return Newf(InvalidState, format, args...)
}

// UnexpectedError returns an exit-code-10 error.
func UnexpectedError(msg string) *Error { return &Error{code: Unexpected, msg: msg} }

// UnexpectedErrorf returns an exit-code-10 error with a formatted message.
func UnexpectedErrorf(format string, args ...any) *Error {
	return Newf(Unexpected, format, args...)
}
```

- [ ] **Step 3: Create internal/cli/root.go**

```go
package cli

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Run executes the specscore CLI with the given arguments.
func Run(args []string) error {
	rootCmd := &cobra.Command{
		Use:           "specscore",
		Short:         "SpecScore CLI — validate and query specification repositories",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	rootCmd.SetErr(os.Stderr)

	rootCmd.AddCommand(
		versionCommand(),
	)

	rootCmd.SetArgs(args[1:])
	return rootCmd.ExecuteContext(context.Background())
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the specscore version",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = cmd.OutOrStdout().Write([]byte("specscore " + version + " (" + commit + ") " + date + "\n"))
		},
	}
}

// Fatal prints the error and exits with the appropriate code.
func Fatal(err error) {
	if err == nil {
		return
	}
	_, _ = os.Stderr.WriteString(err.Error() + "\n")

	type exitCoder interface {
		ExitCode() int
	}
	var ec exitCoder
	if errors.As(err, &ec) {
		os.Exit(ec.ExitCode())
	}
	os.Exit(1)
}
```

- [ ] **Step 4: Create cmd/specscore/main.go**

```go
package main

import (
	"os"

	cli "github.com/synchestra-io/specscore/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		cli.Fatal(err)
	}
}
```

- [ ] **Step 5: Create AGENTS.md**

```markdown
# SpecScore — AI Agent Rules

## Build, test, and lint commands

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`

These are also run by the CI workflow in `.github/workflows/go-ci.yml`.

## High-level architecture

SpecScore is both the specification format definition and its reference tooling:

- `spec/` is the technical source of truth for the SpecScore format (features, acceptance criteria, development plans, project definition, source references).
- `pkg/` contains the Go library packages — the public API importable by other tools (synchestra, rehearse, etc.).
- `internal/cli/` contains thin cobra command wrappers around `pkg/` functions.
- `cmd/specscore/` is the CLI entry point.
- `docs/` contains user-facing explanations.

Key packages:

- `pkg/exitcode` — shared exit code constants and error type
- `pkg/feature` — feature discovery, traversal, metadata, dependency resolution
- `pkg/lint` — specification linting engine with pluggable rules
- `pkg/sourceref` — `specscore:` annotation parsing and source-to-spec linking
- `pkg/projectdef` — `specscore-project.yaml` schema and read/write

## Directory structure

- Every directory MUST have a `README.md` file.
- Every `README.md` MUST have an "Outstanding Questions" section. If there are none, it explicitly states "None at this time."
- Every `README.md` that has child directories MUST include a brief summary for each immediate child.

## Go error handling requirements

Every function call that returns an error must have its error value handled explicitly:

- Capture error returns: `result, err := someFunction()`
- Check or explicitly ignore errors:
  - **Check**: `if err != nil { return err }`
  - **Explicitly ignore**: `_, _ = someFunction()`
- Do not silently drop error returns

## Go validation after code changes

After any change to `.go` files, agents must run the full Go validation sequence:

- `gofmt -w <changed-go-files>`
- `golangci-lint run ./...`
- `go test ./...`
- `go build ./...`
- `go vet ./...`
```

- [ ] **Step 6: Create CLAUDE.md**

```markdown
# SpecScore — Project Conventions

## Instruction precedence

- Read [`AGENTS.md`](AGENTS.md) first and follow all of its instructions and rules.
- Treat `CLAUDE.md` as additional repository-specific guidance that complements `AGENTS.md`.
```

- [ ] **Step 7: Add cobra dependency and verify build**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go get github.com/spf13/cobra@v1.10.2
go mod tidy
go build ./...
go vet ./...
```

Expected: builds successfully with no errors.

- [ ] **Step 8: Commit**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
git add go.mod go.sum cmd/ internal/ pkg/ AGENTS.md CLAUDE.md
git commit -m "feat: bootstrap Go module with CLI skeleton and exitcode package"
```

---

## Task T2: Add CI/CD workflows

**Repo:** specscore
**Model:** Sonnet
**Depends on:** T1

**Files:**
- Create: `.github/workflows/go-ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `.github/workflows/README.md`
- Create: `.goreleaser.yml`

- [ ] **Step 1: Create .github/workflows/go-ci.yml**

```yaml
name: Go CI

on:
  push:
    paths:
      - '.github/workflows/go-ci.yml'
      - '**/*.go'
      - 'go.mod'
      - 'go.sum'
  pull_request:
    paths:
      - '.github/workflows/go-ci.yml'
      - '**/*.go'
      - 'go.mod'
      - 'go.sum'
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: go-ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:

  strongo_workflow:
    permissions:
      contents: write
    uses: strongo/go-ci-action/.github/workflows/workflow.yml@main

    secrets:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Create .github/workflows/release.yml**

```yaml
name: Release

on:
  push:
    branches:
      - main
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      tag:
        description: 'Tag to release (e.g. v0.1.0) — leave empty to build snapshot of current ref'
        required: false
        type: string

permissions:
  contents: write

concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: true

jobs:
  goreleaser:
    name: ${{ (startsWith(github.ref, 'refs/tags/') || inputs.tag != '') && 'Publish release' || 'Validate snapshot build' }}
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ inputs.tag || github.ref }}

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Run GoReleaser (snapshot — validate only)
        if: ${{ !startsWith(github.ref, 'refs/tags/') && inputs.tag == '' }}
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --snapshot --clean --skip=publish
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Run GoReleaser (full release)
        if: ${{ startsWith(github.ref, 'refs/tags/') || inputs.tag != '' }}
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Upload snapshot artifacts
        if: ${{ !startsWith(github.ref, 'refs/tags/') && inputs.tag == '' }}
        uses: actions/upload-artifact@v4
        with:
          name: snapshot-binaries
          path: |
            dist/*.tar.gz
            dist/*.zip
            dist/*_checksums.txt
          retention-days: 7
```

- [ ] **Step 3: Create .github/workflows/README.md**

```markdown
# GitHub Workflows

## Outstanding Questions

None at this time.
```

- [ ] **Step 4: Create .goreleaser.yml**

```yaml
version: 2

project_name: specscore

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: specscore
    main: ./cmd/specscore
    binary: specscore
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X github.com/synchestra-io/specscore/internal/cli.version={{.Version}}
      - -X github.com/synchestra-io/specscore/internal/cli.commit={{.Commit}}
      - -X github.com/synchestra-io/specscore/internal/cli.date={{.Date}}

archives:
  - id: specscore
    ids:
      - specscore
    name_template: "specscore_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        formats: [zip]
    formats: [tar.gz]

checksum:
  name_template: "specscore_{{ .Version }}_checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
      - Merge pull request
      - Merge branch
```

- [ ] **Step 5: Verify build still passes**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
git add .github/ .goreleaser.yml
git commit -m "ci: add Go CI and release workflows with goreleaser"
```

---

## Task R1: Review bootstrap + CI/CD

**Repo:** specscore
**Model:** Sonnet
**Depends on:** T2

**Review checklist:**

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `specscore version` prints version string
- [ ] `go-ci.yml` matches synchestra's pattern (strongo shared workflow)
- [ ] `release.yml` has snapshot + full release paths
- [ ] `.goreleaser.yml` ldflags inject into `internal/cli` (not `pkg/cli`)
- [ ] `AGENTS.md` covers build commands, architecture, directory conventions, error handling, validation
- [ ] `CLAUDE.md` points to AGENTS.md
- [ ] No synchestra-specific references leaked into specscore

---

## Task T3: Extract pkg/exitcode

**Repo:** specscore
**Model:** Haiku
**Depends on:** R1

`pkg/exitcode` was already created in T1. This task just verifies it and adds a test.

**Files:**
- Verify: `pkg/exitcode/exitcode.go` (already exists from T1)
- Create: `pkg/exitcode/exitcode_test.go`

- [ ] **Step 1: Write test for exitcode**

Create `pkg/exitcode/exitcode_test.go`:

```go
package exitcode

import (
	"testing"
)

func TestError(t *testing.T) {
	err := New(InvalidArgs, "bad input")
	if err.Error() != "bad input" {
		t.Errorf("expected 'bad input', got %q", err.Error())
	}
	if err.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", err.ExitCode())
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		code int
	}{
		{"ConflictError", ConflictError("c"), Conflict},
		{"InvalidArgsError", InvalidArgsError("i"), InvalidArgs},
		{"NotFoundError", NotFoundError("n"), NotFound},
		{"InvalidStateError", InvalidStateError("s"), InvalidState},
		{"UnexpectedError", UnexpectedError("u"), Unexpected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ExitCode() != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, tt.err.ExitCode())
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go test ./pkg/exitcode/... -v
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/exitcode/exitcode_test.go
git commit -m "test: add exitcode package tests"
```

---

## Task T4: Extract pkg/sourceref

**Repo:** specscore
**Model:** Sonnet
**Depends on:** R1

**Files:**
- Create: `pkg/sourceref/sourceref.go`
- Create: `pkg/sourceref/scan.go`
- Create: `pkg/sourceref/sourceref_test.go`

- [ ] **Step 1: Write failing test**

Create `pkg/sourceref/sourceref_test.go`:

```go
package sourceref

import (
	"testing"
)

func TestDetectReference(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"// synchestra:feature/cli/task", true},
		{"# synchestra:plan/chat-feature", true},
		{"// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli", true},
		{"no reference here", false},
		{"synchestra: not a comment", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := DetectReference(tt.line)
			if got != tt.want {
				t.Errorf("DetectReference(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseReference(t *testing.T) {
	tests := []struct {
		input    string
		wantPath string
		wantType string
	}{
		{"synchestra:feature/cli/task", "spec/features/cli/task", "feature"},
		{"synchestra:plan/chat-feature", "spec/plans/chat-feature", "plan"},
		{"synchestra:doc/api", "docs/api", "doc"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ref, err := ParseReference(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.ResolvedPath != tt.wantPath {
				t.Errorf("ResolvedPath = %q, want %q", ref.ResolvedPath, tt.wantPath)
			}
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
		})
	}
}

func TestScanLine(t *testing.T) {
	ref := ScanLine("// synchestra:feature/cli/task/claim")
	if ref == nil {
		t.Fatal("expected reference, got nil")
	}
	if ref.ResolvedPath != "spec/features/cli/task/claim" {
		t.Errorf("ResolvedPath = %q, want %q", ref.ResolvedPath, "spec/features/cli/task/claim")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go test ./pkg/sourceref/... -v
```

Expected: FAIL — files don't exist yet.

- [ ] **Step 3: Create pkg/sourceref/sourceref.go**

Copy from synchestra's `pkg/sourceref/sourceref.go` with updated package path. The file is self-contained (stdlib only). Copy it verbatim — no changes needed beyond the package declaration which is already `package sourceref`.

```go
package sourceref

import (
	"fmt"
	"regexp"
	"strings"
)

// Reference represents a parsed source reference found in source code.
type Reference struct {
	// ResolvedPath is the repo-root-relative path after type-prefix expansion
	ResolvedPath string
	// CrossRepoSuffix is the optional @host/org/repo (empty string if same-repo)
	CrossRepoSuffix string
	// Type is the inferred resource type: "feature", "plan", "doc", or "" if unknown
	Type string
}

// DetectionRegex matches source references preceded by recognized comment prefixes.
var DetectionRegex = regexp.MustCompile(`^\s*(//|#|--|/\*|\*|%|;)\s*(synchestra:|https://synchestra\.io/)`)

// DetectReference checks if a line contains a source reference.
func DetectReference(line string) bool {
	return DetectionRegex.MatchString(line)
}

// ExtractReference extracts the reference string from a line.
func ExtractReference(line string) string {
	idx := strings.Index(line, "synchestra:")
	if idx == -1 {
		idx = strings.Index(line, "https://synchestra.io/")
	}
	if idx == -1 {
		return ""
	}
	extracted := line[idx:]
	if strings.HasPrefix(extracted, "https://") {
		if endIdx := strings.IndexAny(extracted, " \t\n\r"); endIdx != -1 {
			extracted = extracted[:endIdx]
		}
	} else if strings.HasPrefix(extracted, "synchestra:") {
		if endIdx := strings.IndexAny(extracted, " \t\n\r"); endIdx != -1 {
			extracted = extracted[:endIdx]
		}
	}
	return extracted
}

// ParseReference parses an extracted reference string and returns a Reference.
func ParseReference(extracted string) (*Reference, error) {
	if extracted == "" {
		return nil, fmt.Errorf("empty reference")
	}
	if strings.HasPrefix(extracted, "https://synchestra.io/") {
		return parseExpandedURL(extracted)
	}
	if strings.HasPrefix(extracted, "synchestra:") {
		return parseShortNotation(extracted)
	}
	return nil, fmt.Errorf("unrecognized reference format: %s", extracted)
}

func parseExpandedURL(url string) (*Reference, error) {
	url = strings.TrimPrefix(url, "https://synchestra.io/")
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid expanded URL format: too few path segments")
	}
	host := parts[0]
	org := parts[1]
	repo := parts[2]
	resolvedPath := strings.Join(parts[3:], "/")
	currentHost, currentOrg, currentRepo := "github.com", "synchestra-io", "synchestra"
	crossRepoSuffix := ""
	if host != currentHost || org != currentOrg || repo != currentRepo {
		crossRepoSuffix = fmt.Sprintf("@%s/%s/%s", host, org, repo)
	}
	refType := inferType(resolvedPath)
	return &Reference{
		ResolvedPath:    resolvedPath,
		CrossRepoSuffix: crossRepoSuffix,
		Type:            refType,
	}, nil
}

func parseShortNotation(notation string) (*Reference, error) {
	notation = strings.TrimPrefix(notation, "synchestra:")
	crossRepoSuffix := ""
	reference := notation
	if idx := strings.LastIndex(notation, "@"); idx != -1 {
		crossRepoSuffix = notation[idx:]
		reference = notation[:idx]
	}
	resolvedPath, err := resolveReference(reference)
	if err != nil {
		return nil, err
	}
	refType := inferType(resolvedPath)
	return &Reference{
		ResolvedPath:    resolvedPath,
		CrossRepoSuffix: crossRepoSuffix,
		Type:            refType,
	}, nil
}

func resolveReference(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty reference")
	}
	if strings.HasPrefix(ref, "feature/") {
		return "spec/features/" + strings.TrimPrefix(ref, "feature/"), nil
	}
	if strings.HasPrefix(ref, "plan/") {
		return "spec/plans/" + strings.TrimPrefix(ref, "plan/"), nil
	}
	if strings.HasPrefix(ref, "doc/") {
		return "docs/" + strings.TrimPrefix(ref, "doc/"), nil
	}
	return ref, nil
}

func inferType(resolvedPath string) string {
	if strings.HasPrefix(resolvedPath, "spec/features/") {
		return "feature"
	}
	if strings.HasPrefix(resolvedPath, "spec/plans/") {
		return "plan"
	}
	if strings.HasPrefix(resolvedPath, "docs/") {
		return "doc"
	}
	return ""
}

// ScanLine scans a single line for references. Returns nil if none found.
func ScanLine(line string) *Reference {
	if !DetectReference(line) {
		return nil
	}
	extracted := ExtractReference(line)
	if extracted == "" {
		return nil
	}
	ref, err := ParseReference(extracted)
	if err != nil {
		return nil
	}
	return ref
}
```

- [ ] **Step 4: Create pkg/sourceref/scan.go**

Copy from synchestra's `pkg/sourceref/scan.go` verbatim — it only depends on stdlib.

```go
package sourceref

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ScanResult represents the references found in a set of files.
type ScanResult struct {
	FileRefs map[string][]*Reference
}

// ScanFiles scans a list of files for source references.
func ScanFiles(filePaths []string) (*ScanResult, error) {
	result := &ScanResult{
		FileRefs: make(map[string][]*Reference),
	}
	for _, filePath := range filePaths {
		refs, err := scanFile(filePath)
		if err != nil {
			continue
		}
		if len(refs) > 0 {
			result.FileRefs[filePath] = refs
		}
	}
	return result, nil
}

func scanFile(filePath string) ([]*Reference, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	seen := make(map[string]bool)
	var refs []*Reference

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ref := ScanLine(line)
		if ref != nil {
			key := ref.ResolvedPath + ref.CrossRepoSuffix
			if !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ResolvedPath+refs[i].CrossRepoSuffix < refs[j].ResolvedPath+refs[j].CrossRepoSuffix
	})
	return refs, nil
}

// ExpandGlobPattern expands a glob pattern to a list of file paths.
func ExpandGlobPattern(pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "**/*"
	}
	if _, err := filepath.Match(pattern, "test"); err != nil && pattern != "**" && pattern != "**/*" {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, err
		}
	}
	var matches []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		normalPath := filepath.ToSlash(path)
		if len(normalPath) >= 2 && normalPath[0:2] == "./" {
			normalPath = normalPath[2:]
		}
		ok, err := matchGlobPattern(normalPath, pattern)
		if err != nil {
			return nil
		}
		if ok {
			matches = append(matches, normalPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func matchGlobPattern(path string, pattern string) (bool, error) {
	if pattern == "**/*" || pattern == "**" {
		return true, nil
	}
	return filepath.Match(pattern, path)
}

// GetUniqueReferences extracts unique references, optionally filtered by type.
func GetUniqueReferences(result *ScanResult, typeFilter string) []*Reference {
	seen := make(map[string]*Reference)
	for _, refs := range result.FileRefs {
		for _, ref := range refs {
			if typeFilter != "" && ref.Type != typeFilter {
				continue
			}
			key := ref.ResolvedPath + ref.CrossRepoSuffix
			if _, exists := seen[key]; !exists {
				seen[key] = ref
			}
		}
	}
	var unique []*Reference
	for _, ref := range seen {
		unique = append(unique, ref)
	}
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].ResolvedPath+unique[i].CrossRepoSuffix < unique[j].ResolvedPath+unique[j].CrossRepoSuffix
	})
	return unique
}

// FormatOutput formats scan results for output.
func FormatOutput(result *ScanResult, singleFile bool, typeFilter string) string {
	if len(result.FileRefs) == 0 {
		return ""
	}
	var output []string
	if singleFile {
		refs := GetUniqueReferences(result, typeFilter)
		for _, ref := range refs {
			output = append(output, ref.ResolvedPath+ref.CrossRepoSuffix)
		}
	} else {
		fileNames := make([]string, 0, len(result.FileRefs))
		for fname := range result.FileRefs {
			fileNames = append(fileNames, fname)
		}
		sort.Strings(fileNames)
		for i, fname := range fileNames {
			if i > 0 {
				output = append(output, "")
			}
			output = append(output, fname)
			refs := result.FileRefs[fname]
			filtered := refs
			if typeFilter != "" {
				filtered = nil
				for _, ref := range refs {
					if ref.Type == typeFilter {
						filtered = append(filtered, ref)
					}
				}
			}
			for _, ref := range filtered {
				output = append(output, "  "+ref.ResolvedPath+ref.CrossRepoSuffix)
			}
		}
	}
	if len(output) == 0 {
		return ""
	}
	result2 := ""
	for i, s := range output {
		if i > 0 {
			result2 += "\n"
		}
		result2 += s
	}
	return fmt.Sprintf("%s\n", result2)
}
```

- [ ] **Step 5: Run tests**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go test ./pkg/sourceref/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/sourceref/
git commit -m "feat: add sourceref package for spec annotation parsing"
```

---

## Task T5: Extract pkg/feature

**Repo:** specscore
**Model:** Opus
**Depends on:** T3

This is the most complex extraction. The synchestra code mixes library logic with cobra command wiring. We must separate them — `pkg/feature` gets pure functions that take paths and return data; no cobra, no `fmt.Print`, no `os.Exit`.

**Files:**
- Create: `pkg/feature/discover.go` — feature discovery, tree building, dependency parsing
- Create: `pkg/feature/info.go` — feature metadata, sections, status parsing, refs, children, plans
- Create: `pkg/feature/deps.go` — dependency and reference resolution (non-transitive)
- Create: `pkg/feature/tree.go` — tree building and filtering
- Create: `pkg/feature/fields.go` — enriched feature type and field resolution
- Create: `pkg/feature/slug.go` — slug validation and generation
- Create: `pkg/feature/template.go` — README template generation
- Create: `pkg/feature/newfeature.go` — scaffold new feature directory
- Create: `pkg/feature/transitive.go` — transitive dependency/reference resolution

**Key API design decisions:**
- All functions take `featuresDir string` as the first parameter (no auto-discovery from CWD — that's CLI concern)
- Return data structures, not formatted strings
- Errors use `exitcode` types from `pkg/exitcode`
- No `cobra` import anywhere in `pkg/feature`

The agent implementing this task should:

- [ ] **Step 1:** Read all files in synchestra's `pkg/cli/feature/` to understand the full codebase
- [ ] **Step 2:** Create `pkg/feature/discover.go` — extract `discoverFeatures`, `findSpecRepoRoot`, `resolveFeaturesDir`, `buildTree`, `featureNode`, `printTree`, `parseDependencies`, `extractFeatureID`, `featureIDFromRelativePath`, `featureExists`, `featureReadmePath` and all supporting types/functions from synchestra's `pkg/cli/feature/discover.go`. Remove the `cobra` import — `resolveFeaturesDir` should take a `startDir string` parameter instead of reading CWD internally. Export the key functions: `Discover`, `FindSpecRepoRoot`, `Exists`, `ReadmePath`.
- [ ] **Step 3:** Create `pkg/feature/info.go` — extract `featureInfo`, `childInfo`, `sectionInfo`, `parseFeatureStatus`, `findFeatureRefs`, `discoverChildFeatures`, `parseContentsTable`, `findLinkedPlans`, `planReferencesFeature`, `parseSections` from synchestra's `pkg/cli/feature/info.go`. Export key types as `Info`, `ChildInfo`, `SectionInfo` and expose `GetInfo(featuresDir, featureID string) (*Info, error)`.
- [ ] **Step 4:** Create `pkg/feature/fields.go` — extract `enrichedFeature`, `resolveFields`, `countOutstandingQuestions`, `validFields`, `parseFieldNames`, output helpers. Export as `EnrichedFeature`, `ResolveFields`, `ValidFields`, `ParseFieldNames`. Remove cobra dependency from `effectiveFormat` — that stays in the CLI layer.
- [ ] **Step 5:** Create `pkg/feature/tree.go` — extract tree building logic: `filterFocusedFeatures`, `markFocus`, `buildEnrichedTree`. Keep `featureNode` and `buildTree` in discover.go (they're used by tree too). Export `BuildTree`, `FilterFocused`, `BuildEnrichedTree`.
- [ ] **Step 6:** Create `pkg/feature/slug.go` — extract `validateSlug`, `generateSlug` from synchestra. Export as `ValidateSlug`, `GenerateSlug`.
- [ ] **Step 7:** Create `pkg/feature/template.go` — extract `generateReadme`, `validStatuses`, `isValidStatus`. Export as `GenerateReadme`, `IsValidStatus`.
- [ ] **Step 8:** Create `pkg/feature/newfeature.go` — extract the scaffolding logic from `new.go`: directory creation, README writing, parent contents update, feature index update. Do NOT include git operations (commit/push) — those stay in synchestra. Export as `New(featuresDir, featureID string, opts NewOptions) (*NewResult, error)` where `NewResult` includes the list of changed files.
- [ ] **Step 9:** Create `pkg/feature/transitive.go` — extract `resolveTransitiveDeps`, `resolveTransitiveRefs`, `walkTransitive`, `enrichTransitiveNodes`, `printTransitiveText` and all supporting types. Export as `TransitiveDeps`, `TransitiveRefs`.
- [ ] **Step 10:** Write tests for key exported functions — at minimum: `TestDiscover`, `TestValidateSlug`, `TestGenerateSlug`, `TestParseDependencies`, `TestGenerateReadme`, `TestBuildTree`.
- [ ] **Step 11:** Run full validation

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
gofmt -w ./pkg/feature/
go build ./...
go vet ./...
go test ./pkg/feature/... -v
```

- [ ] **Step 12: Commit**

```bash
git add pkg/feature/
git commit -m "feat: add feature package — discovery, metadata, deps, tree, scaffolding"
```

---

## Task T6: Extract pkg/lint

**Repo:** specscore
**Model:** Opus
**Depends on:** T3, T5

**Files:**
- Create: `pkg/lint/lint.go` — public API (Violation, Options, Lint, FilterBySeverity)
- Create: `pkg/lint/linter.go` — internal linter engine, checker interface, walkSpecDirs
- Create: `pkg/lint/checkers_extended.go` — stub checkers (heading-levels, feature-ref-syntax, etc.)
- Create: `pkg/lint/readme_exists.go`
- Create: `pkg/lint/oq_section.go`
- Create: `pkg/lint/index_entries.go`
- Create: `pkg/lint/plan_hierarchy.go`
- Create: `pkg/lint/plan_roi.go`

The agent implementing this task should:

- [ ] **Step 1:** Read all files in synchestra's `pkg/cli/spec/` to understand the linting code
- [ ] **Step 2:** Create `pkg/lint/lint.go` with public types and API:

```go
package lint

// Violation represents a single linting violation.
type Violation struct {
	File     string `json:"file" yaml:"file"`
	Line     int    `json:"line" yaml:"line"`
	Severity string `json:"severity" yaml:"severity"`
	Rule     string `json:"rule" yaml:"rule"`
	Message  string `json:"message" yaml:"message"`
}

// Options holds linting options.
type Options struct {
	SpecRoot string
	Rules    []string // enabled rules; nil = all
	Ignore   []string // disabled rules
	Severity string   // minimum severity: error, warning, info
}

// Lint runs all enabled lint rules against the spec tree.
func Lint(opts Options) ([]Violation, error) { ... }

// FilterBySeverity filters violations to those at or above the minimum severity.
func FilterBySeverity(violations []Violation, minSeverity string) []Violation { ... }

// AllRuleNames returns the canonical set of known rule names.
func AllRuleNames() map[string]bool { ... }

// ValidateRuleNames checks that all names are known rules.
func ValidateRuleNames(names []string) error { ... }
```

- [ ] **Step 3:** Create `pkg/lint/linter.go` — move the `linter` struct, `checker` interface, `newLinter`, `registerChecker`, `isRuleEnabled`, `lint`, `walkSpecDirs` from synchestra's `linter.go`. Keep these unexported — they're internal to the lint package.
- [ ] **Step 4:** Copy each checker file from synchestra (`readme_exists.go`, `oq_section.go`, `index_entries.go`, `checkers_extended.go`, `plan_hierarchy.go`, `plan_roi.go`), updating the package declaration from `spec` to `lint`. No other changes needed — they only use stdlib.
- [ ] **Step 5:** Write basic test:

```go
package lint

import "testing"

func TestFilterBySeverity(t *testing.T) {
	violations := []Violation{
		{Severity: "error", Rule: "r1"},
		{Severity: "warning", Rule: "r2"},
		{Severity: "info", Rule: "r3"},
	}
	filtered := FilterBySeverity(violations, "warning")
	if len(filtered) != 2 {
		t.Errorf("expected 2, got %d", len(filtered))
	}
}
```

- [ ] **Step 6:** Run validation

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
gofmt -w ./pkg/lint/
go build ./...
go vet ./...
go test ./pkg/lint/... -v
```

- [ ] **Step 7: Commit**

```bash
git add pkg/lint/
git commit -m "feat: add lint package — spec validation engine with all rules"
```

---

## Task T7: Extract pkg/projectdef

**Repo:** specscore
**Model:** Sonnet
**Depends on:** T3

**Files:**
- Create: `pkg/projectdef/projectdef.go`
- Create: `pkg/projectdef/projectdef_test.go`

- [ ] **Step 1: Write failing test**

Create `pkg/projectdef/projectdef_test.go`:

```go
package projectdef

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpecConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := SpecConfig{
		Title:     "Test Project",
		StateRepo: "https://github.com/test/state.git",
		Repos:     []string{"https://github.com/test/code.git"},
	}
	if err := WriteSpecConfig(dir, cfg); err != nil {
		t.Fatalf("WriteSpecConfig: %v", err)
	}
	got, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatalf("ReadSpecConfig: %v", err)
	}
	if got.Title != cfg.Title {
		t.Errorf("Title = %q, want %q", got.Title, cfg.Title)
	}
	if got.StateRepo != cfg.StateRepo {
		t.Errorf("StateRepo = %q, want %q", got.StateRepo, cfg.StateRepo)
	}
}

func TestParseStateRepo(t *testing.T) {
	tests := []struct {
		stateRepo  string
		wantMode   string
		wantBranch string
	}{
		{"worktree://synchestra-state", "worktree", "synchestra-state"},
		{"https://github.com/test/state.git", "repo", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		cfg := SpecConfig{StateRepo: tt.stateRepo}
		mode, branch := cfg.ParseStateRepo()
		if mode != tt.wantMode || branch != tt.wantBranch {
			t.Errorf("ParseStateRepo(%q) = (%q, %q), want (%q, %q)",
				tt.stateRepo, mode, branch, tt.wantMode, tt.wantBranch)
		}
	}
}

func TestSpecConfigFileExists(t *testing.T) {
	dir := t.TempDir()
	cfg := SpecConfig{Title: "t"}
	if err := WriteSpecConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, SpecConfigFile)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist", path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go test ./pkg/projectdef/... -v
```

Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Create pkg/projectdef/projectdef.go**

Extract config types and read/write functions from synchestra's `pkg/cli/project/configfiles.go`:

```go
// Package projectdef provides the specscore-project.yaml schema and read/write operations.
package projectdef

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SpecConfigFile    = "synchestra-spec-repo.yaml"
	StateConfigFile   = "synchestra-state-repo.yaml"
	CodeConfigFile    = "synchestra-code-repo.yaml"
	EmbeddedStateFile = "synchestra-state.yaml"
)

const worktreeScheme = "worktree://"

// PlanningConfig holds planning-related settings from synchestra-spec-repo.yaml.
type PlanningConfig struct {
	WhatsNext string `yaml:"whats_next"`
}

// SpecConfig represents the contents of synchestra-spec-repo.yaml.
type SpecConfig struct {
	Title     string          `yaml:"title"`
	StateRepo string          `yaml:"state_repo"`
	Repos     []string        `yaml:"repos"`
	Planning  *PlanningConfig `yaml:"planning,omitempty"`
}

// WhatsNextMode returns the effective whats_next mode, defaulting to "disabled".
func (c SpecConfig) WhatsNextMode() string {
	if c.Planning != nil && c.Planning.WhatsNext != "" {
		return c.Planning.WhatsNext
	}
	return "disabled"
}

// ParseStateRepo parses the state_repo field.
// Returns (mode, branch):
//   - ("worktree", branchName) for "worktree://branchName"
//   - ("repo", "") for any other non-empty value
//   - ("", "") if state_repo is empty
func (c SpecConfig) ParseStateRepo() (mode, branch string) {
	if c.StateRepo == "" {
		return "", ""
	}
	if strings.HasPrefix(c.StateRepo, worktreeScheme) {
		b := c.StateRepo[len(worktreeScheme):]
		if b == "" {
			return "", ""
		}
		return "worktree", b
	}
	return "repo", ""
}

// StateConfig represents the contents of synchestra-state-repo.yaml.
type StateConfig struct {
	Title     string   `yaml:"title"`
	MainRepo  string   `yaml:"main_repo"`
	SpecRepos []string `yaml:"spec_repos"`
	CodeRepos []string `yaml:"code_repos,omitempty"`
}

// CodeConfig represents the contents of synchestra-code-repo.yaml.
type CodeConfig struct {
	SpecRepos []string `yaml:"spec_repos"`
}

// EmbeddedStateConfig lives on the orphan branch.
type EmbeddedStateConfig struct {
	Title        string           `yaml:"title"`
	Mode         string           `yaml:"mode"`
	SourceBranch string           `yaml:"source_branch"`
	Sync         *EmbeddedSyncCfg `yaml:"sync,omitempty"`
}

// EmbeddedSyncCfg controls sync policy for embedded state.
type EmbeddedSyncCfg struct {
	Pull string `yaml:"pull"`
	Push string `yaml:"push"`
}

func WriteSpecConfig(dir string, cfg SpecConfig) error {
	return writeYAML(filepath.Join(dir, SpecConfigFile), cfg)
}

func WriteStateConfig(dir string, cfg StateConfig) error {
	return writeYAML(filepath.Join(dir, StateConfigFile), cfg)
}

func WriteCodeConfig(dir string, cfg CodeConfig) error {
	return writeYAML(filepath.Join(dir, CodeConfigFile), cfg)
}

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

func WriteEmbeddedStateConfig(dir string, cfg EmbeddedStateConfig) error {
	return writeYAML(filepath.Join(dir, EmbeddedStateFile), cfg)
}

func ReadEmbeddedStateConfig(dir string) (EmbeddedStateConfig, error) {
	var cfg EmbeddedStateConfig
	data, err := os.ReadFile(filepath.Join(dir, EmbeddedStateFile))
	if err != nil {
		return cfg, fmt.Errorf("reading embedded state config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing embedded state config: %w", err)
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

- [ ] **Step 4: Add yaml dependency**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go get gopkg.in/yaml.v3@v3.0.1
go mod tidy
```

- [ ] **Step 5: Run tests**

```bash
go test ./pkg/projectdef/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/projectdef/ go.mod go.sum
git commit -m "feat: add projectdef package — YAML config schema and read/write"
```

---

## Task R2: Review all extracted packages

**Repo:** specscore
**Model:** Opus
**Depends on:** T4, T5, T6, T7

**Review checklist:**

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes — all package tests green
- [ ] `gofmt -d .` shows no formatting issues
- [ ] **No cobra import anywhere in `pkg/`** — `grep -r "spf13/cobra" pkg/` returns nothing
- [ ] **No `fmt.Print`/`fmt.Println` in `pkg/`** — library functions return data, not print
- [ ] **No `os.Exit` in `pkg/`** — exits are CLI concern
- [ ] `pkg/exitcode` has no external dependencies beyond stdlib
- [ ] `pkg/sourceref` has no external dependencies beyond stdlib
- [ ] `pkg/feature` depends only on `pkg/exitcode` and stdlib (plus yaml for field output)
- [ ] `pkg/lint` depends only on stdlib (no exitcode dependency — it returns Violation structs)
- [ ] `pkg/projectdef` depends only on yaml.v3
- [ ] All exported functions have consistent parameter patterns (featuresDir first, etc.)
- [ ] No synchestra-specific logic leaked (no `synchestra` in user-facing strings unless in legacy annotation patterns)

---

## Task T8: Wire internal/cli commands

**Repo:** specscore
**Model:** Sonnet
**Depends on:** R2

**Files:**
- Modify: `internal/cli/root.go` — add feature, spec, code subcommands
- Create: `internal/cli/feature.go` — thin cobra wrappers around `pkg/feature`
- Create: `internal/cli/spec.go` — thin cobra wrapper around `pkg/lint`
- Create: `internal/cli/code.go` — thin cobra wrapper around `pkg/sourceref`

The agent implementing this task should:

- [ ] **Step 1:** Read the existing `internal/cli/root.go` and all `pkg/` package APIs
- [ ] **Step 2:** Create `internal/cli/feature.go` — implement cobra commands `feature info`, `feature list`, `feature tree`, `feature deps`, `feature refs`, `feature new` as thin wrappers. Each command: parses flags → calls `pkg/feature` function → formats output (text/yaml/json). Mirror synchestra's flag names exactly (`--project`, `--fields`, `--format`, `--transitive`, `--direction`, etc.).
- [ ] **Step 3:** Create `internal/cli/spec.go` — implement `spec lint` command wrapping `pkg/lint.Lint()`. Mirror synchestra's flags: `--rules`, `--ignore`, `--severity`, `--format`. Handle output formatting (text/json/yaml) in the CLI layer.
- [ ] **Step 4:** Create `internal/cli/code.go` — implement `code deps` command wrapping `pkg/sourceref.ScanFiles()` and `FormatOutput()`. Mirror synchestra's flags: `--path`, `--type`.
- [ ] **Step 5:** Update `internal/cli/root.go` to register the new command groups:

```go
rootCmd.AddCommand(
    versionCommand(),
    featureCommand(),
    specCommand(),
    codeCommand(),
)
```

- [ ] **Step 6:** Run full validation

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
gofmt -w ./internal/cli/
go build ./...
go vet ./...
go test ./...
```

- [ ] **Step 7:** Manual smoke test

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
go run ./cmd/specscore version
go run ./cmd/specscore feature list
go run ./cmd/specscore spec lint
```

- [ ] **Step 8: Commit**

```bash
git add internal/cli/ cmd/
git commit -m "feat: wire CLI commands — feature, spec lint, code deps"
```

---

## Task R3: Review CLI wiring

**Repo:** specscore
**Model:** Sonnet
**Depends on:** T8

**Review checklist:**

- [ ] `go build ./...` passes
- [ ] `specscore version` outputs version info
- [ ] `specscore feature list` discovers features in specscore's own `spec/features/`
- [ ] `specscore feature info feature` shows metadata for the `feature` spec
- [ ] `specscore feature tree` shows the hierarchy
- [ ] `specscore spec lint` runs against specscore's own `spec/` tree
- [ ] `specscore feature --help` shows all subcommands
- [ ] Flag names match synchestra's exactly
- [ ] CLI layer does NOT contain business logic — it's all in `pkg/`

---

## Task T9: Update synchestra to import specscore

**Repo:** synchestra
**Model:** Opus
**Depends on:** R3

This is the high-risk refactor — changing synchestra to use specscore packages.

**Files:**
- Modify: `go.mod` — add specscore dependency with replace directive
- Delete: `pkg/cli/exitcode/` — imported from specscore now
- Delete: `pkg/sourceref/` — imported from specscore now
- Modify: `pkg/cli/feature/*.go` — rewrite as thin wrappers around `specscore/pkg/feature`
- Modify: `pkg/cli/spec/*.go` — rewrite as thin wrapper around `specscore/pkg/lint`
- Modify: `pkg/cli/code/*.go` — rewrite as thin wrapper around `specscore/pkg/sourceref`
- Modify: `pkg/cli/project/configfiles.go` — import from `specscore/pkg/projectdef`
- Modify: `pkg/cli/main.go` — update imports
- Modify: all files that import `pkg/cli/exitcode` or `pkg/sourceref` — update import paths

The agent implementing this task should:

- [ ] **Step 1:** Add specscore to `go.mod` with replace directive:

Add to synchestra's `go.mod`:
```
require github.com/synchestra-io/specscore v0.0.0

replace github.com/synchestra-io/specscore => ../specscore
```

Then run `go mod tidy`.

- [ ] **Step 2:** Update all `exitcode` imports. Every file importing `github.com/synchestra-io/synchestra/pkg/cli/exitcode` must change to `github.com/synchestra-io/specscore/pkg/exitcode`. Search with: `grep -r "synchestra/pkg/cli/exitcode" pkg/`

- [ ] **Step 3:** Update all `sourceref` imports. Every file importing `github.com/synchestra-io/synchestra/pkg/sourceref` must change to `github.com/synchestra-io/specscore/pkg/sourceref`.

- [ ] **Step 4:** Rewrite `pkg/cli/feature/` as thin wrappers. Each command file keeps its cobra definition and flags but delegates to `specscore/pkg/feature` for the actual work. The `discover.go` helper functions (like `resolveFeaturesDir`, `discoverFeatures`, etc.) are removed — those now live in specscore. Fields, template, slug, transitive files are removed — they live in specscore.

- [ ] **Step 5:** Rewrite `pkg/cli/spec/` as thin wrapper around `specscore/pkg/lint`. Remove `linter.go`, all checker files. Keep only `spec.go` (command group) and `lint.go` (cobra wrapper calling `lint.Lint()`).

- [ ] **Step 6:** Rewrite `pkg/cli/code/` as thin wrapper. Keep `code.go` and `deps.go` but have `deps.go` import from `specscore/pkg/sourceref`.

- [ ] **Step 7:** Update `pkg/cli/project/configfiles.go` to re-export types from specscore/pkg/projectdef:

```go
package project

import "github.com/synchestra-io/specscore/pkg/projectdef"

// Re-export config types and functions from specscore.
type SpecConfig = projectdef.SpecConfig
type StateConfig = projectdef.StateConfig
// ... etc

const SpecConfigFile = projectdef.SpecConfigFile
// ... etc

var ReadSpecConfig = projectdef.ReadSpecConfig
var WriteSpecConfig = projectdef.WriteSpecConfig
// ... etc
```

- [ ] **Step 8:** Delete `pkg/cli/exitcode/` directory (now imported from specscore)
- [ ] **Step 9:** Delete `pkg/sourceref/` directory (now imported from specscore)

- [ ] **Step 10:** Run full validation

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
gofmt -w .
go mod tidy
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "refactor: import specscore packages, remove extracted code"
```

---

## Task T10: Finalize — remove replace, tag v0.1.0

**Repo:** both
**Model:** Sonnet
**Depends on:** T9

- [ ] **Step 1:** Push specscore to GitHub (if not already)

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
git push origin main
```

- [ ] **Step 2:** Tag specscore v0.1.0

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/specscore
git tag v0.1.0
git push origin v0.1.0
```

- [ ] **Step 3:** Wait for the tag to be available on GitHub, then update synchestra's go.mod

Remove the `replace` directive from synchestra's `go.mod` and update the require:

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
```

Edit `go.mod`: remove the `replace github.com/synchestra-io/specscore => ../specscore` line. Then:

```bash
go get github.com/synchestra-io/specscore@v0.1.0
go mod tidy
```

- [ ] **Step 4:** Verify synchestra builds from clean state

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
go build ./...
go vet ./...
go test ./...
```

- [ ] **Step 5:** Verify specscore builds independently

```bash
cd /tmp
git clone https://github.com/synchestra-io/specscore.git specscore-test
cd specscore-test
go build ./...
go test ./...
```

- [ ] **Step 6: Commit synchestra**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
git add go.mod go.sum
git commit -m "chore: pin specscore v0.1.0, remove replace directive"
```

---

## Task R4: Final review

**Repo:** both
**Model:** Opus
**Depends on:** T10

**Review checklist:**

- [ ] **specscore builds independently** — `go build ./...` from clean clone
- [ ] **specscore tests pass** — `go test ./...`
- [ ] **specscore CLI works** — `specscore version`, `specscore feature list`, `specscore spec lint`
- [ ] **synchestra builds independently** — `go build ./...` (no replace directive)
- [ ] **synchestra tests pass** — `go test ./...`
- [ ] **synchestra CLI works** — `synchestra feature list`, `synchestra spec lint`, `synchestra code deps`
- [ ] **No replace directive** in synchestra's go.mod
- [ ] **No duplicate code** — `pkg/cli/exitcode/` and `pkg/sourceref/` deleted from synchestra
- [ ] **Thin wrappers** — synchestra's feature/spec/code CLI files contain only cobra wiring, no business logic
- [ ] **CI passes on both repos** — workflows triggered and green
- [ ] **goreleaser snapshot builds** on specscore
