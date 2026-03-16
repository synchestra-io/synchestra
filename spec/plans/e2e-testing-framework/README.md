# E2E Testing Framework & Acceptance Criteria Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a markdown-native test scenario runner (`pkg/testscenario/`) and establish acceptance criteria as first-class feature artifacts, then dogfood both by writing the CLI project lifecycle E2E test.

**Architecture:** A Go package (`pkg/testscenario/`) that parses markdown scenario files, resolves AC references from feature spec `acs/` directories, executes steps sequentially (with opt-in parallel groups), and reports results. The package has no dependencies on Synchestra-specific code — it receives a configurable spec root path and resolves everything from the filesystem. CLI commands (`synchestra test run`, `synchestra test list`) wire the package to the command tree.

**Tech Stack:** Go 1.26, standard library (`os/exec` for shell execution, `sync` for parallel groups), `github.com/spf13/cobra` for CLI commands. No new external dependencies for the core package.

**Spec:** `docs/superpowers/specs/2026-03-16-e2e-testing-and-acceptance-criteria-design.md`

**AGENTS.md rules to follow:**
- Every `.go` file must have `// Features implemented:` / `// Features depended on:` comments after the `package` declaration
- After any change to `.go` files: `gofmt -w`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`, `go vet ./...`
- Every directory MUST have a `README.md` with an "Outstanding Questions" section

---

## File Structure

```
pkg/
  testscenario/
    README.md         — package documentation
    types.go          — Scenario, Step, ACRef, Output, StepResult, ACResult structs
    parser.go         — markdown scenario parser (headings, metadata, code blocks)
    parser_test.go    — parser tests
    context.go        — execution context: context/step output storage, variable resolution
    context_test.go   — context tests
    ac.go             — AC file parser + resolver (reads acs/*.md, extracts verification scripts)
    ac_test.go        — AC resolution tests
    runner.go         — step executor: sequential/parallel, shell execution, output capture
    runner_test.go    — runner tests
    include.go        — sub-flow resolution: recursive inclusion, cycle detection
    include_test.go   — include tests
    reporter.go       — results collection, text report formatting
    reporter_test.go  — reporter tests

cli/
  test/
    README.md         — CLI test command group documentation
    test.go           — Cobra command group (`synchestra test`)
    run.go            — `synchestra test run` command
    list.go           — `synchestra test list` command
```

Each file has one responsibility. The parser knows nothing about execution; the runner knows nothing about parsing; the AC resolver is a standalone utility.

---

## Chunk 1: Types and Parser

### Task 1: Create `pkg/testscenario/` package with types

**Files:**
- Create: `pkg/testscenario/README.md`
- Create: `pkg/testscenario/types.go`

- [ ] **Step 1: Create `pkg/testscenario/README.md`**

```markdown
# pkg/testscenario

Markdown-native test scenario runner. Parses `.md` scenario files into structured
step sequences, resolves acceptance criteria references, executes steps with
input/output passing, and reports results.

This package has no dependencies on Synchestra-specific code. It receives a
configurable spec root path and resolves AC references from the filesystem.

## Spec

See `docs/superpowers/specs/2026-03-16-e2e-testing-and-acceptance-criteria-design.md`

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `pkg/testscenario/types.go`**

```go
package testscenario

// Features implemented: test-scenario

// OutputStore indicates where a step output is stored.
type OutputStore string

const (
	StoreContext OutputStore = "context"
	StoreStep    OutputStore = "step"
	StoreBoth    OutputStore = "both"
)

// Output defines a named value extracted from step execution.
type Output struct {
	Name    string
	Store   OutputStore
	Extract string // shell expression to extract value
}

// ACRef references acceptance criteria to verify after a step.
type ACRef struct {
	FeaturePath string // e.g., "cli/project/new"
	FeatureLink string // markdown link target
	ACs         string // "*" or comma-separated AC slugs
}

// Step is a named step in a test scenario.
type Step struct {
	Name      string
	DependsOn []string
	Parallel  bool
	Outputs   []Output
	ACs       []ACRef
	Include   string // path to sub-flow .md file, empty if inline code
	Code      string // bash code block content, empty if include
}

// Scenario is a parsed test scenario.
type Scenario struct {
	Title       string
	Description string
	Tags        []string
	Setup       string // bash code for setup
	Teardown    string // bash code for teardown
	Steps       []Step
}

// ACFile is a parsed acceptance criteria file.
type ACFile struct {
	Slug         string
	Status       string
	FeaturePath  string
	Description  string
	Inputs       []ACInput
	Verification string // bash verification script
}

// ACInput is a named input for an AC verification script.
type ACInput struct {
	Name        string
	Required    bool
	Description string
}

// StepResult holds the outcome of executing a single step.
type StepResult struct {
	StepName string
	Passed   bool
	Error    string
	Stdout   string
	Stderr   string
	ExitCode int
	ACResults []ACResult
}

// ACResult holds the outcome of a single AC verification.
type ACResult struct {
	FeaturePath string
	ACSlug      string
	Passed      bool
	Error       string
}

// ScenarioResult holds the outcome of a full scenario run.
type ScenarioResult struct {
	ScenarioTitle string
	Passed        bool
	StepResults   []StepResult
	SetupError    string
	TeardownError string
}
```

- [ ] **Step 3: Run Go validation**

Run: `gofmt -w pkg/testscenario/types.go && golangci-lint run ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass (no tests yet).

- [ ] **Step 4: Commit**

```bash
git add pkg/testscenario/
git commit -m "feat(testscenario): add types package with scenario, step, and result structs"
```

---

### Task 2: Build the markdown scenario parser

**Files:**
- Create: `pkg/testscenario/parser.go`
- Create: `pkg/testscenario/parser_test.go`

- [ ] **Step 1: Write failing parser tests**

Create `pkg/testscenario/parser_test.go` with tests for:
1. Parsing scenario header (title, description, tags)
2. Parsing Setup and Teardown blocks
3. Parsing a step with code block, outputs, and ACs
4. Parsing a step with `Parallel: true`
5. Parsing a step with `Include:`
6. Parsing a step with `Depends on:`
7. Validation: duplicate step names → error
8. Validation: step with neither code nor include → error
9. Validation: step with both code and include → error

```go
package testscenario

// Features implemented: test-scenario

import "testing"

func TestParseScenario_header(t *testing.T) {
	input := `# Scenario: My test

**Description:** A test scenario.
**Tags:** e2e, cli

## setup-step

` + "```bash\necho hello\n```"

	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Title != "My test" {
		t.Errorf("title = %q, want %q", s.Title, "My test")
	}
	if s.Description != "A test scenario." {
		t.Errorf("description = %q, want %q", s.Description, "A test scenario.")
	}
	if len(s.Tags) != 2 || s.Tags[0] != "e2e" || s.Tags[1] != "cli" {
		t.Errorf("tags = %v, want [e2e cli]", s.Tags)
	}
}

func TestParseScenario_setupTeardown(t *testing.T) {
	input := "# Scenario: T\n\n## Setup\n\n```bash\nexport X=1\n```\n\n## do-thing\n\n```bash\necho ok\n```\n\n## Teardown\n\n```bash\nrm -rf /tmp/test\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Setup != "export X=1" {
		t.Errorf("setup = %q, want %q", s.Setup, "export X=1")
	}
	if s.Teardown != "rm -rf /tmp/test" {
		t.Errorf("teardown = %q, want %q", s.Teardown, "rm -rf /tmp/test")
	}
}

func TestParseScenario_stepWithOutputsAndACs(t *testing.T) {
	input := "# Scenario: T\n\n## create-project\n\n**Outputs:**\n\n| Name | Store | Extract |\n|---|---|---|\n| project_id | context | `echo test` |\n\n**ACs:**\n\n| Feature | ACs |\n|---|---|\n| [cli/project/new](spec/features/cli/project/new/) | * |\n\n```bash\nsynchestra project new\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(s.Steps))
	}
	step := s.Steps[0]
	if step.Name != "create-project" {
		t.Errorf("name = %q, want %q", step.Name, "create-project")
	}
	if len(step.Outputs) != 1 || step.Outputs[0].Name != "project_id" || step.Outputs[0].Store != StoreContext {
		t.Errorf("outputs = %+v, want [{project_id context echo test}]", step.Outputs)
	}
	if len(step.ACs) != 1 || step.ACs[0].FeaturePath != "cli/project/new" || step.ACs[0].ACs != "*" {
		t.Errorf("acs = %+v", step.ACs)
	}
	if step.Code != "synchestra project new" {
		t.Errorf("code = %q", step.Code)
	}
}

func TestParseScenario_parallelStep(t *testing.T) {
	input := "# Scenario: T\n\n## step-a\n**Parallel:** true\n\n```bash\necho a\n```\n\n## step-b\n**Parallel:** true\n\n```bash\necho b\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Steps[0].Parallel || !s.Steps[1].Parallel {
		t.Errorf("parallel flags: step-a=%v, step-b=%v", s.Steps[0].Parallel, s.Steps[1].Parallel)
	}
}

func TestParseScenario_includeStep(t *testing.T) {
	input := "# Scenario: T\n\n## start-container\n\n**Include:** [flows/start.md](flows/start.md)\n"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Steps[0].Include != "flows/start.md" {
		t.Errorf("include = %q, want %q", s.Steps[0].Include, "flows/start.md")
	}
}

func TestParseScenario_dependsOn(t *testing.T) {
	input := "# Scenario: T\n\n## step-a\n\n```bash\necho a\n```\n\n## step-b\n**Depends on:** step-a\n\n```bash\necho b\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Steps[1].DependsOn) != 1 || s.Steps[1].DependsOn[0] != "step-a" {
		t.Errorf("depends_on = %v, want [step-a]", s.Steps[1].DependsOn)
	}
}

func TestParseScenario_duplicateStepNames(t *testing.T) {
	input := "# Scenario: T\n\n## same-name\n\n```bash\necho 1\n```\n\n## same-name\n\n```bash\necho 2\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate step names")
	}
}

func TestParseScenario_stepWithNeitherCodeNorInclude(t *testing.T) {
	input := "# Scenario: T\n\n## empty-step\n\n**Depends on:** (none)\n"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for step with neither code nor include")
	}
}

func TestParseScenario_stepWithBothCodeAndInclude(t *testing.T) {
	input := "# Scenario: T\n\n## bad-step\n\n**Include:** [flows/x.md](flows/x.md)\n\n```bash\necho oops\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for step with both code and include")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -count=1`
Expected: FAIL — `ParseScenario` undefined.

- [ ] **Step 3: Implement the parser**

Create `pkg/testscenario/parser.go`:

The parser works by:
1. Splitting the markdown into sections by `## ` headings
2. Parsing the `# Scenario:` title line and `**Description:**`/`**Tags:**` metadata
3. For each `## ` section:
   - If heading is "Setup" or "Teardown" → extract code block, store on Scenario
   - Otherwise → parse as a Step: extract `**Depends on:**`, `**Parallel:**`, `**Outputs:**` table, `**ACs:**` table, `**Include:**`, and code block
4. Validate: unique names, each step has exactly one of code or include, `Depends on` references exist and point to earlier steps

Key implementation details:
- Use `strings.Split` on `"\n## "` to split sections (after separating the H1)
- Parse markdown tables by splitting on `|` and trimming cells
- Extract code blocks between ` ```bash ` and ` ``` ` markers
- Extract markdown links from `[text](url)` patterns for Feature column in ACs table
- Feature path is extracted from the link text (not the URL)

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"strings"
)

// ParseScenario parses a markdown scenario file into a Scenario struct.
func ParseScenario(data []byte) (*Scenario, error) {
	lines := strings.Split(string(data), "\n")
	s := &Scenario{}
	stepNames := make(map[string]bool)

	// Phase 1: Extract H1 title ("# Scenario: <title>")
	// Phase 2: Extract header metadata (**Description:**, **Tags:**)
	// Phase 3: Split remaining content by "## " headings into sections
	// Phase 4: For each section:
	//   - "Setup" → extract code block into s.Setup
	//   - "Teardown" → extract code block into s.Teardown
	//   - Anything else → parse as Step:
	//     - Name from heading text (must be kebab-case)
	//     - Extract **Depends on:** → split by comma, trim
	//     - Extract **Parallel:** true/false
	//     - Extract **Outputs:** table → parse rows into []Output
	//     - Extract **ACs:** table → parse Feature column for path, ACs column for selector
	//     - Extract **Include:** → parse markdown link for path
	//     - Extract ```bash code block
	// Phase 5: Validate:
	//   - No duplicate step names (check stepNames map)
	//   - Each step has exactly one of Include or Code (not both, not neither)
	//   - Depends on references exist and point to earlier steps

	// Helper: extractCodeBlock(lines []string) string — finds ```bash...``` and returns content
	// Helper: parseTable(lines []string) [][]string — parses markdown table rows into cells
	// Helper: parseMarkdownLink(text string) (text, url string) — extracts [text](url)

	_ = lines
	_ = stepNames
	return s, nil
}
```

The function signature and approach are defined; the implementation parses line-by-line, tracking current section context.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/parser.go pkg/testscenario/parser_test.go
git commit -m "feat(testscenario): implement markdown scenario parser with validation"
```

---

## Chunk 2: Execution Context and AC Resolver

### Task 3: Build the execution context (variable storage and resolution)

**Files:**
- Create: `pkg/testscenario/context.go`
- Create: `pkg/testscenario/context_test.go`

- [ ] **Step 1: Write failing context tests**

Tests for:
1. Store output to context scope → retrieve via `context.{name}`
2. Store output to step scope → retrieve via `steps.{step-name}.outputs.{name}`
3. Store output to both → retrievable via either syntax
4. Duplicate context key write → error
5. Resolve `${{ context.project_id }}` in a string → substituted value
6. Resolve `${{ steps.create.outputs.id }}` in a string → substituted value
7. Resolve unknown variable → error

The `ExecContext` struct holds two maps:
- `contextVars map[string]string` — global context
- `stepOutputs map[string]map[string]string` — per-step outputs (`stepName → outputName → value`)

```go
package testscenario

// Features implemented: test-scenario

import "testing"

func TestExecContext_storeAndResolveContext(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("create", "project_id", "p-123", StoreContext); err != nil {
		t.Fatal(err)
	}
	val, err := ctx.ResolveVar("context.project_id")
	if err != nil {
		t.Fatal(err)
	}
	if val != "p-123" {
		t.Errorf("got %q, want %q", val, "p-123")
	}
}

func TestExecContext_storeAndResolveStep(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("create", "raw", "data", StoreStep); err != nil {
		t.Fatal(err)
	}
	val, err := ctx.ResolveVar("steps.create.outputs.raw")
	if err != nil {
		t.Fatal(err)
	}
	if val != "data" {
		t.Errorf("got %q, want %q", val, "data")
	}
}

func TestExecContext_storeBoth(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("s1", "id", "x", StoreBoth); err != nil {
		t.Fatal(err)
	}
	v1, _ := ctx.ResolveVar("context.id")
	v2, _ := ctx.ResolveVar("steps.s1.outputs.id")
	if v1 != "x" || v2 != "x" {
		t.Errorf("context=%q step=%q, both should be %q", v1, v2, "x")
	}
}

func TestExecContext_duplicateContextKey(t *testing.T) {
	ctx := NewExecContext()
	_ = ctx.StoreOutput("s1", "id", "x", StoreContext)
	err := ctx.StoreOutput("s2", "id", "y", StoreContext)
	if err == nil {
		t.Fatal("expected error for duplicate context key")
	}
}

func TestExecContext_resolveInString(t *testing.T) {
	ctx := NewExecContext()
	_ = ctx.StoreOutput("create", "pid", "p-1", StoreContext)
	result, err := ctx.ResolveString("synchestra remove --id ${{ context.pid }}")
	if err != nil {
		t.Fatal(err)
	}
	if result != "synchestra remove --id p-1" {
		t.Errorf("got %q", result)
	}
}

func TestExecContext_resolveUnknownVar(t *testing.T) {
	ctx := NewExecContext()
	_, err := ctx.ResolveVar("context.missing")
	if err == nil {
		t.Fatal("expected error for unknown variable")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -run TestExecContext -count=1`
Expected: FAIL — `NewExecContext` undefined.

- [ ] **Step 3: Implement `context.go`**

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)

// ExecContext holds variable state during scenario execution.
type ExecContext struct {
	contextVars map[string]string
	stepOutputs map[string]map[string]string
}

// NewExecContext creates a new empty execution context.
func NewExecContext() *ExecContext {
	return &ExecContext{
		contextVars: make(map[string]string),
		stepOutputs: make(map[string]map[string]string),
	}
}

// StoreOutput stores a named output from a step.
func (c *ExecContext) StoreOutput(stepName, name, value string, store OutputStore) error {
	switch store {
	case StoreContext, StoreBoth:
		if _, exists := c.contextVars[name]; exists {
			return fmt.Errorf("duplicate context key %q", name)
		}
		c.contextVars[name] = value
	}
	switch store {
	case StoreStep, StoreBoth:
		if c.stepOutputs[stepName] == nil {
			c.stepOutputs[stepName] = make(map[string]string)
		}
		c.stepOutputs[stepName][name] = value
	}
	return nil
}

// ResolveVar resolves a variable reference like "context.pid" or "steps.create.outputs.id".
func (c *ExecContext) ResolveVar(ref string) (string, error) {
	if strings.HasPrefix(ref, "context.") {
		name := strings.TrimPrefix(ref, "context.")
		if val, ok := c.contextVars[name]; ok {
			return val, nil
		}
		return "", fmt.Errorf("unknown context variable %q", name)
	}
	if strings.HasPrefix(ref, "steps.") {
		parts := strings.SplitN(strings.TrimPrefix(ref, "steps."), ".outputs.", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid step output reference %q", ref)
		}
		stepName, outputName := parts[0], parts[1]
		if outputs, ok := c.stepOutputs[stepName]; ok {
			if val, ok := outputs[outputName]; ok {
				return val, nil
			}
		}
		return "", fmt.Errorf("unknown step output %q", ref)
	}
	return "", fmt.Errorf("unknown variable reference %q", ref)
}

// ResolveString replaces all ${{ ... }} references in a string.
func (c *ExecContext) ResolveString(s string) (string, error) {
	var resolveErr error
	result := varPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := varPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		val, err := c.ResolveVar(strings.TrimSpace(sub[1]))
		if err != nil {
			resolveErr = err
			return match
		}
		return val
	})
	return result, resolveErr
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -run TestExecContext -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go test ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/context.go pkg/testscenario/context_test.go
git commit -m "feat(testscenario): add execution context with variable storage and resolution"
```

---

### Task 4: Build the AC file parser and resolver

**Files:**
- Create: `pkg/testscenario/ac.go`
- Create: `pkg/testscenario/ac_test.go`

- [ ] **Step 1: Write failing AC tests**

Tests for:
1. Parse a well-formed AC `.md` file → extract slug, status, inputs, verification script
2. Parse AC file with optional input (Required=No) → correctly parsed
3. Resolve wildcard `*` for a feature path → finds all `.md` files in `acs/` dir
4. Resolve specific AC slug → finds the one file
5. Resolve AC for non-existent feature path → error
6. Validate required AC input is present in available vars → pass
7. Validate required AC input is missing → configuration error

The tests use `t.TempDir()` to create mock feature directories with AC files.

```go
package testscenario

// Features implemented: test-scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func writeACFile(t *testing.T, dir, slug, content string) {
	t.Helper()
	acsDir := filepath.Join(dir, "acs")
	if err := os.MkdirAll(acsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(acsDir, slug+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleAC = `# AC: not-in-list

**Status:** implemented
**Feature:** [cli/project/remove](../README.md)

## Description

Deleted project absent from list.

## Inputs

| Name | Required | Description |
|---|---|---|
| project_id | Yes | ID of the deleted project |

## Verification

` + "```bash\n! echo $project_id\n```"

func TestParseACFile(t *testing.T) {
	ac, err := ParseACFile([]byte(sampleAC), "not-in-list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ac.Slug != "not-in-list" {
		t.Errorf("slug = %q", ac.Slug)
	}
	if ac.Status != "implemented" {
		t.Errorf("status = %q", ac.Status)
	}
	if len(ac.Inputs) != 1 || !ac.Inputs[0].Required {
		t.Errorf("inputs = %+v", ac.Inputs)
	}
	if ac.Verification != "! echo $project_id" {
		t.Errorf("verification = %q", ac.Verification)
	}
}

func TestResolveACs_wildcard(t *testing.T) {
	specRoot := t.TempDir()
	featureDir := filepath.Join(specRoot, "features", "cli", "project", "remove")
	writeACFile(t, featureDir, "not-in-list", sampleAC)
	writeACFile(t, featureDir, "recreate", sampleAC)

	resolver := NewACResolver(specRoot)
	acs, err := resolver.Resolve("cli/project/remove", "*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acs) != 2 {
		t.Errorf("got %d ACs, want 2", len(acs))
	}
}

func TestResolveACs_specific(t *testing.T) {
	specRoot := t.TempDir()
	featureDir := filepath.Join(specRoot, "features", "cli", "project", "remove")
	writeACFile(t, featureDir, "not-in-list", sampleAC)
	writeACFile(t, featureDir, "recreate", sampleAC)

	resolver := NewACResolver(specRoot)
	acs, err := resolver.Resolve("cli/project/remove", "not-in-list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acs) != 1 || acs[0].Slug != "not-in-list" {
		t.Errorf("got %+v", acs)
	}
}

func TestResolveACs_nonExistentFeature(t *testing.T) {
	specRoot := t.TempDir()
	resolver := NewACResolver(specRoot)
	_, err := resolver.Resolve("does/not/exist", "*")
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -run "TestParseACFile|TestResolveACs" -count=1`
Expected: FAIL — `ParseACFile`, `NewACResolver` undefined.

- [ ] **Step 3: Implement `ac.go`**

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ACResolver resolves AC references from the filesystem.
type ACResolver struct {
	specRoot string
}

// NewACResolver creates a resolver rooted at the given spec directory.
func NewACResolver(specRoot string) *ACResolver {
	return &ACResolver{specRoot: specRoot}
}

// Resolve finds and parses AC files for a feature path and selector.
func (r *ACResolver) Resolve(featurePath, selector string) ([]ACFile, error) {
	acsDir := filepath.Join(r.specRoot, "features", filepath.FromSlash(featurePath), "acs")
	if selector == "*" {
		return r.resolveAll(acsDir)
	}
	return r.resolveSpecific(acsDir, selector)
}

func (r *ACResolver) resolveAll(acsDir string) ([]ACFile, error) {
	entries, err := os.ReadDir(acsDir)
	if err != nil {
		return nil, fmt.Errorf("reading acs directory %s: %w", acsDir, err)
	}
	var acs []ACFile
	var slugs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slugs = append(slugs, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(slugs) // alphabetical order for wildcard
	for _, slug := range slugs {
		ac, err := r.readACFile(acsDir, slug)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (r *ACResolver) resolveSpecific(acsDir, selector string) ([]ACFile, error) {
	// selector can be a single slug or comma-separated
	slugs := strings.Split(selector, ",")
	var acs []ACFile
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		// Strip markdown link syntax if present: [slug](path) → slug
		if idx := strings.Index(slug, "]"); idx > 0 && slug[0] == '[' {
			slug = slug[1:idx]
		}
		ac, err := r.readACFile(acsDir, slug)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (r *ACResolver) readACFile(acsDir, slug string) (ACFile, error) {
	path := filepath.Join(acsDir, slug+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ACFile{}, fmt.Errorf("reading AC file %s: %w", path, err)
	}
	return ParseACFile(data, slug)
}

// ParseACFile parses an AC markdown file.
func ParseACFile(data []byte, slug string) (ACFile, error) {
	ac := ACFile{Slug: slug}
	lines := strings.Split(string(data), "\n")

	// Parse **Status:** line → ac.Status
	// Parse **Feature:** line → ac.FeaturePath (extract from markdown link text)
	// Find "## Description" section → ac.Description (text until next ##)
	// Find "## Inputs" section → parse table rows into []ACInput
	//   Each row: Name | Required (Yes/No) | Description
	// Find "## Verification" section → extract ```bash code block → ac.Verification
	// Ignore "## Scenarios" section (back-references, not needed for execution)

	_ = lines
	return ac, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -run "TestParseACFile|TestResolveACs" -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go test ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/ac.go pkg/testscenario/ac_test.go
git commit -m "feat(testscenario): add AC file parser and resolver"
```

---

## Chunk 3: Runner and Include Resolution

### Task 5: Build the step runner

**Files:**
- Create: `pkg/testscenario/runner.go`
- Create: `pkg/testscenario/runner_test.go`

- [ ] **Step 1: Write failing runner tests**

Tests for:
1. Run a simple single-step scenario → captures stdout, exit code 0, passes
2. Run a step that fails (exit code 1) → step marked failed
3. Run scenario with Setup and Teardown → both execute
4. Teardown runs even when a step fails
5. Run scenario with context output → downstream step sees the value
6. Run two parallel steps → both execute (order may vary)
7. Run step with AC verification (using mock AC files on disk) → AC results populated

```go
package testscenario

// Features implemented: test-scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunner_singleStep(t *testing.T) {
	s := &Scenario{
		Title: "simple",
		Steps: []Step{{Name: "echo-test", Code: "echo hello"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if len(result.StepResults) != 1 || !result.StepResults[0].Passed {
		t.Errorf("step failed: %+v", result.StepResults)
	}
}

func TestRunner_failingStep(t *testing.T) {
	s := &Scenario{
		Title: "fail",
		Steps: []Step{{Name: "bad", Code: "exit 1"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if result.Passed {
		t.Error("expected scenario to fail")
	}
	if result.StepResults[0].ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.StepResults[0].ExitCode)
	}
}

func TestRunner_setupAndTeardown(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "teardown-ran")
	s := &Scenario{
		Title:    "lifecycle",
		Setup:    "export MARKER=" + marker,
		Teardown: "touch " + marker,
		Steps:    []Step{{Name: "noop", Code: "echo ok"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	_ = r.Run(s)
	if _, err := os.Stat(marker); err != nil {
		t.Error("teardown did not run")
	}
}

func TestRunner_teardownRunsOnFailure(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "teardown-ran")
	s := &Scenario{
		Title:    "fail-teardown",
		Teardown: "touch " + marker,
		Steps:    []Step{{Name: "fail", Code: "exit 1"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	_ = r.Run(s)
	if _, err := os.Stat(marker); err != nil {
		t.Error("teardown did not run after failure")
	}
}

func TestRunner_contextOutputPassthrough(t *testing.T) {
	s := &Scenario{
		Title: "context",
		Steps: []Step{
			{
				Name: "produce",
				Code: "echo myvalue",
				Outputs: []Output{{Name: "val", Store: StoreContext, Extract: "cat $STEP_STDOUT"}},
			},
			{
				Name: "consume",
				Code: "echo got-${{ context.val }}",
			},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if result.StepResults[1].Stdout != "got-myvalue" {
		t.Errorf("stdout = %q, want %q", result.StepResults[1].Stdout, "got-myvalue")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -run TestRunner -count=1`
Expected: FAIL — `NewRunner` undefined.

- [ ] **Step 3: Implement `runner.go`**

The runner:
1. Creates `ExecContext`
2. Runs Setup in a shell
3. Iterates steps in order. For parallel groups (consecutive `Parallel: true` steps), launches goroutines and waits with `sync.WaitGroup`
4. For each step: resolves `${{ }}` references in code, executes via `os/exec`, captures stdout/stderr to temp files, extracts outputs, resolves and runs ACs
5. Runs Teardown in a deferred block

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// RunnerConfig holds configuration for the test runner.
type RunnerConfig struct {
	SpecRoot string // path to the spec root directory
}

// Runner executes test scenarios.
type Runner struct {
	config     RunnerConfig
	acResolver *ACResolver
}

// NewRunner creates a new test runner.
func NewRunner(config RunnerConfig) *Runner {
	return &Runner{
		config:     config,
		acResolver: NewACResolver(config.SpecRoot),
	}
}

// Run executes a scenario and returns the result.
func (r *Runner) Run(s *Scenario) ScenarioResult {
	result := ScenarioResult{ScenarioTitle: s.Title, Passed: true}
	ctx := NewExecContext()

	// 1. Run Setup block (if present) via execBash(s.Setup, env)
	//    If fails → set result.SetupError, result.Passed = false, skip to teardown

	// 2. Group steps into sequential steps and parallel groups
	//    Walk s.Steps: consecutive Parallel=true steps form a group

	// 3. For each group:
	//    Sequential (single step): call r.runStep(step, ctx) → append StepResult
	//    Parallel group: launch goroutines per step, collect with sync.WaitGroup
	//    If any step fails: set result.Passed = false, continue (don't abort remaining steps)

	// 4. For each step in runStep:
	//    a. Resolve ${{ }} references in Code via ctx.ResolveString()
	//    b. Execute via execBash() → capture stdout/stderr to temp files, get exit code
	//    c. If exit code != 0 → step failed
	//    d. Extract outputs: for each Output, run Extract expression via execBash()
	//       with STEP_STDOUT, STEP_STDERR, STEP_EXIT_CODE env vars
	//       Store result via ctx.StoreOutput()
	//    e. Resolve ACs: for each ACRef, call r.acResolver.Resolve()
	//       Run each AC's Verification script with context vars + step outputs as env vars
	//       If AC fails → step fails

	// 5. defer: Run Teardown block (always, even on panic/failure)
	//    If fails → set result.TeardownError

	_ = ctx
	return result
}

// execBash runs a bash script string and returns stdout, stderr, exit code.
func execBash(script string, env []string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	exitCode = cmd.ProcessState.ExitCode()
	return strings.TrimRight(outBuf.String(), "\n"), errBuf.String(), exitCode, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -run TestRunner -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go test ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/runner.go pkg/testscenario/runner_test.go
git commit -m "feat(testscenario): add step runner with sequential/parallel execution"
```

---

### Task 6: Build include/sub-flow resolution

**Files:**
- Create: `pkg/testscenario/include.go`
- Create: `pkg/testscenario/include_test.go`

- [ ] **Step 1: Write failing include tests**

Tests for:
1. Resolve a simple include → parses the referenced file and returns its scenario
2. Recursive include → error (cycle detected)
3. Include file not found → error
4. Nested include (A includes B, B includes C) → resolves correctly

```go
package testscenario

// Features implemented: test-scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInclude_simple(t *testing.T) {
	dir := t.TempDir()
	flowContent := "# Scenario: Sub-flow\n\n## sub-step\n\n```bash\necho sub\n```"
	if err := os.MkdirAll(filepath.Join(dir, "flows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flows", "sub.md"), []byte(flowContent), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver()
	scenario, err := resolver.Resolve(filepath.Join(dir, "flows", "sub.md"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenario.Steps) != 1 || scenario.Steps[0].Name != "sub-step" {
		t.Errorf("steps = %+v", scenario.Steps)
	}
}

func TestResolveInclude_circular(t *testing.T) {
	dir := t.TempDir()
	// a.md includes b.md, b.md includes a.md
	aContent := "# Scenario: A\n\n## step-a\n\n**Include:** [b.md](b.md)\n"
	bContent := "# Scenario: B\n\n## step-b\n\n**Include:** [a.md](a.md)\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(aContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte(bContent), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver()
	_, err := resolver.Resolve(filepath.Join(dir, "a.md"), nil)
	if err == nil {
		t.Fatal("expected error for circular include")
	}
}

func TestResolveInclude_notFound(t *testing.T) {
	resolver := NewIncludeResolver()
	_, err := resolver.Resolve("/nonexistent/flow.md", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -run TestResolveInclude -count=1`
Expected: FAIL — `NewIncludeResolver` undefined.

- [ ] **Step 3: Implement `include.go`**

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"os"
)

// IncludeResolver resolves sub-flow includes with cycle detection.
type IncludeResolver struct{}

// NewIncludeResolver creates a new include resolver.
func NewIncludeResolver() *IncludeResolver {
	return &IncludeResolver{}
}

// Resolve reads and parses an included scenario file. The seen set tracks
// visited paths for cycle detection. Pass nil for the initial call.
func (r *IncludeResolver) Resolve(path string, seen map[string]bool) (*Scenario, error) {
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[path] {
		return nil, fmt.Errorf("circular include detected: %s", path)
	}
	seen[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading include %s: %w", path, err)
	}
	return ParseScenario(data)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -run TestResolveInclude -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go test ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/include.go pkg/testscenario/include_test.go
git commit -m "feat(testscenario): add include resolver with cycle detection"
```

---

## Chunk 4: Reporter, CLI Commands, and Dogfooding

### Task 7: Build the reporter

**Files:**
- Create: `pkg/testscenario/reporter.go`
- Create: `pkg/testscenario/reporter_test.go`

- [ ] **Step 1: Write failing reporter tests**

Tests for:
1. Format a passing scenario result → contains "PASS" and scenario title
2. Format a failing scenario result → contains "FAIL", step name, and error
3. Format AC results → each AC listed with pass/fail

```go
package testscenario

// Features implemented: test-scenario

import (
	"strings"
	"testing"
)

func TestFormatResult_passing(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "My Test",
		Passed:        true,
		StepResults: []StepResult{
			{StepName: "step-a", Passed: true},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "My Test") {
		t.Errorf("output = %q", out)
	}
}

func TestFormatResult_failing(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "My Test",
		Passed:        false,
		StepResults: []StepResult{
			{StepName: "bad-step", Passed: false, Error: "exit code 1"},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "FAIL") || !strings.Contains(out, "bad-step") {
		t.Errorf("output = %q", out)
	}
}

func TestFormatResult_withACResults(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "AC Test",
		Passed:        false,
		StepResults: []StepResult{
			{
				StepName: "remove",
				Passed:   false,
				ACResults: []ACResult{
					{FeaturePath: "cli/project/remove", ACSlug: "not-in-list", Passed: true},
					{FeaturePath: "cli/project/remove", ACSlug: "recreate", Passed: false, Error: "assertion failed"},
				},
			},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "not-in-list") || !strings.Contains(out, "recreate") {
		t.Errorf("output = %q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/testscenario/... -v -run TestFormatResult -count=1`
Expected: FAIL — `FormatResult` undefined.

- [ ] **Step 3: Implement `reporter.go`**

A simple text reporter that outputs a structured summary. Format:

```
=== Scenario: My Test ===
  ✓ step-a
  ✗ bad-step: exit code 1
    ✓ AC cli/project/remove/not-in-list
    ✗ AC cli/project/remove/recreate: assertion failed

FAIL (1/2 steps passed)
```

```go
package testscenario

// Features implemented: test-scenario

import (
	"fmt"
	"strings"
)

// FormatResult formats a ScenarioResult as human-readable text.
func FormatResult(r ScenarioResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Scenario: %s ===\n", r.ScenarioTitle)
	if r.SetupError != "" {
		fmt.Fprintf(&b, "  ✗ Setup: %s\n", r.SetupError)
	}
	passed, total := 0, len(r.StepResults)
	for _, sr := range r.StepResults {
		if sr.Passed {
			fmt.Fprintf(&b, "  ✓ %s\n", sr.StepName)
			passed++
		} else {
			fmt.Fprintf(&b, "  ✗ %s: %s\n", sr.StepName, sr.Error)
		}
		for _, ac := range sr.ACResults {
			if ac.Passed {
				fmt.Fprintf(&b, "    ✓ AC %s/%s\n", ac.FeaturePath, ac.ACSlug)
			} else {
				fmt.Fprintf(&b, "    ✗ AC %s/%s: %s\n", ac.FeaturePath, ac.ACSlug, ac.Error)
			}
		}
	}
	if r.TeardownError != "" {
		fmt.Fprintf(&b, "  ✗ Teardown: %s\n", r.TeardownError)
	}
	if r.Passed {
		fmt.Fprintf(&b, "\nPASS (%d/%d steps passed)\n", passed, total)
	} else {
		fmt.Fprintf(&b, "\nFAIL (%d/%d steps passed)\n", passed, total)
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/testscenario/... -v -run TestFormatResult -count=1`
Expected: All PASS.

- [ ] **Step 5: Run full Go validation**

Run: `gofmt -w pkg/testscenario/*.go && golangci-lint run ./pkg/testscenario/... && go test ./pkg/testscenario/... && go build ./pkg/testscenario/... && go vet ./pkg/testscenario/...`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/testscenario/reporter.go pkg/testscenario/reporter_test.go
git commit -m "feat(testscenario): add text result reporter"
```

---

### Task 8: Wire up CLI commands

**Files:**
- Create: `cli/test/README.md`
- Create: `cli/test/test.go`
- Create: `cli/test/run.go`
- Create: `cli/test/list.go`
- Modify: `cli/main.go` — add `test` command group

- [ ] **Step 1: Create `cli/test/README.md`**

```markdown
# cli/test

CLI commands for running test scenarios.

| Command | Description |
|---|---|
| `synchestra test run` | Run test scenarios |
| `synchestra test list` | List available test scenarios |

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create `cli/test/test.go`**

```go
package test

// Features implemented: cli/test

import "github.com/spf13/cobra"

// Command returns the `synchestra test` command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run and manage test scenarios",
	}
	cmd.AddCommand(
		runCommand(),
		listCommand(),
	)
	return cmd
}
```

- [ ] **Step 3: Create `cli/test/run.go`**

```go
package test

// Features implemented: cli/test
// Features depended on:  test-scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/pkg/testscenario"
)

func runCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [path]",
		Short: "Run test scenario files",
		Long:  "Run one or more test scenario .md files. Pass a file path or directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
	cmd.Flags().StringSlice("tag", nil, "filter scenarios by tag")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	specRoot := "spec" // TODO: read from synchestra-spec.yaml project_dirs.specifications
	tags, _ := cmd.Flags().GetStringSlice("tag")

	// Determine target: file or directory
	target := specRoot + "/tests"
	if len(args) > 0 {
		target = args[0]
	}

	// Collect .md files (single file or walk directory)
	files, err := collectScenarioFiles(target)
	if err != nil {
		return fmt.Errorf("collecting scenarios: %w", err)
	}

	anyFailed := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}
		scenario, err := testscenario.ParseScenario(data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", f, err)
		}
		if len(tags) > 0 && !matchesTags(scenario.Tags, tags) {
			continue
		}
		runner := testscenario.NewRunner(testscenario.RunnerConfig{SpecRoot: specRoot})
		result := runner.Run(scenario)
		fmt.Fprint(cmd.OutOrStdout(), testscenario.FormatResult(result))
		if !result.Passed {
			anyFailed = true
		}
	}
	if anyFailed {
		return fmt.Errorf("one or more scenarios failed")
	}
	return nil
}

// collectScenarioFiles returns all .md files under a path (or the path itself if a file).
func collectScenarioFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	var files []string
	return files, filepath.Walk(path, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !i.IsDir() && strings.HasSuffix(p, ".md") && i.Name() != "README.md" {
			files = append(files, p)
		}
		return nil
	})
}

func matchesTags(scenarioTags, filterTags []string) bool {
	tagSet := make(map[string]bool, len(scenarioTags))
	for _, t := range scenarioTags {
		tagSet[t] = true
	}
	for _, ft := range filterTags {
		if tagSet[ft] {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Create `cli/test/list.go`**

```go
package test

// Features implemented: cli/test
// Features depended on:  test-scenario

import (
	"github.com/spf13/cobra"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available test scenarios",
		RunE:  runList,
	}
	cmd.Flags().StringSlice("tag", nil, "filter scenarios by tag")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	specRoot := "spec" // TODO: read from synchestra-spec.yaml
	tags, _ := cmd.Flags().GetStringSlice("tag")

	// Collect from both spec/tests/ and spec/features/*/_tests/
	var allFiles []string
	for _, dir := range []string{
		filepath.Join(specRoot, "tests"),
	} {
		files, err := collectScenarioFiles(dir)
		if err != nil {
			continue // directory may not exist
		}
		allFiles = append(allFiles, files...)
	}
	// Also walk feature _tests/ directories
	featuresDir := filepath.Join(specRoot, "features")
	_ = filepath.Walk(featuresDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "_tests" {
			files, _ := collectScenarioFiles(p)
			allFiles = append(allFiles, files...)
		}
		return nil
	})

	fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n", "SCENARIO", "DESCRIPTION", "TAGS")
	for _, f := range allFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		scenario, err := testscenario.ParseScenario(data)
		if err != nil {
			continue
		}
		if len(tags) > 0 && !matchesTags(scenario.Tags, tags) {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n",
			f, scenario.Description, strings.Join(scenario.Tags, ", "))
	}
	return nil
}
```

- [ ] **Step 5: Add test command to `cli/main.go`**

Add `testcmd "github.com/synchesta-io/synchestra/cli/test"` import and `testcmd.Command()` to `rootCmd.AddCommand(...)`.

- [ ] **Step 6: Run full Go validation**

Run: `gofmt -w cli/test/*.go cli/main.go && golangci-lint run ./... && go test ./... && go build ./... && go vet ./...`
Expected: All pass.

- [ ] **Step 7: Commit**

```bash
git add cli/test/ cli/main.go
git commit -m "feat(cli): add synchestra test run and test list commands"
```

---

### Task 9: Update feature spec and create feature spec for test-scenario

**Files:**
- Create: `spec/features/test-scenario/README.md`
- Modify: `spec/features/feature/README.md` — add Acceptance Criteria to required sections, add `acs/` and `_tests/` to directory structure, formalize `_` prefix convention
- Modify: `spec/features/README.md` — add test-scenario to feature index
- Create: `spec/tests/README.md`

- [ ] **Step 1: Create `spec/features/test-scenario/README.md`**

The feature spec for the test scenario system, following the established feature README pattern. Includes:
- Status: In Progress
- Summary, Problem, Behavior sections based on the design spec
- Acceptance Criteria section (initially "Not defined yet." with Outstanding Question)
- Outstanding Questions section

- [ ] **Step 2: Update `spec/features/feature/README.md`**

Add to Required sections table:
- Acceptance Criteria: Yes — always present, states "Not defined yet." if empty

Add to Behavior → Feature location section the `acs/` and `_tests/` directories.

Add the `_` prefix convention:
> Directories prefixed with `_` are reserved for Synchestra tooling and are not sub-features. They are excluded from the feature index and Contents table.

- [ ] **Step 3: Update `spec/features/README.md`**

Add `test-scenario` to the feature index table.

- [ ] **Step 4: Create `spec/tests/README.md`**

```markdown
# Tests

Cross-feature end-to-end test scenarios.

| Scenario | Description | Tags |
|---|---|---|
| (none yet) | | |

## Outstanding Questions

None at this time.
```

- [ ] **Step 5: Commit**

```bash
git add spec/features/test-scenario/ spec/features/feature/README.md spec/features/README.md spec/tests/
git commit -m "feat(spec): add test-scenario feature, update feature spec with AC requirements"
```

---

### Task 10: Dogfood — Write initial ACs and the project lifecycle E2E scenario

**Files:**
- Create: `spec/features/cli/project/new/acs/creates-spec-config.md`
- Create: `spec/features/cli/project/new/acs/creates-state-config.md`
- Create: `spec/tests/project-lifecycle.md`
- Modify: `spec/features/cli/project/new/README.md` — add Acceptance Criteria section with table
- Create: `spec/tests/flows/README.md`

- [ ] **Step 1: Create AC files for `cli/project/new`**

Create `spec/features/cli/project/new/acs/creates-spec-config.md`:

```markdown
# AC: creates-spec-config

**Status:** implemented
**Feature:** [cli/project/new](../README.md)

## Description

After `synchestra project new`, `synchestra-spec.yaml` exists in the spec repo
with the correct title and state_repo fields.

## Inputs

| Name | Required | Description |
|---|---|---|
| spec_repo_path | Yes | Path to the spec repository |
| expected_title | Yes | Expected project title |

## Verification

` ``bash
test -f "$spec_repo_path/synchestra-spec.yaml"
title=$(grep 'title:' "$spec_repo_path/synchestra-spec.yaml" | head -1 | sed 's/title: *//')
test "$title" = "$expected_title"
` ``

## Scenarios

(None yet.)
```

