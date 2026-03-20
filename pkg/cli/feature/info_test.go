package feature

// Features implemented: cli/feature/info

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseFeatureStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "bold status",
			content: "# Feature\n\n**Status:** Conceptual\n\n## Summary\n",
			want:    "Conceptual",
		},
		{
			name:    "plain status",
			content: "# Feature\n\nStatus: Implemented\n",
			want:    "Implemented",
		},
		{
			name:    "backtick status",
			content: "# Feature\n\n**Status:** `Draft`\n",
			want:    "Draft",
		},
		{
			name:    "no status",
			content: "# Feature\n\n## Summary\n\nHello\n",
			want:    "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "README.md")
			os.WriteFile(path, []byte(tt.content), 0o644)

			got, err := parseFeatureStatus(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("parseFeatureStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseContentsTable(t *testing.T) {
	content := `# Feature

## Contents

| Child | Description |
|---|---|
| [Alpha](alpha/README.md) | First child |
| [Beta](beta/README.md) | Second child |

## Outstanding Questions

None at this time.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	os.WriteFile(path, []byte(content), 0o644)

	entries, err := parseContentsTable(path)
	if err != nil {
		t.Fatal(err)
	}

	if !entries["alpha"] {
		t.Error("expected alpha in contents entries")
	}
	if !entries["beta"] {
		t.Error("expected beta in contents entries")
	}
	if entries["gamma"] {
		t.Error("did not expect gamma in contents entries")
	}
}

func TestDiscoverChildFeatures(t *testing.T) {
	// Create parent feature with Contents table listing only "alpha"
	parentContent := `# Parent

## Contents

| Child | Description |
|---|---|
| [Alpha](alpha/README.md) | Listed child |

## Outstanding Questions

None at this time.
`
	featDir := setupTestFeatures(t, map[string]string{
		"parent":       parentContent,
		"parent/alpha": "# Alpha",
		"parent/beta":  "# Beta",
	})

	parentReadme := filepath.Join(featDir, "parent", "README.md")
	children, err := discoverChildFeatures(featDir, "parent", parentReadme)
	if err != nil {
		t.Fatal(err)
	}

	if len(children) != 2 {
		t.Fatalf("got %d children, want 2", len(children))
	}

	// alpha should be in_readme=true, beta should be in_readme=false
	for _, c := range children {
		switch c.Path {
		case "parent/alpha":
			if !c.InReadme {
				t.Error("alpha should have in_readme=true")
			}
		case "parent/beta":
			if c.InReadme {
				t.Error("beta should have in_readme=false")
			}
		default:
			t.Errorf("unexpected child: %s", c.Path)
		}
	}
}

func TestFindLinkedPlans(t *testing.T) {
	// Create a repo structure with plans that reference features
	repoDir := t.TempDir()

	// Create spec/plans/plan-a/README.md referencing our feature
	planADir := filepath.Join(repoDir, "spec", "plans", "plan-a")
	if err := os.MkdirAll(planADir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planADir, "README.md"), []byte(`# Plan A

**Status:** draft
**Features:**
  - [My Feature](../../features/cli/task/claim/README.md)
**Source:** something

## Tasks
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create spec/plans/plan-b/README.md NOT referencing our feature
	planBDir := filepath.Join(repoDir, "spec", "plans", "plan-b")
	if err := os.MkdirAll(planBDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planBDir, "README.md"), []byte(`# Plan B

**Status:** draft
**Features:**
  - [Other](../../features/other/README.md)

## Tasks
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create plans index README (should be skipped)
	if err := os.WriteFile(filepath.Join(repoDir, "spec", "plans", "README.md"), []byte("# Plans\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plans, err := findLinkedPlans(repoDir, "cli/task/claim")
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1: %v", len(plans), plans)
	}
	if plans[0] != "plan-a" {
		t.Errorf("got plan %q, want plan-a", plans[0])
	}
}

func TestParseSections(t *testing.T) {
	content := `# Feature Title

## Summary

Some summary text.

## Behavior

Overview text.

### Claiming Protocol

Steps here.

### Conflict Handling

More steps.

## Dependencies

- dep-a
- dep-b

## Outstanding Questions

- Question 1
- Question 2
- Question 3
`
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	os.WriteFile(path, []byte(content), 0o644)

	sections, err := parseSections(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(sections) != 4 {
		t.Fatalf("got %d sections, want 4", len(sections))
	}

	// Summary
	if sections[0].Title != "Summary" {
		t.Errorf("section[0] = %q, want Summary", sections[0].Title)
	}

	// Behavior with children
	if sections[1].Title != "Behavior" {
		t.Errorf("section[1] = %q, want Behavior", sections[1].Title)
	}
	if len(sections[1].Children) != 2 {
		t.Fatalf("Behavior has %d children, want 2", len(sections[1].Children))
	}
	if sections[1].Children[0].Title != "Claiming Protocol" {
		t.Errorf("child[0] = %q, want Claiming Protocol", sections[1].Children[0].Title)
	}
	if sections[1].Children[1].Title != "Conflict Handling" {
		t.Errorf("child[1] = %q, want Conflict Handling", sections[1].Children[1].Title)
	}

	// Dependencies with items
	if sections[2].Title != "Dependencies" {
		t.Errorf("section[2] = %q, want Dependencies", sections[2].Title)
	}
	if sections[2].Items != 2 {
		t.Errorf("Dependencies items = %d, want 2", sections[2].Items)
	}

	// Outstanding Questions with items
	if sections[3].Title != "Outstanding Questions" {
		t.Errorf("section[3] = %q, want Outstanding Questions", sections[3].Title)
	}
	if sections[3].Items != 3 {
		t.Errorf("OQ items = %d, want 3", sections[3].Items)
	}
}

func TestParseSections_LineRanges(t *testing.T) {
	content := `# Title

## First

Line 4
Line 5

## Second

Line 9
`
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	os.WriteFile(path, []byte(content), 0o644)

	sections, err := parseSections(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	if sections[0].Lines != "3-7" {
		t.Errorf("First lines = %q, want 3-7", sections[0].Lines)
	}
	if sections[1].Lines != "8-10" {
		t.Errorf("Second lines = %q, want 8-10", sections[1].Lines)
	}
}

func TestRunInfo_YAMLOutput(t *testing.T) {
	featDir := setupTestFeatures(t, map[string]string{
		"myfeature": `# My Feature

**Status:** Conceptual

## Summary

A test feature.

## Dependencies

- other-feature

## Outstanding Questions

- Is this a test?
`,
		"other-feature": "# Other",
	})

	// Need to set up the spec repo structure for resolveFeaturesDir
	// Instead, test via the internal functions
	readmePath := filepath.Join(featDir, "myfeature", "README.md")

	status, err := parseFeatureStatus(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if status != "Conceptual" {
		t.Errorf("status = %q, want Conceptual", status)
	}

	deps, err := parseDependencies(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 || deps[0] != "other-feature" {
		t.Errorf("deps = %v, want [other-feature]", deps)
	}

	sections, err := parseSections(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 3 {
		t.Errorf("got %d sections, want 3", len(sections))
	}

	// Verify YAML marshaling works
	info := featureInfo{
		Path:     "myfeature",
		Status:   status,
		Deps:     deps,
		Refs:     []string{},
		Sections: sections,
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(info); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "path: myfeature") {
		t.Errorf("YAML missing path field:\n%s", output)
	}
	if !strings.Contains(output, "status: Conceptual") {
		t.Errorf("YAML missing status field:\n%s", output)
	}
}

func TestRunInfo_JSONOutput(t *testing.T) {
	info := featureInfo{
		Path:     "test/feature",
		Status:   "Draft",
		Deps:     []string{"dep-a"},
		Refs:     []string{"ref-b"},
		Children: []childInfo{{Path: "test/feature/child", InReadme: true}},
		Plans:    []string{"my-plan"},
		Sections: []sectionInfo{{Title: "Summary", Lines: "3-5"}},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(info); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, `"path": "test/feature"`) {
		t.Errorf("JSON missing path:\n%s", output)
	}
	if !strings.Contains(output, `"in_readme": true`) {
		t.Errorf("JSON missing in_readme:\n%s", output)
	}
}

func TestRunInfo_TextOutput(t *testing.T) {
	info := featureInfo{
		Path:   "test/feature",
		Status: "Conceptual",
		Deps:   []string{"dep-a", "dep-b"},
		Refs:   []string{},
		Children: []childInfo{
			{Path: "test/feature/alpha", InReadme: true},
			{Path: "test/feature/beta", InReadme: false},
		},
		Plans: []string{"my-plan"},
		Sections: []sectionInfo{
			{Title: "Summary", Lines: "3-5"},
			{Title: "Behavior", Lines: "7-20", Children: []sectionInfo{
				{Title: "Sub", Lines: "9-15"},
			}},
		},
	}

	var buf bytes.Buffer
	if err := writeTextInfo(&buf, info); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Feature: test/feature") {
		t.Errorf("text missing feature header:\n%s", output)
	}
	if !strings.Contains(output, "✓ test/feature/alpha") {
		t.Errorf("text missing alpha child:\n%s", output)
	}
	if !strings.Contains(output, "✗ test/feature/beta") {
		t.Errorf("text missing beta child:\n%s", output)
	}
	if !strings.Contains(output, "Plans:   my-plan") {
		t.Errorf("text missing plans:\n%s", output)
	}
}

func TestFindFeatureRefs(t *testing.T) {
	featDir := setupTestFeatures(t, map[string]string{
		"target": "# Target\n\n## Outstanding Questions\n\nNone.\n",
		"referrer": `# Referrer

## Dependencies

- target

## Outstanding Questions

None.
`,
		"unrelated": "# Unrelated\n\n## Outstanding Questions\n\nNone.\n",
	})

	refs, err := findFeatureRefs(featDir, "target")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 1 || refs[0] != "referrer" {
		t.Errorf("refs = %v, want [referrer]", refs)
	}
}

func TestFindLinkedPlans_NoPlanDir(t *testing.T) {
	repoDir := t.TempDir()
	plans, err := findLinkedPlans(repoDir, "some/feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Errorf("expected no plans, got %v", plans)
	}
}
