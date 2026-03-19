package feature

// Features implemented: cli/feature

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// setupTestFeatures creates a temporary features directory with the given structure.
// features is a map of feature ID → README.md content.
func setupTestFeatures(t *testing.T, features map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for id, content := range features {
		featureDir := filepath.Join(dir, filepath.FromSlash(id))
		if err := os.MkdirAll(featureDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(featureDir, "README.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDiscoverFeatures(t *testing.T) {
	featDir := setupTestFeatures(t, map[string]string{
		"alpha":           "# Alpha",
		"beta":            "# Beta",
		"alpha/child-one": "# Child One",
		"alpha/child-two": "# Child Two",
	})

	// Also create a reserved _args directory that should be skipped
	argsDir := filepath.Join(featDir, "alpha", "_args")
	os.MkdirAll(argsDir, 0o755)
	os.WriteFile(filepath.Join(argsDir, "README.md"), []byte("# Args"), 0o644)

	features, err := discoverFeatures(featDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"alpha", "alpha/child-one", "alpha/child-two", "beta"}
	if len(features) != len(expected) {
		t.Fatalf("got %d features %v, want %d %v", len(features), features, len(expected), expected)
	}
	for i, id := range features {
		if id != expected[i] {
			t.Errorf("feature[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestDiscoverFeatures_Empty(t *testing.T) {
	dir := t.TempDir()
	features, err := discoverFeatures(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 0 {
		t.Errorf("got %d features, want 0", len(features))
	}
}

func TestBuildTree(t *testing.T) {
	ids := []string{"alpha", "alpha/child", "alpha/child/deep", "beta", "gamma"}
	roots := buildTree(ids)

	if len(roots) != 3 {
		t.Fatalf("got %d roots, want 3", len(roots))
	}
	if roots[0].name != "alpha" {
		t.Errorf("root[0] = %q, want alpha", roots[0].name)
	}
	if len(roots[0].children) != 1 || roots[0].children[0].name != "child" {
		t.Errorf("alpha should have 1 child named 'child'")
	}
	if len(roots[0].children[0].children) != 1 || roots[0].children[0].children[0].name != "deep" {
		t.Errorf("alpha/child should have 1 child named 'deep'")
	}
}

func TestPrintTree(t *testing.T) {
	ids := []string{"alpha", "alpha/child", "beta"}
	roots := buildTree(ids)
	var sb strings.Builder
	printTree(&sb, roots, 0)

	got := sb.String()
	want := "alpha\n\tchild\nbeta\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestParseDependencies_BareIDs(t *testing.T) {
	content := `# Feature: Test

## Dependencies

- claim-and-push
- conflict-resolution

## Outstanding Questions

None at this time.
`
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644)

	deps, err := parseDependencies(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"claim-and-push", "conflict-resolution"}
	if len(deps) != len(expected) {
		t.Fatalf("got %d deps %v, want %d %v", len(deps), deps, len(expected), expected)
	}
	for i, d := range deps {
		if d != expected[i] {
			t.Errorf("dep[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseDependencies_MarkdownLinks(t *testing.T) {
	content := `# Feature: GitHub App

## Dependencies

- [API](../api/README.md) — callback endpoint
- [Project Definition](../project-definition/README.md) — config format

## Outstanding Questions
`
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644)

	deps, err := parseDependencies(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"api", "project-definition"}
	sort.Strings(expected)
	if len(deps) != len(expected) {
		t.Fatalf("got %d deps %v, want %d %v", len(deps), deps, len(expected), expected)
	}
	for i, d := range deps {
		if d != expected[i] {
			t.Errorf("dep[%d] = %q, want %q", i, d, expected[i])
		}
	}
}

func TestParseDependencies_NoDependencies(t *testing.T) {
	content := `# Feature: Independent

## Summary

Does its own thing.

## Outstanding Questions

None at this time.
`
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644)

	deps, err := parseDependencies(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 0 {
		t.Errorf("got %d deps, want 0", len(deps))
	}
}

func TestExtractFeatureID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claim-and-push", "claim-and-push"},
		{"cli/task", "cli/task"},
		{"[API](../api/README.md)", "api"},
		{"[API](../api/README.md) — description", "api"},
		{"[Project Definition](../project-definition/README.md) — config format", "project-definition"},
		{"[CLI](../cli/README.md) — entry point", "cli"},
		{"bare-id — some description", "bare-id"},
	}
	for _, tt := range tests {
		got := extractFeatureID(tt.input)
		if got != tt.want {
			t.Errorf("extractFeatureID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFeatureIDFromRelativePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"../api/README.md", "api"},
		{"../cli/README.md", "cli"},
		{"../project-definition/README.md", "project-definition"},
		{"../../some/nested/README.md", "some/nested"},
		{"./local/README.md", "local"},
	}
	for _, tt := range tests {
		got := featureIDFromRelativePath(tt.input)
		if got != tt.want {
			t.Errorf("featureIDFromRelativePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
