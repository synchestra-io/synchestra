package feature

// Features implemented: cli/feature/list, cli/feature/tree, cli/feature/deps, cli/feature/refs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

// executeCommand runs a cobra command with the given args and captures combined
// stdout/stderr output.
func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// testFeatures defines the feature README content used across all integration tests.
var testFeatures = map[string]string{
	"alpha": `# Alpha

**Status:** Approved

## Dependencies

- beta

## Outstanding Questions

- Question 1
- Question 2
`,
	"beta": `# Beta

**Status:** Draft

## Outstanding Questions

None at this time.
`,
	"gamma": `# Gamma

**Status:** Implemented

## Dependencies

- alpha

## Outstanding Questions

- Question 1
`,
	"alpha/child": `# Alpha Child

**Status:** Draft

## Outstanding Questions

None at this time.
`,
}

// setupSpecRepoWithFeatures creates a temp directory with spec/features/ structure,
// populates it with the provided feature READMEs, changes CWD to the temp dir,
// and registers cleanup to restore the original CWD.
// Returns the temp dir root.
func setupSpecRepoWithFeatures(t *testing.T, features map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	featDir := filepath.Join(tmpDir, "spec", "features")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for id, content := range features {
		featureDir := filepath.Join(featDir, filepath.FromSlash(id))
		if err := os.MkdirAll(featureDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(featureDir, "README.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return tmpDir
}

// requireExitCode asserts that err is an *exitcode.Error with the expected code.
func requireExitCode(t *testing.T, err error, wantCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected exit code %d, got nil error", wantCode)
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if ee.ExitCode() != wantCode {
		t.Errorf("expected exit code %d, got %d (msg: %s)", wantCode, ee.ExitCode(), ee.Error())
	}
}

// ---------------------------------------------------------------------------
// List command tests
// ---------------------------------------------------------------------------

func TestList_FieldsStatus(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(listCommand(), "--fields=status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "path: alpha") {
		t.Errorf("expected output to contain 'path: alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Approved") {
		t.Errorf("expected output to contain 'status: Approved', got:\n%s", out)
	}
	if !strings.Contains(out, "path: beta") {
		t.Errorf("expected output to contain 'path: beta', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Draft") {
		t.Errorf("expected output to contain 'status: Draft', got:\n%s", out)
	}
}

func TestList_FieldsStatusAndOQ(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(listCommand(), "--fields=status,oq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "status: Approved") {
		t.Errorf("expected YAML with 'status: Approved', got:\n%s", out)
	}
	if !strings.Contains(out, "oq:") {
		t.Errorf("expected YAML with 'oq:' field, got:\n%s", out)
	}
}

func TestList_FieldsInvalid(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	_, err := executeCommand(listCommand(), "--fields=invalid")
	requireExitCode(t, err, 2)
}

func TestList_FormatJSON(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(listCommand(), "--format=json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "[") {
		t.Errorf("expected JSON array output starting with '[', got:\n%s", out)
	}
	if !strings.Contains(out, `"path"`) {
		t.Errorf("expected JSON to contain '\"path\"', got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Tree command tests — --direction flag
// ---------------------------------------------------------------------------

func TestTree_FocusedDefault(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(treeCommand(), "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "* alpha") {
		t.Errorf("expected focused tree to contain '* alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "child") {
		t.Errorf("expected focused tree to include child, got:\n%s", out)
	}
}

func TestTree_DirectionUp(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(treeCommand(), "alpha", "--direction=up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "* alpha") {
		t.Errorf("expected '* alpha' in up tree, got:\n%s", out)
	}
	// --direction=up should NOT include children
	if strings.Contains(out, "child") {
		t.Errorf("expected no child in --direction=up output, got:\n%s", out)
	}
}

func TestTree_DirectionDown(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(treeCommand(), "alpha", "--direction=down")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "* alpha") {
		t.Errorf("expected '* alpha' in down tree, got:\n%s", out)
	}
	if !strings.Contains(out, "child") {
		t.Errorf("expected child in --direction=down output, got:\n%s", out)
	}
}

func TestTree_DirectionWithoutFeatureID(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	_, err := executeCommand(treeCommand(), "--direction=up")
	requireExitCode(t, err, 2)
}

func TestTree_NonexistentFeature(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	_, err := executeCommand(treeCommand(), "nonexistent")
	requireExitCode(t, err, 3)
}

// ---------------------------------------------------------------------------
// Tree command tests — --fields flag
// ---------------------------------------------------------------------------

func TestTree_FieldsStatus(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(treeCommand(), "--fields=status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "path:") {
		t.Errorf("expected YAML tree with 'path:', got:\n%s", out)
	}
	if !strings.Contains(out, "status:") {
		t.Errorf("expected YAML tree with 'status:', got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Deps command tests — --fields and --transitive flags
// ---------------------------------------------------------------------------

func TestDeps_PlainText(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(depsCommand(), "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected deps to contain 'beta', got:\n%s", out)
	}
}

func TestDeps_FieldsStatus(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(depsCommand(), "alpha", "--fields=status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "path: beta") {
		t.Errorf("expected YAML with 'path: beta', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Draft") {
		t.Errorf("expected YAML with 'status: Draft', got:\n%s", out)
	}
}

func TestDeps_Transitive(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	// gamma depends on alpha, alpha depends on beta
	out, err := executeCommand(depsCommand(), "gamma", "--transitive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected transitive deps to contain 'alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("expected transitive deps to contain 'beta', got:\n%s", out)
	}
	// beta should be indented under alpha (tab-indented)
	lines := strings.Split(out, "\n")
	betaIndented := false
	for _, line := range lines {
		if strings.Contains(line, "beta") && strings.HasPrefix(line, "\t") {
			betaIndented = true
			break
		}
	}
	if !betaIndented {
		t.Errorf("expected 'beta' to be indented under 'alpha', got:\n%s", out)
	}
}

func TestDeps_TransitiveWithFields(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(depsCommand(), "gamma", "--transitive", "--fields=status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "path: alpha") {
		t.Errorf("expected YAML with 'path: alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Approved") {
		t.Errorf("expected alpha to have 'status: Approved', got:\n%s", out)
	}
	if !strings.Contains(out, "path: beta") {
		t.Errorf("expected YAML with 'path: beta', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Draft") {
		t.Errorf("expected beta to have 'status: Draft', got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Refs command tests — --fields and --transitive flags
// ---------------------------------------------------------------------------

func TestRefs_PlainText(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(refsCommand(), "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected refs to contain 'alpha', got:\n%s", out)
	}
}

func TestRefs_FieldsStatus(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	out, err := executeCommand(refsCommand(), "beta", "--fields=status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "path: alpha") {
		t.Errorf("expected YAML with 'path: alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "status: Approved") {
		t.Errorf("expected YAML with 'status: Approved', got:\n%s", out)
	}
}

func TestRefs_Transitive(t *testing.T) {
	setupSpecRepoWithFeatures(t, testFeatures)

	// beta is depended on by alpha; alpha is depended on by gamma
	out, err := executeCommand(refsCommand(), "beta", "--transitive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected transitive refs to contain 'alpha', got:\n%s", out)
	}
	if !strings.Contains(out, "gamma") {
		t.Errorf("expected transitive refs to contain 'gamma', got:\n%s", out)
	}
	// gamma should be indented under alpha (tab-indented)
	lines := strings.Split(out, "\n")
	gammaIndented := false
	for _, line := range lines {
		if strings.Contains(line, "gamma") && strings.HasPrefix(line, "\t") {
			gammaIndented = true
			break
		}
	}
	if !gammaIndented {
		t.Errorf("expected 'gamma' to be indented under 'alpha', got:\n%s", out)
	}
}