Create `spec/features/cli/project/new/acs/creates-state-config.md`:

```markdown
# AC: creates-state-config

**Status:** implemented
**Feature:** [cli/project/new](../README.md)

## Description

After `synchestra project new`, `synchestra-state.yaml` exists in the state repo
with the spec_repo field pointing to the spec repo.

## Inputs

| Name | Required | Description |
|---|---|---|
| state_repo_path | Yes | Path to the state repository |

## Verification

` ``bash
test -f "$state_repo_path/synchestra-state.yaml"
grep -q 'spec_repo:' "$state_repo_path/synchestra-state.yaml"
` ``

## Scenarios

(None yet.)
```

(Note: in the actual files, the code fences use three backticks without the space shown here.)

- [ ] **Step 2: Update `spec/features/cli/project/new/README.md`**

Add an Acceptance Criteria section before the Outstanding Questions section:

```markdown
## Acceptance Criteria

| AC | Description | Status |
|---|---|---|
| [creates-spec-config](acs/creates-spec-config.md) | synchestra-spec.yaml created in spec repo | implemented |
| [creates-state-config](acs/creates-state-config.md) | synchestra-state.yaml created in state repo | implemented |
```

- [ ] **Step 3: Create `spec/tests/project-lifecycle.md`**

```markdown
# Scenario: Project lifecycle

**Description:** End-to-end test of creating a Synchestra project and verifying config files.
**Tags:** e2e, cli, project

## Setup

` ``bash
export TEST_DIR=$(mktemp -d)
export SPEC_BARE=$(mktemp -d)/spec.git
export STATE_BARE=$(mktemp -d)/state.git
export TARGET_BARE=$(mktemp -d)/target.git
git init --bare "$SPEC_BARE"
git init --bare "$STATE_BARE"
git init --bare "$TARGET_BARE"
# Seed spec repo with a README
SEED=$(mktemp -d)
git clone "$SPEC_BARE" "$SEED/spec"
cd "$SEED/spec" && git config user.email "test@test" && git config user.name "Test"
echo "# Test Project" > README.md && git add . && git commit -m "init" && git push origin HEAD
cd -
export HOME="$TEST_DIR"
` ``

## create-project

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| spec_repo_path | context | `echo $HOME/synchestra-repos/spec` |
| state_repo_path | context | `echo $HOME/synchestra-repos/state` |
| expected_title | context | `echo "Test Project"` |

` ``bash
synchestra project new \
  --spec-repo "$SPEC_BARE" \
  --state-repo "$STATE_BARE" \
  --target-repo "$TARGET_BARE"
` ``

## verify-configs

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/new](spec/features/cli/project/new/) | * |

` ``bash
echo "Verifying config files exist"
` ``

## Teardown

` ``bash
rm -rf "$TEST_DIR" "$SPEC_BARE" "$STATE_BARE" "$TARGET_BARE"
` ``
```

(Note: in the actual file, code fences use three backticks without the space shown here.)

- [ ] **Step 4: Create `spec/tests/flows/README.md`**

```markdown
# Flows

Reusable sub-flow scenario files for cross-feature E2E tests.

(No flows defined yet.)

## Outstanding Questions

None at this time.
```

- [ ] **Step 5: Run the scenario**

Run: `go run . test run spec/tests/project-lifecycle.md`
Expected: Scenario runs and reports results. May need debugging — this is the first real end-to-end execution.

- [ ] **Step 6: Commit**

```bash
git add spec/features/cli/project/new/acs/ spec/features/cli/project/new/README.md spec/tests/
git commit -m "feat(dogfood): add initial ACs for project new and CLI lifecycle E2E scenario"
```
