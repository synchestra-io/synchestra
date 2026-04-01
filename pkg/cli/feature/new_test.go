package feature

// Features implemented: cli/feature/new

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// setupSpecRepo creates a temp dir with a spec/features directory tree, writes a
// minimal feature index README, changes CWD to the temp dir and restores it on
// cleanup.  It returns the temp dir root and the features dir path.
func setupSpecRepo(t *testing.T) (tmpDir, featDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	featDir = filepath.Join(tmpDir, "spec", "features")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write index README with an existing table so UpdateFeatureIndex can append.
	indexContent := "# Features\n\n| Feature | Description |\n|---|---|\n"
	if err := os.WriteFile(filepath.Join(featDir, "README.md"), []byte(indexContent), 0o644); err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return tmpDir, featDir
}

// executeNew runs newCommand() with the provided args and returns the command
// output and any error returned by Execute().
func executeNew(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// TestRunNew_BasicTopLevel creates a top-level feature by title and verifies the
// directory, README and YAML output are all correct.
func TestRunNew_BasicTopLevel(t *testing.T) {
	_, featDir := setupSpecRepo(t)

	out, err := executeNew(t, "--title", "My New Feature")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	featureDir := filepath.Join(featDir, "my-new-feature")
	if _, statErr := os.Stat(featureDir); os.IsNotExist(statErr) {
		t.Fatalf("expected directory %s to be created", featureDir)
	}
	readmePath := filepath.Join(featureDir, "README.md")
	if _, statErr := os.Stat(readmePath); os.IsNotExist(statErr) {
		t.Fatalf("expected README.md to be created at %s", readmePath)
	}
	if !strings.Contains(out, "path: my-new-feature") {
		t.Errorf("expected stdout to contain 'path: my-new-feature', got:\n%s", out)
	}
}

// TestRunNew_WithParent creates a child feature under an existing parent and
// verifies both the child directory and the parent's updated Contents section.
func TestRunNew_WithParent(t *testing.T) {
	_, featDir := setupSpecRepo(t)

	// Create parent feature.
	parentDir := filepath.Join(featDir, "parent-feat")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentReadme := filepath.Join(parentDir, "README.md")
	if err := os.WriteFile(parentReadme, []byte("# Parent Feature\n\n## Summary\n\nParent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeNew(t, "--title", "Child Feature", "--parent", "parent-feat")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	childReadme := filepath.Join(featDir, "parent-feat", "child-feature", "README.md")
	if _, statErr := os.Stat(childReadme); os.IsNotExist(statErr) {
		t.Fatalf("expected child README at %s", childReadme)
	}

	data, readErr := os.ReadFile(parentReadme)
	if readErr != nil {
		t.Fatal(readErr)
	}
	contents := string(data)
	if !strings.Contains(contents, "## Contents") {
		t.Errorf("expected parent README to contain '## Contents', got:\n%s", contents)
	}
	if !strings.Contains(contents, "child-feature") {
		t.Errorf("expected parent README to reference 'child-feature', got:\n%s", contents)
	}
}

// TestRunNew_WithDependsOn creates a feature with two declared dependencies and
// verifies the generated README lists both deps.
func TestRunNew_WithDependsOn(t *testing.T) {
	_, featDir := setupSpecRepo(t)

	// Create dependency features.
	for _, dep := range []string{"dep-a", "dep-b"} {
		dir := filepath.Join(featDir, dep)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+dep+"\n\n## Summary\n\nDep.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := executeNew(t, "--title", "New Thing", "--depends-on", "dep-a,dep-b")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	readmePath := filepath.Join(featDir, "new-thing", "README.md")
	data, readErr := os.ReadFile(readmePath)
	if readErr != nil {
		t.Fatalf("could not read generated README: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, "## Dependencies") {
		t.Errorf("expected README to contain '## Dependencies', got:\n%s", content)
	}
	if !strings.Contains(content, "dep-a") {
		t.Errorf("expected README to reference 'dep-a', got:\n%s", content)
	}
	if !strings.Contains(content, "dep-b") {
		t.Errorf("expected README to reference 'dep-b', got:\n%s", content)
	}
}

// TestRunNew_InvalidDependency verifies that a non-existent dependency returns
// an exitcode.Error with code 2.
func TestRunNew_InvalidDependency(t *testing.T) {
	setupSpecRepo(t)

	_, err := executeNew(t, "--title", "Bad", "--depends-on", "nonexistent-feature")
	if err == nil {
		t.Fatal("expected an error for invalid dependency, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}
}

// TestRunNew_MissingTitle verifies that omitting --title returns an exitcode.Error
// with code 2.
func TestRunNew_MissingTitle(t *testing.T) {
	setupSpecRepo(t)

	_, err := executeNew(t)
	if err == nil {
		t.Fatal("expected an error when title is missing, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}
}

// TestRunNew_InvalidStatus verifies that an unrecognised status returns an
// exitcode.Error with code 2.
func TestRunNew_InvalidStatus(t *testing.T) {
	setupSpecRepo(t)

	_, err := executeNew(t, "--title", "X", "--status", "invalid-status")
	if err == nil {
		t.Fatal("expected an error for invalid status, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}
}

// TestRunNew_ParentNotFound verifies that a missing --parent returns an
// exitcode.Error with code 3.
func TestRunNew_ParentNotFound(t *testing.T) {
	setupSpecRepo(t)

	_, err := executeNew(t, "--title", "X", "--parent", "nonexistent")
	if err == nil {
		t.Fatal("expected an error for missing parent, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 3 {
		t.Errorf("expected exit code 3, got %d", exitErr.ExitCode())
	}
}

// TestRunNew_AlreadyExists verifies that trying to create a feature whose
// directory already exists returns an exitcode.Error with code 4.
func TestRunNew_AlreadyExists(t *testing.T) {
	_, featDir := setupSpecRepo(t)

	// Pre-create the "existing" feature.
	existingDir := filepath.Join(featDir, "existing")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "README.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeNew(t, "--title", "Existing", "--slug", "existing")
	if err == nil {
		t.Fatal("expected an error for already-existing feature, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 4 {
		t.Errorf("expected exit code 4, got %d", exitErr.ExitCode())
	}
}

// TestRunNew_ParentSlugConflict verifies that providing both --parent and a
// slash-containing --slug returns an exitcode.Error with code 2.
func TestRunNew_ParentSlugConflict(t *testing.T) {
	setupSpecRepo(t)

	_, err := executeNew(t, "--title", "X", "--parent", "some-parent", "--slug", "nested/path")
	if err == nil {
		t.Fatal("expected an error for parent+slug conflict, got nil")
	}
	var exitErr *exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}
}
