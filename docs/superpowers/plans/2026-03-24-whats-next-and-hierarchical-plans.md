# What's Next & Hierarchical Plans Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add hierarchical plans (roadmaps), optional ROI metadata, and an AI-generated WHATS-NEXT.md prioritization report to Synchestra.

**Architecture:** Three layers of change: (1) spec updates to the development-plan feature spec documenting the new conventions, (2) lint rules enforcing hierarchy and metadata validity, (3) config extension for the `whats-next` setting. The WHATS-NEXT.md report generation is an AI skill (not Go code) — it reads spec/plan state and writes markdown.

**Tech Stack:** Go (cobra CLI, spec-lint checkers), YAML config, Markdown conventions, AI skills (SKILL.md)

---

### Task 1: Update development-plan feature spec

Add hierarchical plans, ROI metadata, and WHATS-NEXT.md conventions to the existing feature spec at `spec/features/development-plan/README.md`.

**Files:**
- Modify: `spec/features/development-plan/README.md`

**Reference:** Design spec at `docs/superpowers/specs/2026-03-24-whats-next-and-hierarchical-plans-design.md`

- [ ] **Step 1: Read the design spec and the current feature spec**

Read both files end-to-end to understand what sections exist and what needs to change.

- [ ] **Step 2: Add "Plan hierarchy" section after "Nesting limit"**

Add a new `### Plan hierarchy` section after `### Nesting limit` (line ~216) documenting roadmap plans:

```markdown
### Plan hierarchy

Plans support nesting to express roadmap-level groupings, mirroring the feature hierarchy:

\```text
spec/plans/
  README.md                          ← index
  chat-feature/
    README.md                        ← roadmap plan (parent)
    chat-infrastructure/
      README.md                      ← child plan
    chat-workflow-engine/
      README.md                      ← child plan
  e2e-testing-framework/
    README.md                        ← standalone plan (no children)
\```

A **roadmap** (parent plan) defines ordering and dependencies between child plans. It does not have implementation steps — its Steps section is replaced by a **Child Plans** section listing child plans with their relationships. A **child plan** follows the standard plan format with steps, task mappings, and acceptance criteria. A **standalone plan** (no children) works exactly as before.

Nesting is limited to **two levels**: roadmap → child plan. Deeper nesting belongs in task decomposition.

#### Roadmap document structure

A roadmap uses the standard plan header fields plus optional ROI metadata (see [Optional ROI metadata](#optional-roi-metadata)). Instead of a Steps section, it has a Child Plans section:

\```markdown
## Child Plans

| Order | Plan | Status | Effort | Impact |
|-------|------|--------|--------|--------|
| 1 | [chat-infrastructure](chat-infrastructure/) | draft | L | high |
| 2 | [chat-workflow-engine](chat-workflow-engine/) | draft | M | high |
\```

Table order defines the recommended execution sequence. Child plans may declare explicit dependencies between each other using `Depends on` in their headers.

#### Roadmap status derivation

A roadmap's status is derived from its children:

| Derived status | Condition |
|---|---|
| `draft` | At least one child is `draft` |
| `in_review` | All children are `in_review` or `approved` |
| `approved` | All children are `approved` |
| `in_progress` | At least one child plan has linked tasks in progress |
| `superseded` | Explicitly set when the roadmap is replaced |
```

- [ ] **Step 3: Add "Optional ROI metadata" section after plan hierarchy**

Add a new `### Optional ROI metadata` section:

```markdown
### Optional ROI metadata

Plan headers support two optional fields for prioritization:

\```markdown
**Effort:** M
**Impact:** high
\```

| Field | Values | Description |
|---|---|---|
| **Effort** | `S`, `M`, `L`, `XL` | Rough estimate of work required |
| **Impact** | `low`, `medium`, `high`, `critical` | Expected value delivered |

Both fields are optional. When absent, AI tooling infers effort from step count and dependency depth, and impact from feature importance. During plan authoring, AI agents suggest values; the human author accepts, declines, or overwrites.

For roadmaps, effort/impact describe the aggregate. Child plans carry independent estimates.

| Effort | Rough meaning |
|--------|---------------|
| `S` | A few hours, 1-3 steps |
| `M` | A few days, 3-6 steps, limited dependencies |
| `L` | A week or more, 5-10 steps, cross-cutting |
| `XL` | Multi-week, many steps, multiple child plans or deep dependencies |

| Impact | Rough meaning |
|--------|---------------|
| `low` | Nice-to-have, no users blocked |
| `medium` | Improves existing capability |
| `high` | Enables important new capability |
| `critical` | Unblocks core functionality or other critical work |
```

