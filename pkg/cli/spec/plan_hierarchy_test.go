package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestPlanHierarchyChecker_RoadmapWithSteps(t *testing.T) {
	// A roadmap (has child plan subdirectories with README.md) that also has a ## Steps section
	// should produce a violation.
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/roadmap-a/README.md":            "# Roadmap A\n\n## Steps\n\n- Step 1\n- Step 2\n",
		"plans/roadmap-a/child-plan/README.md": "# Child Plan\n\n## Steps\n\n- Do something\n",
	})

	c := newPlanHierarchyChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}

	// Expect exactly one violation: roadmap should not have Steps
	var stepsViolations []Violation
	for _, viol := range v {
		if strings.Contains(viol.Message, "Steps") {
			stepsViolations = append(stepsViolations, viol)
		}
	}
	if len(stepsViolations) != 1 {
		t.Fatalf("expected 1 Steps violation, got %d: %v", len(stepsViolations), v)
	}
	if stepsViolations[0].Rule != "plan-hierarchy" {
		t.Errorf("expected rule plan-hierarchy, got %s", stepsViolations[0].Rule)
	}
	if stepsViolations[0].Severity != "error" {
		t.Errorf("expected severity error, got %s", stepsViolations[0].Severity)
	}
	if !strings.Contains(stepsViolations[0].File, "roadmap-a/README.md") {
		t.Errorf("expected file to reference roadmap-a/README.md, got %s", stepsViolations[0].File)
	}
}

func TestPlanHierarchyChecker_ThreeLevelNesting(t *testing.T) {
	// Three-level nesting: roadmap -> child -> grandchild should produce a violation.
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/roadmap-a/README.md":                       "# Roadmap A\n\n## Child Plans\n\n- child-plan\n",
		"plans/roadmap-a/child-plan/README.md":            "# Child Plan\n\n## Child Plans\n\n- grandchild\n",
		"plans/roadmap-a/child-plan/grandchild/README.md": "# Grandchild\n\n## Steps\n\n- Do something\n",
	})

	c := newPlanHierarchyChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}

	var nestingViolations []Violation
	for _, viol := range v {
		if strings.Contains(viol.Message, "nesting") || strings.Contains(viol.Message, "depth") {
			nestingViolations = append(nestingViolations, viol)
		}
	}
	if len(nestingViolations) == 0 {
		t.Fatalf("expected nesting violation, got none; all violations: %v", v)
	}
	if nestingViolations[0].Severity != "error" {
		t.Errorf("expected severity error, got %s", nestingViolations[0].Severity)
	}
}

func TestPlanHierarchyChecker_ValidHierarchy(t *testing.T) {
	// Valid hierarchy: roadmap with Child Plans + child plan with Steps + standalone plan with Steps
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/roadmap-a/README.md":            "# Roadmap A\n\n## Child Plans\n\n- child-plan\n",
		"plans/roadmap-a/child-plan/README.md": "# Child Plan\n\n## Steps\n\n- Do something\n",
		"plans/standalone/README.md":           "# Standalone Plan\n\n## Steps\n\n- Step 1\n",
	})

	c := newPlanHierarchyChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations for valid hierarchy, got %d: %v", len(v), v)
	}
}
