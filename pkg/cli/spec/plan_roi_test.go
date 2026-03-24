package spec

import (
	"strings"
	"testing"
)

func TestPlanROIChecker_InvalidEffort(t *testing.T) {
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/my-plan/README.md": "# My Plan\n\n**Effort:** huge\n**Impact:** high\n\n## Steps\n\n- Step 1\n",
	})

	c := newPlanROIChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if !strings.Contains(v[0].Message, "Effort") {
		t.Errorf("expected violation to mention Effort, got: %s", v[0].Message)
	}
	if v[0].Severity != "warning" {
		t.Errorf("expected severity warning, got %s", v[0].Severity)
	}
	if v[0].Rule != "plan-roi-metadata" {
		t.Errorf("expected rule plan-roi-metadata, got %s", v[0].Rule)
	}
}

func TestPlanROIChecker_ValidMetadata(t *testing.T) {
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/my-plan/README.md": "# My Plan\n\n**Effort:** M\n**Impact:** high\n\n## Steps\n\n- Step 1\n",
	})

	c := newPlanROIChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(v) != 0 {
		t.Errorf("expected 0 violations for valid metadata, got %d: %v", len(v), v)
	}
}

func TestPlanROIChecker_NoMetadata(t *testing.T) {
	root := setupPlanHierarchyFixture(t, map[string]string{
		"plans/my-plan/README.md": "# My Plan\n\n## Steps\n\n- Step 1\n",
	})

	c := newPlanROIChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(v) != 0 {
		t.Errorf("expected 0 violations when metadata absent, got %d: %v", len(v), v)
	}
}