- [ ] **Step 4: Update the header fields table**

Add Effort and Impact to the header fields table (around line 168):

```markdown
| **Effort** | No | Rough effort estimate: `S`, `M`, `L`, `XL` (see [Optional ROI metadata](#optional-roi-metadata)) |
| **Impact** | No | Expected impact: `low`, `medium`, `high`, `critical` (see [Optional ROI metadata](#optional-roi-metadata)) |
```

- [ ] **Step 5: Update the plans index section**

Update the plans index example (around line 288) to show hierarchy with indentation and the new Effort/Impact columns:

```markdown
| Plan | Status | Progress | Features | Effort | Impact | Author | Approved |
|---|---|---|---|---|---|---|---|
| [chat-feature](chat-feature/) | draft | — | chat, chat/workflow | XL | critical | @alex | — |
| &ensp;[chat-infrastructure](chat-feature/chat-infrastructure/) | draft | — | chat | L | high | @alex | — |
| &ensp;[chat-workflow-engine](chat-feature/chat-workflow-engine/) | draft | — | chat/workflow | M | high | @alex | — |
| [e2e-testing-framework](e2e-testing-framework/) | draft | — | testing-framework | — | — | @alex | — |
```

- [ ] **Step 6: Add "What's Next report" section before Project Configuration**

Add a new `### What's Next report` section documenting WHATS-NEXT.md:

```markdown
### What's Next report

`spec/plans/WHATS-NEXT.md` is an AI-generated prioritization report that surfaces what work is completed, in progress, and recommended next. It is generated by the `synchestra-whats-next` skill or the `synchestra plans whats-next` command (future).

The report is opt-in, controlled by the `planning.whats_next` setting (see [Project Configuration](#project-configuration)).

#### Report structure

\```markdown
# What's Next

**Generated:** 2026-03-24
**Mode:** incremental | full

## Completed Since Last Update

- [plan-slug](plan-slug/) — completed YYYY-MM-DD

## In Progress

- [plan-slug](plan-slug/) — N/M steps done, no blockers

## Recommended Next

1. **[plan-slug](plan-slug/)** — Impact: high, Effort: M. Reasoning.

### Reasoning

AI explanation of prioritization.

## Outstanding Questions

(ambiguities surfaced during analysis)
\```

#### Update mechanism

- **Incremental:** Reads previous WHATS-NEXT.md plus the completion delta. Regenerates only affected sections.
- **Full:** Scans all features, plans, and task statuses from scratch.
- The file is committed to git after each update.

#### Prioritization inputs

1. Explicit ROI metadata (effort/impact) when present
2. Dependency graph — what is newly unblocked
3. Momentum — preference for advancing roadmaps already in progress
4. Feature status — features closer to "stable" get a boost
5. AI inference from plan complexity when ROI metadata is absent
```

- [ ] **Step 7: Update Project Configuration section**

Add `whats_next` to the planning config block (around line 679):

```yaml
planning:
  auto_create: false
  auto_generate_tasks: false
  enforce_freeze: warn
  validate_artifacts: warn
  whats_next: disabled          # disabled | incremental | full (default: disabled)
```

Add a description row:

```markdown
| `whats_next` | `disabled` | When to regenerate `WHATS-NEXT.md`: `disabled` (never auto-generate), `incremental` (on completion events, using previous report + delta), `full` (on completion events, from scratch). The `synchestra plans whats-next` command works regardless of this setting. |
```

- [ ] **Step 8: Update feature linking in "Feature README back-reference"**

Update the example to show that features can reference both roadmaps and child plans:

```markdown
| [chat-feature](../../plans/chat-feature/) | draft | @alex | — |
| [chat-infrastructure](../../plans/chat-feature/chat-infrastructure/) | draft | @alex | — |
```

- [ ] **Step 9: Update "Interaction with Other Features" table**

Add a row for the What's Next report if not already covered.

