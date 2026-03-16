package test

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScenarioFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHasTag(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		tag  string
		want bool
	}{
		{"found", []string{"demo", "manual"}, "manual", true},
		{"not found", []string{"demo", "integration"}, "manual", false},
		{"empty tags", nil, "manual", false},
		{"exact match only", []string{"manual-test"}, "manual", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasTag(tt.tags, tt.tag); got != tt.want {
				t.Errorf("hasTag(%v, %q) = %v, want %v", tt.tags, tt.tag, got, tt.want)
			}
		})
	}
}

func TestMatchesTags(t *testing.T) {
	tests := []struct {
		name      string
		scenario  []string
		filter    []string
		wantMatch bool
	}{
		{"single match", []string{"demo", "integration"}, []string{"demo"}, true},
		{"no match", []string{"demo"}, []string{"integration"}, false},
		{"multiple filter one match", []string{"demo"}, []string{"integration", "demo"}, true},
		{"empty scenario tags", nil, []string{"demo"}, false},
		{"empty filter", []string{"demo"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesTags(tt.scenario, tt.filter); got != tt.wantMatch {
				t.Errorf("matchesTags(%v, %v) = %v, want %v", tt.scenario, tt.filter, got, tt.wantMatch)
			}
		})
	}
}

func TestCollectScenarioFiles_singleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.md")
	if err := os.WriteFile(f, []byte("# Scenario: x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := collectScenarioFiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != f {
		t.Errorf("got %v, want [%s]", files, f)
	}
}

func TestCollectScenarioFiles_directory(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("a.md", "# Scenario: a")
	writeFile("b.md", "# Scenario: b")
	writeFile("README.md", "# Readme")
	writeFile("notes.txt", "not a scenario")

	files, err := collectScenarioFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Base(f) == "README.md" {
			t.Error("README.md should be excluded")
		}
		if filepath.Base(f) == "notes.txt" {
			t.Error("non-.md files should be excluded")
		}
	}
}

func TestRunCommand_skipsManualInDirectory(t *testing.T) {
	dir := t.TempDir()

	// Non-manual scenario that passes.
	writeScenarioFile(t, filepath.Join(dir, "auto.md"), `# Scenario: Auto

**Description:** Auto test.
**Tags:** integration

## step-a

`+"```bash\nexit 0\n```\n")

	// Manual scenario that would fail.
	writeScenarioFile(t, filepath.Join(dir, "manual.md"), `# Scenario: Manual

**Description:** Manual demo.
**Tags:** demo, manual

## fail-step

`+"```bash\nexit 1\n```\n")

	cmd := runCommand()
	cmd.SetArgs([]string{dir, "--format", "json"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected pass (manual skipped), got error: %v", err)
	}
}

func TestRunCommand_runsManualWhenSpecificFile(t *testing.T) {
	dir := t.TempDir()

	f := filepath.Join(dir, "manual.md")
	writeScenarioFile(t, f, `# Scenario: Manual

**Description:** Manual demo.
**Tags:** manual

## pass-step

`+"```bash\nexit 0\n```\n")

	cmd := runCommand()
	cmd.SetArgs([]string{f, "--format", "json"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("specific file should run even if manual, got error: %v", err)
	}
}

func TestRunCommand_runsManualWithFlag(t *testing.T) {
	dir := t.TempDir()

	writeScenarioFile(t, filepath.Join(dir, "manual.md"), `# Scenario: Manual

**Description:** Manual demo.
**Tags:** manual

## pass-step

`+"```bash\nexit 0\n```\n")

	cmd := runCommand()
	cmd.SetArgs([]string{dir, "--format", "json", "--run-manual-tests"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("--run-manual-tests should include manual scenarios, got error: %v", err)
	}
}
