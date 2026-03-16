package testscenario

// Features implemented: testing-framework/test-runner

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