- [ ] **Step 10: Run spec-lint to verify the updated spec is valid**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint`
Expected: No new violations in `spec/features/development-plan/README.md`

- [ ] **Step 11: Commit**

```bash
git add spec/features/development-plan/README.md
git commit -m "spec: add hierarchical plans, ROI metadata, and WHATS-NEXT.md to development-plan feature"
```

---

### Task 2: Extend SpecConfig with planning.whats_next

Add the `whats_next` config field to `SpecConfig` so it can be parsed from `synchestra-spec-repo.yaml`.

**Files:**
- Modify: `pkg/cli/project/configfiles.go`
- Test: `pkg/cli/project/configfiles_test.go`

- [ ] **Step 1: Write the failing test**

```go
// In pkg/cli/project/configfiles_test.go
func TestReadSpecConfig_PlanningWhatsNext(t *testing.T) {
	dir := t.TempDir()
	content := []byte("title: test\nplanning:\n  whats_next: incremental\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Planning == nil {
		t.Fatal("expected Planning to be non-nil")
	}
	if cfg.Planning.WhatsNext != "incremental" {
		t.Fatalf("expected WhatsNext=incremental, got %s", cfg.Planning.WhatsNext)
	}
}

func TestReadSpecConfig_PlanningWhatsNextDefault(t *testing.T) {
	dir := t.TempDir()
	content := []byte("title: test\n")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-spec-repo.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Planning section absent = nil, whats_next defaults to disabled
	whatsNext := cfg.WhatsNextMode()
	if whatsNext != "disabled" {
		t.Fatalf("expected default disabled, got %s", whatsNext)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/project/ -run TestReadSpecConfig_Planning -v`
Expected: FAIL — `Planning` field and `WhatsNextMode` method don't exist

- [ ] **Step 3: Add PlanningConfig struct and extend SpecConfig**

In `pkg/cli/project/configfiles.go`, add:

```go
// PlanningConfig holds planning-related settings from synchestra-spec-repo.yaml.
type PlanningConfig struct {
	WhatsNext string `yaml:"whats_next"`
}

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/project/ -run TestReadSpecConfig_Planning -v`
Expected: PASS

- [ ] **Step 5: Run full test suite to check for regressions**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/project/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/cli/project/configfiles.go pkg/cli/project/configfiles_test.go
git commit -m "feat: add planning.whats_next config field to SpecConfig"
```

---

### Task 3: Add plan-hierarchy lint rule

Add a lint checker that validates hierarchical plan conventions: roadmaps must not have Steps sections, child plans must have Steps sections, and nesting is limited to 2 levels.

**Files:**
- Create: `pkg/cli/spec/plan_hierarchy.go`
- Create: `pkg/cli/spec/plan_hierarchy_test.go`
- Modify: `pkg/cli/spec/linter.go` (register the new checker)

- [ ] **Step 1: Write the failing test**

```go
// In pkg/cli/spec/plan_hierarchy_test.go
package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanHierarchyChecker_RoadmapWithSteps(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/my-roadmap/README.md": "# Plan: My Roadmap\n\n**Status:** draft\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/my-roadmap/child-plan/README.md": "# Plan: Child Plan\n\n**Status:** draft\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanHierarchyChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Roadmap has Steps section — should be a violation
	found := false
	for _, v := range violations {
		if v.Rule == "plan-hierarchy" && contains(v.Message, "Steps") {
			found = true
		}
	}
	if !found {
		t.Error("expected violation for roadmap with Steps section")
	}
}

func TestPlanHierarchyChecker_ThreeLevelNesting(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/roadmap/README.md": "# Plan: Roadmap\n\n**Status:** draft\n\n## Child Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/roadmap/child/README.md": "# Plan: Child\n\n**Status:** draft\n\n## Steps\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/roadmap/child/grandchild/README.md": "# Plan: Grandchild\n\n**Status:** draft\n\n## Steps\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanHierarchyChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, v := range violations {
		if v.Rule == "plan-hierarchy" && contains(v.Message, "nesting") {
			found = true
		}
	}
	if !found {
		t.Error("expected violation for three-level nesting")
	}
}

func TestPlanHierarchyChecker_ValidHierarchy(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/roadmap/README.md": "# Plan: Roadmap\n\n**Status:** draft\n\n## Child Plans\n\n| Order | Plan |\n|---|---|\n| 1 | [child](child/) |\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/roadmap/child/README.md": "# Plan: Child\n\n**Status:** draft\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/standalone/README.md": "# Plan: Standalone\n\n**Status:** draft\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanHierarchyChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, v := range violations {
		if v.Rule == "plan-hierarchy" {
			t.Errorf("unexpected violation: %s in %s", v.Message, v.File)
		}
	}
}

func setupPlanHierarchyFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -run TestPlanHierarchy -v`
Expected: FAIL — `newPlanHierarchyChecker` doesn't exist

- [ ] **Step 3: Implement plan_hierarchy.go**

Create `pkg/cli/spec/plan_hierarchy.go`:

```go
package spec

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type planHierarchyChecker struct{}

func newPlanHierarchyChecker() checker {
	return &planHierarchyChecker{}
}

func (c *planHierarchyChecker) name() string     { return "plan-hierarchy" }
func (c *planHierarchyChecker) severity() string { return "error" }

func (c *planHierarchyChecker) check(specRoot string) ([]Violation, error) {
	var violations []Violation

	plansDir := filepath.Join(specRoot, "plans")
	info, err := os.Stat(plansDir)
	if err != nil || !info.IsDir() {
		return violations, nil
	}

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		planDir := filepath.Join(plansDir, entry.Name())
		v, err := c.checkPlanDir(specRoot, planDir, 1)
		if err != nil {
			return nil, err
		}
		violations = append(violations, v...)
	}

	return violations, nil
}

func (c *planHierarchyChecker) checkPlanDir(specRoot, planDir string, depth int) ([]Violation, error) {
	var violations []Violation

	readmePath := filepath.Join(planDir, "README.md")
	relPath, _ := filepath.Rel(specRoot, readmePath)

	if _, err := os.Stat(readmePath); err != nil {
		return violations, nil
	}

	// Find child plan directories (subdirs that contain README.md)
	children := findChildPlanDirs(planDir)
	isRoadmap := len(children) > 0

	hasSteps := hasSection(readmePath, "## Steps")
	hasChildPlans := hasSection(readmePath, "## Child Plans")

	if isRoadmap && hasSteps {
		violations = append(violations, Violation{
			File:     relPath,
			Severity: "error",
			Rule:     "plan-hierarchy",
			Message:  "Roadmap plan must not have a Steps section — use Child Plans instead",
		})
	}

	if !isRoadmap && hasChildPlans {
		violations = append(violations, Violation{
			File:     relPath,
			Severity: "warning",
			Rule:     "plan-hierarchy",
			Message:  "Plan has Child Plans section but no child plan directories",
		})
	}

	// Check nesting depth
	for _, childDir := range children {
		childDepth := depth + 1
		if childDepth > 2 {
			childReadme := filepath.Join(childDir, "README.md")
			childRel, _ := filepath.Rel(specRoot, childReadme)
			violations = append(violations, Violation{
				File:     childRel,
				Severity: "error",
				Rule:     "plan-hierarchy",
				Message:  "Plan nesting exceeds 2 levels (roadmap → plan); deeper nesting belongs in task decomposition",
			})
			continue
		}

		v, err := c.checkPlanDir(specRoot, childDir, childDepth)
		if err != nil {
			return nil, err
		}
		violations = append(violations, v...)
	}

	return violations, nil
}

// findChildPlanDirs returns subdirectories that contain a README.md (i.e., are plans).
// Skips directories named "acs" and "reports" which are plan support dirs, not child plans.
func findChildPlanDirs(planDir string) []string {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil
	}

	skipDirs := map[string]bool{"acs": true, "reports": true}
	var children []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || skipDirs[entry.Name()] {
			continue
		}
		childReadme := filepath.Join(planDir, entry.Name(), "README.md")
		if _, err := os.Stat(childReadme); err == nil {
			children = append(children, filepath.Join(planDir, entry.Name()))
		}
	}
	return children
}

// hasSection checks if a README contains a specific heading.
func hasSection(readmePath, heading string) bool {
	file, err := os.Open(readmePath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), heading) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Register the checker in linter.go**

In `pkg/cli/spec/linter.go`, add to `newLinter()`:

```go
l.registerChecker(newPlanHierarchyChecker())
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -run TestPlanHierarchy -v`
Expected: PASS

- [ ] **Step 6: Run full lint test suite**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/cli/spec/plan_hierarchy.go pkg/cli/spec/plan_hierarchy_test.go pkg/cli/spec/linter.go
git commit -m "feat: add plan-hierarchy lint rule for roadmap/nesting validation"
```

---

### Task 4: Add plan-roi-metadata lint rule

Add a lint checker that validates ROI metadata values when present (Effort must be S/M/L/XL, Impact must be low/medium/high/critical).

**Files:**
- Create: `pkg/cli/spec/plan_roi.go`
- Create: `pkg/cli/spec/plan_roi_test.go`
- Modify: `pkg/cli/spec/linter.go` (register the new checker)

- [ ] **Step 1: Write the failing test**

```go
// In pkg/cli/spec/plan_roi_test.go
package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanROIChecker_InvalidEffort(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/my-plan/README.md": "# Plan: My Plan\n\n**Status:** draft\n**Effort:** huge\n**Impact:** high\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanROIChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, v := range violations {
		if v.Rule == "plan-roi-metadata" && strings.Contains(v.Message, "Effort") {
			found = true
		}
	}
	if !found {
		t.Error("expected violation for invalid Effort value")
	}
}

func TestPlanROIChecker_ValidMetadata(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/my-plan/README.md": "# Plan: My Plan\n\n**Status:** draft\n**Effort:** M\n**Impact:** high\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanROIChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, v := range violations {
		if v.Rule == "plan-roi-metadata" {
			t.Errorf("unexpected violation: %s", v.Message)
		}
	}
}

func TestPlanROIChecker_NoMetadata(t *testing.T) {
	dir := setupPlanHierarchyFixture(t, map[string]string{
		"spec/plans/README.md": "# Plans\n\n## Outstanding Questions\n\nNone.",
		"spec/plans/my-plan/README.md": "# Plan: My Plan\n\n**Status:** draft\n\n## Steps\n\n### 1. Do something\n\n## Outstanding Questions\n\nNone.",
	})

	checker := newPlanROIChecker()
	violations, err := checker.check(filepath.Join(dir, "spec"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No ROI metadata = no violations (it's optional)
	for _, v := range violations {
		if v.Rule == "plan-roi-metadata" {
			t.Errorf("unexpected violation for absent metadata: %s", v.Message)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -run TestPlanROI -v`
Expected: FAIL — `newPlanROIChecker` doesn't exist

- [ ] **Step 3: Implement plan_roi.go**

Create `pkg/cli/spec/plan_roi.go`:

```go
package spec

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var (
	validEffort = map[string]bool{"S": true, "M": true, "L": true, "XL": true}
	validImpact = map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
)

type planROIChecker struct{}

func newPlanROIChecker() checker {
	return &planROIChecker{}
}

func (c *planROIChecker) name() string     { return "plan-roi-metadata" }
func (c *planROIChecker) severity() string { return "warning" }

func (c *planROIChecker) check(specRoot string) ([]Violation, error) {
	var violations []Violation

	plansDir := filepath.Join(specRoot, "plans")
	info, err := os.Stat(plansDir)
	if err != nil || !info.IsDir() {
		return violations, nil
	}

	err = filepath.Walk(plansDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "README.md" {
			return nil
		}
		// Skip the plans index README
		if filepath.Dir(path) == plansDir {
			return nil
		}

		v := c.checkPlanReadme(specRoot, path)
		violations = append(violations, v...)
		return nil
	})

	return violations, err
}

func (c *planROIChecker) checkPlanReadme(specRoot, readmePath string) []Violation {
	var violations []Violation
	relPath, _ := filepath.Rel(specRoot, readmePath)

	file, err := os.Open(readmePath)
	if err != nil {
		return violations
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Stop scanning after the header (first ## heading)
		if strings.HasPrefix(line, "## ") {
			break
		}

		if strings.HasPrefix(line, "**Effort:**") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "**Effort:**"))
			if value != "" && !validEffort[value] {
				violations = append(violations, Violation{
					File:     relPath,
					Line:     lineNum,
					Severity: "warning",
					Rule:     "plan-roi-metadata",
					Message:  "Invalid Effort value: " + value + " (valid: S, M, L, XL)",
				})
			}
		}

		if strings.HasPrefix(line, "**Impact:**") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "**Impact:**"))
			if value != "" && !validImpact[value] {
				violations = append(violations, Violation{
					File:     relPath,
					Line:     lineNum,
					Severity: "warning",
					Rule:     "plan-roi-metadata",
					Message:  "Invalid Impact value: " + value + " (valid: low, medium, high, critical)",
				})
			}
		}
	}

	return violations
}
```

- [ ] **Step 4: Register the checker in linter.go**

In `pkg/cli/spec/linter.go`, add to `newLinter()`:

```go
l.registerChecker(newPlanROIChecker())
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -run TestPlanROI -v`
Expected: PASS

- [ ] **Step 6: Run full lint test suite**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./pkg/cli/spec/ -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/cli/spec/plan_roi.go pkg/cli/spec/plan_roi_test.go pkg/cli/spec/linter.go
git commit -m "feat: add plan-roi-metadata lint rule for Effort/Impact validation"
```

---

### Task 5: Create synchestra-whats-next skill

Create an AI skill that generates/updates `spec/plans/WHATS-NEXT.md` by analyzing feature, plan, and task state.

**Files:**
- Create: `ai-plugin/skills/synchestra-whats-next/SKILL.md`
- Create: `ai-plugin/skills/synchestra-whats-next/README.md`

- [ ] **Step 1: Create the skill directory**

```bash
mkdir -p /Users/alexandertrakhimenok/projects/synchestra-io/synchestra/ai-plugin/skills/synchestra-whats-next
```

- [ ] **Step 2: Write SKILL.md**

Create `ai-plugin/skills/synchestra-whats-next/SKILL.md`:

```markdown
---
name: synchestra-whats-next
description: Generates or updates the WHATS-NEXT.md prioritization report for plans. Use when a plan or task is completed, or when the user wants to see what to work on next.
---

# Skill: synchestra-whats-next

Generate or update `spec/plans/WHATS-NEXT.md` — an AI-generated prioritization report that surfaces completed work, in-progress plans, and recommended next targets.

**CLI reference:** [development-plan feature spec](../../spec/features/development-plan/README.md)

## When to use

- After completing a plan or task (when `planning.whats_next` is `incremental` or `full`)
- When the user asks "what should we work on next?"
- When the user explicitly invokes this skill

## Modes

### Incremental (default)

1. Read the existing `spec/plans/WHATS-NEXT.md`
2. Determine what changed since the last generation (completed plans/tasks, status changes)
3. Update only the affected sections
4. Commit the updated file

### Full (--full or first-time generation)

1. Scan all features via `synchestra feature list --fields=status`
2. Scan all plans in `spec/plans/` — read each README.md for status, effort, impact, dependencies
3. If a state store is available, check task progress for approved plans
4. Generate the complete report from scratch
5. Commit the file

## Report structure

Write `spec/plans/WHATS-NEXT.md` with this structure:

\```markdown
# What's Next

**Generated:** YYYY-MM-DD
**Mode:** incremental | full

## Completed Since Last Update

- [plan-slug](plan-slug/) — completed YYYY-MM-DD

## In Progress

- [plan-slug](plan-slug/) — N/M steps done, blockers (if any)

## Recommended Next

1. **[plan-slug](plan-slug/)** — Impact: X, Effort: Y. One-sentence reasoning.
2. ...

### Reasoning

2-5 sentences explaining the prioritization: dependency unlocks, ROI, momentum, competing priorities.

## Outstanding Questions

(any ambiguities surfaced during analysis)
\```

## Prioritization logic

Rank candidates by combining these signals (in priority order):

1. **Explicit ROI metadata** — `**Effort:**` and `**Impact:**` fields in plan headers. Higher impact / lower effort = higher priority.
2. **Dependency unlocks** — Plans newly unblocked by recent completions get a priority boost.
3. **Momentum** — Prefer advancing roadmaps that are already in progress over starting new ones.
4. **Feature importance** — Plans targeting features closer to "stable" status get a boost.
5. **AI inference** — When ROI metadata is absent, infer effort from step count/dependency depth and impact from feature importance/downstream dependents.

## Process

1. Check config: run `synchestra feature info development-plan` or read `synchestra-spec-repo.yaml` for `planning.whats_next` setting.
2. If invoked automatically and config is `disabled`, skip silently.
3. Determine mode (incremental or full).
4. Gather data (features, plans, tasks).
5. Generate report following the structure above.
6. Write to `spec/plans/WHATS-NEXT.md`.
7. Commit: `git commit -m "chore: update WHATS-NEXT.md (mode)"`

## Notes

- This skill generates content that costs tokens. Only invoke automatically when the config enables it.
- The report is committed to git, providing a history of how priorities evolved.
- When no plans exist or all are in draft, generate a minimal report noting this.
```

- [ ] **Step 3: Write README.md**

Create `ai-plugin/skills/synchestra-whats-next/README.md`:

```markdown
# synchestra-whats-next

Generates or updates the `WHATS-NEXT.md` prioritization report.

See [SKILL.md](SKILL.md) for full instructions.

## Outstanding Questions

None at this time.
```

- [ ] **Step 4: Run spec-lint to verify the new skill directory is valid**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint`
Expected: No violations for the new skill directory

- [ ] **Step 5: Commit**

```bash
git add ai-plugin/skills/synchestra-whats-next/
git commit -m "feat: add synchestra-whats-next skill for WHATS-NEXT.md generation"
```

---

### Task 6: Restructure existing plans into hierarchy (spec)

Move the existing `chat-infrastructure` and `chat-workflow-engine` plans under `chat-feature` to demonstrate and validate the hierarchical plan convention. Update `chat-feature/README.md` to be a roadmap.

**Files:**
- Move: `spec/plans/chat-infrastructure/` → `spec/plans/chat-feature/chat-infrastructure/`
- Move: `spec/plans/chat-workflow-engine/` → `spec/plans/chat-feature/chat-workflow-engine/`
- Modify: `spec/plans/chat-feature/README.md` (convert to roadmap format)
- Modify: `spec/plans/README.md` (update index table with hierarchy)

- [ ] **Step 1: Move chat-infrastructure under chat-feature**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
git mv spec/plans/chat-infrastructure spec/plans/chat-feature/chat-infrastructure
```

- [ ] **Step 2: Move chat-workflow-engine under chat-feature**

```bash
git mv spec/plans/chat-workflow-engine spec/plans/chat-feature/chat-workflow-engine
```

- [ ] **Step 3: Update internal links in moved plans**

Read both moved plan READMEs. Update any relative links (e.g., `../../features/` becomes `../../../features/`) to account for the new depth.

- [ ] **Step 4: Convert chat-feature/README.md to roadmap format**

Read the current `spec/plans/chat-feature/README.md`. Replace the Steps section with a Child Plans section. Keep the header fields, Context, and Acceptance Criteria. Add ROI metadata if appropriate.

The Child Plans section should be:

```markdown
## Child Plans

| Order | Plan | Status | Effort | Impact |
|-------|------|--------|--------|--------|
| 1 | [chat-infrastructure](chat-infrastructure/) | draft | L | high |
| 2 | [chat-workflow-engine](chat-workflow-engine/) | draft | M | high |
```

- [ ] **Step 5: Update spec/plans/README.md index**

Update the plans table to show hierarchy with indentation. Use `&ensp;` for visual indentation of child plans.

- [ ] **Step 6: Run spec-lint to verify**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint`
Expected: No new violations (the plan-hierarchy checker from Task 3 should pass)

- [ ] **Step 7: Commit**

```bash
git add spec/plans/
git commit -m "refactor: restructure chat plans into roadmap hierarchy"
```

---

### Task 7: End-to-end verification

Run the full test suite and lint check to verify everything works together.

**Files:** (none — verification only)

- [ ] **Step 1: Run all Go tests**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go test ./... 2>&1 | tail -20`
Expected: All tests PASS

- [ ] **Step 2: Run spec-lint on the full spec tree**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint`
Expected: No errors (warnings are acceptable)

- [ ] **Step 3: Build the binary**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go build -o /dev/null .`
Expected: Clean build, exit 0

- [ ] **Step 4: Verify plan hierarchy with the new lint rule**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint --rules plan-hierarchy`
Expected: No violations

- [ ] **Step 5: Verify ROI metadata with the new lint rule**

Run: `cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra && go run . spec lint --rules plan-roi-metadata`
Expected: No violations (existing plans without ROI metadata should not trigger violations)
