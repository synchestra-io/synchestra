package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- readmeExistsChecker ---

func TestReadmeExists_AllPresent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Root")
	mkdir(t, filepath.Join(root, "child"))
	writeFile(t, filepath.Join(root, "child", "README.md"), "# Child")

	c := newReadmeExistsChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations, got %d: %v", len(v), v)
	}
}

func TestReadmeExists_Missing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Root")
	mkdir(t, filepath.Join(root, "child"))
	// No README.md in child

	c := newReadmeExistsChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if v[0].Rule != "readme-exists" {
		t.Errorf("expected rule readme-exists, got %s", v[0].Rule)
	}
	if v[0].Severity != "error" {
		t.Errorf("expected severity error, got %s", v[0].Severity)
	}
}

func TestReadmeExists_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Root")
	mkdir(t, filepath.Join(root, ".hidden"))
	// No README in .hidden — should not be flagged

	c := newReadmeExistsChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations for hidden dir, got %d", len(v))
	}
}

// --- oqSectionChecker ---

func TestOQSection_Present(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n## Outstanding Questions\n\n- Should we add X?\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations, got %d: %v", len(v), v)
	}
}

func TestOQSection_NoneAtThisTime(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n## Outstanding Questions\n\nNone at this time.\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations, got %d: %v", len(v), v)
	}
}

func TestOQSection_Missing(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n## Summary\n\nSome text.\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if v[0].Rule != "oq-section" {
		t.Errorf("expected rule oq-section, got %s", v[0].Rule)
	}
}

func TestOQSection_Empty(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n## Outstanding Questions\n\n## Next Section\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation (oq-not-empty), got %d: %v", len(v), v)
	}
	if v[0].Rule != "oq-not-empty" {
		t.Errorf("expected rule oq-not-empty, got %s", v[0].Rule)
	}
	if v[0].Severity != "warning" {
		t.Errorf("expected severity warning, got %s", v[0].Severity)
	}
}

func TestOQSection_EmptyAtEndOfFile(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n## Outstanding Questions\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if v[0].Rule != "oq-not-empty" {
		t.Errorf("expected rule oq-not-empty, got %s", v[0].Rule)
	}
}

func TestOQSection_PlansDir(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"plans/my-plan/README.md": "# Plan\n\n## Some Section\n\nNo OQ here.\n",
	})

	c := newOQSectionChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation for plans, got %d: %v", len(v), v)
	}
	if !strings.Contains(v[0].File, "plans/") {
		t.Errorf("expected plans path, got %s", v[0].File)
	}
}

// --- indexEntriesChecker ---

func TestIndexEntries_Valid(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md":      "# CLI\n\n| Dir | Desc |\n|---|---|\n| [task](task/README.md) | Task mgmt |\n",
		"features/cli/task/README.md": "# Task\n",
	})

	c := newIndexEntriesChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations, got %d: %v", len(v), v)
	}
}

func TestIndexEntries_NonExistentDir(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md": "# CLI\n\n| Dir | Desc |\n|---|---|\n| [missing](missing/README.md) | Does not exist |\n",
	})

	c := newIndexEntriesChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if !strings.Contains(v[0].Message, "missing") {
		t.Errorf("expected message about 'missing', got %s", v[0].Message)
	}
}

func TestIndexEntries_SkipsArgsDir(t *testing.T) {
	root := setupSpecTree(t, map[string]string{
		"features/cli/README.md":       "# CLI\n\n| Dir | Desc |\n|---|---|\n| [_args](_args/README.md) | Args |\n| [task](task/README.md) | Task |\n",
		"features/cli/_args/README.md": "# Args\n",
		"features/cli/task/README.md":  "# Task\n",
	})

	c := newIndexEntriesChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations (_args should be skipped), got %d: %v", len(v), v)
	}
}

func TestIndexEntries_SkipsCodeBlocks(t *testing.T) {
	content := "# Feature\n\n```\n| [child](child/README.md) | example |\n```\n"
	root := setupSpecTree(t, map[string]string{
		"features/f/README.md": content,
	})

	c := newIndexEntriesChecker()
	v, err := c.check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Errorf("expected 0 violations (code block should be skipped), got %d: %v", len(v), v)
	}
}

