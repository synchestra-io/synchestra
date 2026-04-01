package feature

// Features implemented: cli/feature/info

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	specfeature "github.com/synchestra-io/specscore/pkg/feature"
	"gopkg.in/yaml.v3"
)

func TestRunInfo_YAMLOutput(t *testing.T) {
	// Verify YAML marshaling works with the specscore Info type
	info := specfeature.Info{
		Path:   "myfeature",
		Status: "Conceptual",
		Deps:   []string{"other-feature"},
		Refs:   []string{},
		Sections: []specfeature.SectionInfo{
			{Title: "Summary", Lines: "5-7"},
			{Title: "Dependencies", Lines: "9-12", Items: 1},
			{Title: "Outstanding Questions", Lines: "14-16", Items: 1},
		},
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
	info := specfeature.Info{
		Path:     "test/feature",
		Status:   "Draft",
		Deps:     []string{"dep-a"},
		Refs:     []string{"ref-b"},
		Children: []specfeature.ChildInfo{{Path: "test/feature/child", InReadme: true}},
		Plans:    []string{"my-plan"},
		Sections: []specfeature.SectionInfo{{Title: "Summary", Lines: "3-5"}},
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
	info := specfeature.Info{
		Path:   "test/feature",
		Status: "Conceptual",
		Deps:   []string{"dep-a", "dep-b"},
		Refs:   []string{},
		Children: []specfeature.ChildInfo{
			{Path: "test/feature/alpha", InReadme: true},
			{Path: "test/feature/beta", InReadme: false},
		},
		Plans: []string{"my-plan"},
		Sections: []specfeature.SectionInfo{
			{Title: "Summary", Lines: "3-5"},
			{Title: "Behavior", Lines: "7-20", Children: []specfeature.SectionInfo{
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