// --- linter orchestration ---

func TestLinter_RulesFilter(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "empty-dir"))
	// No README.md in empty-dir → readme-exists violation

	opts := LintOptions{
		SpecRoot: root,
		Rules:    []string{"oq-section"},
		Severity: "error",
		Format:   "text",
	}
	l := newLinter(opts)
	if l.isRuleEnabled("readme-exists") {
		t.Error("readme-exists should be disabled when --rules=oq-section")
	}
	if !l.isRuleEnabled("oq-section") {
		t.Error("oq-section should be enabled")
	}
}

func TestLinter_IgnoreFilter(t *testing.T) {
	opts := LintOptions{
		SpecRoot: t.TempDir(),
		Ignore:   []string{"code-annotations"},
		Severity: "error",
		Format:   "text",
	}
	l := newLinter(opts)
	if l.isRuleEnabled("code-annotations") {
		t.Error("code-annotations should be disabled when --ignore=code-annotations")
	}
	if !l.isRuleEnabled("readme-exists") {
		t.Error("readme-exists should be enabled")
	}
}

func TestFilterBySeverity(t *testing.T) {
	violations := []Violation{
		{Severity: "error", Rule: "readme-exists"},
		{Severity: "warning", Rule: "heading-levels"},
		{Severity: "info", Rule: "diag"},
	}

	errOnly := filterBySeverity(violations, "error")
	if len(errOnly) != 1 {
		t.Errorf("error filter: expected 1, got %d", len(errOnly))
	}

	warnAndUp := filterBySeverity(violations, "warning")
	if len(warnAndUp) != 2 {
		t.Errorf("warning filter: expected 2, got %d", len(warnAndUp))
	}

	all := filterBySeverity(violations, "info")
	if len(all) != 3 {
		t.Errorf("info filter: expected 3, got %d", len(all))
	}
}

func TestValidateRuleNames(t *testing.T) {
	if err := validateRuleNames([]string{"readme-exists", "oq-section"}); err != nil {
		t.Errorf("valid rules should not error: %v", err)
	}
	if err := validateRuleNames([]string{"bogus"}); err == nil {
		t.Error("expected error for unknown rule 'bogus'")
	}
	if err := validateRuleNames(nil); err != nil {
		t.Errorf("nil rules should not error: %v", err)
	}
}

// --- output format ---

func TestOutputJSON_Empty(t *testing.T) {
	// Just ensure it doesn't panic on empty slice
	err := outputJSON([]Violation{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestViolation_JSONRoundTrip(t *testing.T) {
	v := Violation{
		File:     "features/cli/README.md",
		Line:     42,
		Severity: "error",
		Rule:     "oq-section",
		Message:  "Outstanding Questions section not found",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var got Violation
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != v {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, v)
	}
}

// --- end-to-end against real spec tree ---

func TestLintRealSpecTree(t *testing.T) {
	// Only run when invoked from the repo root (CI or local dev).
	specRoot := "../../.."
	if _, err := os.Stat(filepath.Join(specRoot, "spec", "features")); err != nil {
		t.Skip("not running from repo root; skipping integration test")
	}

	opts := LintOptions{
		SpecRoot: filepath.Join(specRoot, "spec"),
		Severity: "error",
		Format:   "text",
		// Ignore rules that are not yet enforced repo-wide.
		Ignore: []string{"code-annotations"},
	}
	l := newLinter(opts)
	violations, err := l.lint()
	if err != nil {
		t.Fatal(err)
	}
	filtered := filterBySeverity(violations, "error")
	for _, v := range filtered {
		t.Errorf("%s:%d [%s] %s: %s", v.File, v.Line, v.Severity, v.Rule, v.Message)
	}
}

// --- helpers ---

func setupSpecTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)
		mkdir(t, filepath.Dir(fullPath))
		writeFile(t, fullPath, content)
	}
	return root
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
