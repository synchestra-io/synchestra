package feature

// Features implemented: cli/feature

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// parseFieldNames
// ---------------------------------------------------------------------------

func TestParseFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "single valid field status",
			input: "status",
			want:  []string{"status"},
		},
		{
			name:  "multiple valid fields",
			input: "status,oq,deps",
			want:  []string{"status", "oq", "deps"},
		},
		{
			name:      "unknown field returns error",
			input:     "invalid",
			wantErr:   true,
			errSubstr: "unknown field",
		},
		{
			name:  "duplicate fields are deduplicated",
			input: "status,status",
			want:  []string{"status"},
		},
		{
			name:  "whitespace around commas is trimmed",
			input: " status , oq ",
			want:  []string{"status", "oq"},
		},
		{
			name:  "all seven valid fields",
			input: "status,oq,deps,refs,children,plans,proposals",
			want:  []string{"status", "oq", "deps", "refs", "children", "plans", "proposals"},
		},
		{
			name:  "multiple duplicates preserve first occurrence order",
			input: "oq,deps,oq,deps,status",
			want:  []string{"oq", "deps", "status"},
		},
		{
			name:      "mix of valid and invalid returns error",
			input:     "status,bogus",
			wantErr:   true,
			errSubstr: "unknown field",
		},
		{
			name:  "trailing comma produces no extra entries",
			input: "status,",
			want:  []string{"status"},
		},
		{
			name:  "leading comma produces no extra entries",
			input: ",deps",
			want:  []string{"deps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseFieldNames(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.want == nil && got != nil {
				t.Fatalf("got %v, want nil", got)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("field[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// countOutstandingQuestions
// ---------------------------------------------------------------------------

func TestCountOutstandingQuestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name: "three OQ items",
			content: `# Feature: Test

## Outstanding Questions

- Question 1
- Question 2
- Question 3
`,
			want: 3,
		},
		{
			name: "no OQ section returns zero",
			content: `# Feature: Test

## Summary

Just a summary.
`,
			want: 0,
		},
		{
			name: "OQ section with None text returns zero",
			content: `# Feature: Test

## Outstanding Questions

None at this time.
`,
			want: 0,
		},
		{
			name: "OQ section with single item",
			content: `# Feature: Test

## Outstanding Questions

- Only one question
`,
			want: 1,
		},
		{
			name: "OQ section stops at next heading",
			content: `# Feature: Test

## Outstanding Questions

- Question A
- Question B

## Dependencies

- dep1
`,
			want: 2,
		},
		{
			name: "OQ with mixed content counts only bullet items",
			content: `# Feature: Test

## Outstanding Questions

Some preamble text.

- Actual question 1

More text between items.

- Actual question 2
`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			readmePath := filepath.Join(dir, "README.md")
			if err := os.WriteFile(readmePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			got, err := countOutstandingQuestions(readmePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountOutstandingQuestions_NonexistentFile(t *testing.T) {
	t.Parallel()

	_, err := countOutstandingQuestions("/tmp/does-not-exist-at-all/README.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// ---------------------------------------------------------------------------
// resolveFields
// ---------------------------------------------------------------------------

func TestResolveFields(t *testing.T) {
	t.Parallel()

	readmeContent := `# Feature: Test Feature

**Status:** Approved

## Dependencies

- some-dep
- another-dep

## Outstanding Questions

- Question 1
- Question 2
- Question 3
`

	featuresDir := setupTestFeatures(t, map[string]string{
		"test-feature": readmeContent,
		"some-dep":     "# Feature: Some Dep\n\n**Status:** Draft\n",
		"another-dep":  "# Feature: Another Dep\n\n**Status:** Approved\n",
	})

	t.Run("status field", func(t *testing.T) {
		t.Parallel()
		ef := resolveFields(featuresDir, "test-feature", []string{"status"})

		if ef.Path != "test-feature" {
			t.Errorf("Path = %q, want %q", ef.Path, "test-feature")
		}
		if ef.Status != "Approved" {
			t.Errorf("Status = %q, want %q", ef.Status, "Approved")
		}
		if ef.OQ != nil {
			t.Errorf("OQ should be nil when not requested, got %v", *ef.OQ)
		}
	})

	t.Run("oq field", func(t *testing.T) {
		t.Parallel()
		ef := resolveFields(featuresDir, "test-feature", []string{"oq"})

		if ef.OQ == nil {
			t.Fatal("OQ should not be nil")
		}
		if *ef.OQ != 3 {
			t.Errorf("OQ = %d, want 3", *ef.OQ)
		}
		if ef.Status != "" {
			t.Errorf("Status should be empty when not requested, got %q", ef.Status)
		}
	})

	t.Run("deps field", func(t *testing.T) {
		t.Parallel()
		ef := resolveFields(featuresDir, "test-feature", []string{"deps"})

		if len(ef.Deps) != 2 {
			t.Fatalf("got %d deps %v, want 2", len(ef.Deps), ef.Deps)
		}
		// parseDependencies returns bare IDs in file order; verify both are present.
		wantDeps := map[string]bool{"some-dep": true, "another-dep": true}
		for _, d := range ef.Deps {
			if !wantDeps[d] {
				t.Errorf("unexpected dep %q", d)
			}
			delete(wantDeps, d)
		}
		if len(wantDeps) > 0 {
			t.Errorf("missing deps: %v", wantDeps)
		}
	})

	t.Run("empty fields returns enrichedFeature with only Path", func(t *testing.T) {
		t.Parallel()
		ef := resolveFields(featuresDir, "test-feature", nil)

		if ef.Path != "test-feature" {
			t.Errorf("Path = %q, want %q", ef.Path, "test-feature")
		}
		if ef.Status != "" {
			t.Errorf("Status should be empty, got %q", ef.Status)
		}
		if ef.OQ != nil {
			t.Errorf("OQ should be nil, got %v", *ef.OQ)
		}
		if ef.Deps != nil {
			t.Errorf("Deps should be nil, got %v", ef.Deps)
		}
	})

	t.Run("multiple fields at once", func(t *testing.T) {
		t.Parallel()
		ef := resolveFields(featuresDir, "test-feature", []string{"status", "oq", "deps"})

		if ef.Status != "Approved" {
			t.Errorf("Status = %q, want %q", ef.Status, "Approved")
		}
		if ef.OQ == nil || *ef.OQ != 3 {
			t.Errorf("OQ = %v, want 3", ef.OQ)
		}
		if len(ef.Deps) != 2 {
			t.Errorf("got %d deps, want 2", len(ef.Deps))
		}
	})
}

// ---------------------------------------------------------------------------
// enrichedFeature YAML output
// ---------------------------------------------------------------------------

func TestWriteEnrichedYAML(t *testing.T) {
	t.Parallel()

	t.Run("full feature with all fields", func(t *testing.T) {
		t.Parallel()

		oq := 3
		features := []*enrichedFeature{
			{
				Path:   "test-feature",
				Status: "Approved",
				OQ:     &oq,
				Deps:   []string{"dep-a", "dep-b"},
			},
		}

		var buf bytes.Buffer
		if err := writeEnrichedYAML(&buf, features); err != nil {
			t.Fatalf("writeEnrichedYAML error: %v", err)
		}

		output := buf.String()
		for _, want := range []string{"path: test-feature", "status: Approved", "oq: 3", "deps:", "- dep-a", "- dep-b"} {
			if !strings.Contains(output, want) {
				t.Errorf("YAML output missing %q:\n%s", want, output)
			}
		}
	})

	t.Run("omitempty hides unrequested fields", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "minimal-feature"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedYAML(&buf, features); err != nil {
			t.Fatalf("writeEnrichedYAML error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "path: minimal-feature") {
			t.Errorf("YAML output missing path:\n%s", output)
		}
		for _, absent := range []string{"status:", "oq:", "deps:", "refs:", "plans:", "proposals:", "children:", "focus:", "cycle:"} {
			if strings.Contains(output, absent) {
				t.Errorf("YAML output should not contain %q (omitempty):\n%s", absent, output)
			}
		}
	})

	t.Run("multiple features", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "feature-a", Status: "Draft"},
			{Path: "feature-b", Status: "Approved"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedYAML(&buf, features); err != nil {
			t.Fatalf("writeEnrichedYAML error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "feature-a") || !strings.Contains(output, "feature-b") {
			t.Errorf("YAML output missing features:\n%s", output)
		}
	})

	t.Run("roundtrip YAML decode", func(t *testing.T) {
		t.Parallel()

		oq := 2
		focus := true
		features := []*enrichedFeature{
			{
				Path:   "round-trip",
				Focus:  &focus,
				Status: "Approved",
				OQ:     &oq,
				Deps:   []string{"dep-x"},
			},
		}

		var buf bytes.Buffer
		if err := writeEnrichedYAML(&buf, features); err != nil {
			t.Fatalf("writeEnrichedYAML error: %v", err)
		}

		var decoded []enrichedFeature
		if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("yaml.Unmarshal error: %v", err)
		}
		if len(decoded) != 1 {
			t.Fatalf("got %d items, want 1", len(decoded))
		}
		if decoded[0].Path != "round-trip" {
			t.Errorf("Path = %q, want %q", decoded[0].Path, "round-trip")
		}
		if decoded[0].Status != "Approved" {
			t.Errorf("Status = %q, want %q", decoded[0].Status, "Approved")
		}
		if decoded[0].OQ == nil || *decoded[0].OQ != 2 {
			t.Errorf("OQ = %v, want 2", decoded[0].OQ)
		}
	})
}

// ---------------------------------------------------------------------------
// enrichedFeature JSON output
// ---------------------------------------------------------------------------

func TestWriteEnrichedJSON(t *testing.T) {
	t.Parallel()

	t.Run("full feature with all fields", func(t *testing.T) {
		t.Parallel()

		oq := 5
		features := []*enrichedFeature{
			{
				Path:   "json-feature",
				Status: "Draft",
				OQ:     &oq,
				Deps:   []string{"json-dep"},
				Refs:   []string{"json-ref"},
			},
		}

		var buf bytes.Buffer
		if err := writeEnrichedJSON(&buf, features); err != nil {
			t.Fatalf("writeEnrichedJSON error: %v", err)
		}

		output := buf.String()
		for _, want := range []string{`"path": "json-feature"`, `"status": "Draft"`, `"oq": 5`, `"deps"`, `"json-dep"`, `"refs"`, `"json-ref"`} {
			if !strings.Contains(output, want) {
				t.Errorf("JSON output missing %q:\n%s", want, output)
			}
		}
	})

	t.Run("omitempty hides unrequested fields", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "minimal-json"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedJSON(&buf, features); err != nil {
			t.Fatalf("writeEnrichedJSON error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, `"path": "minimal-json"`) {
			t.Errorf("JSON output missing path:\n%s", output)
		}
		for _, absent := range []string{`"status"`, `"oq"`, `"deps"`, `"refs"`, `"plans"`, `"proposals"`, `"children"`, `"focus"`, `"cycle"`} {
			if strings.Contains(output, absent) {
				t.Errorf("JSON output should not contain %q (omitempty):\n%s", absent, output)
			}
		}
	})

	t.Run("roundtrip JSON decode", func(t *testing.T) {
		t.Parallel()

		oq := 1
		cycle := true
		features := []*enrichedFeature{
			{
				Path:  "json-round",
				Cycle: &cycle,
				OQ:    &oq,
				Deps:  []string{"a", "b"},
			},
		}

		var buf bytes.Buffer
		if err := writeEnrichedJSON(&buf, features); err != nil {
			t.Fatalf("writeEnrichedJSON error: %v", err)
		}

		var decoded []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("json.Unmarshal error: %v", err)
		}
		if len(decoded) != 1 {
			t.Fatalf("got %d items, want 1", len(decoded))
		}
		if decoded[0]["path"] != "json-round" {
			t.Errorf("path = %v, want %q", decoded[0]["path"], "json-round")
		}
		if decoded[0]["cycle"] != true {
			t.Errorf("cycle = %v, want true", decoded[0]["cycle"])
		}
		// JSON decodes numbers as float64
		if decoded[0]["oq"] != float64(1) {
			t.Errorf("oq = %v, want 1", decoded[0]["oq"])
		}
	})
}

// ---------------------------------------------------------------------------
// writeEnrichedText
// ---------------------------------------------------------------------------

func TestWriteEnrichedText(t *testing.T) {
	t.Parallel()

	t.Run("simple feature with status", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "my-feat", Status: "Approved"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, []string{"status"}); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "my-feat") {
			t.Errorf("output missing feature path:\n%s", got)
		}
		if !strings.Contains(got, "status=Approved") {
			t.Errorf("output missing status:\n%s", got)
		}
	})

	t.Run("feature with oq count", func(t *testing.T) {
		t.Parallel()

		oq := 4
		features := []*enrichedFeature{
			{Path: "oq-feat", OQ: &oq},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, []string{"oq"}); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "oq=4") {
			t.Errorf("output missing oq:\n%s", got)
		}
	})

	t.Run("feature with deps list", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "dep-feat", Deps: []string{"x", "y"}},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, []string{"deps"}); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "deps=[x,y]") {
			t.Errorf("output missing deps:\n%s", got)
		}
	})

	t.Run("focus marker", func(t *testing.T) {
		t.Parallel()

		focus := true
		features := []*enrichedFeature{
			{Path: "focused-feat", Focus: &focus, Status: "Draft"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, []string{"status"}); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "* focused-feat") {
			t.Errorf("output missing focus marker:\n%s", got)
		}
	})

	t.Run("cycle marker", func(t *testing.T) {
		t.Parallel()

		cycle := true
		features := []*enrichedFeature{
			{Path: "cycle-feat", Cycle: &cycle},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, nil); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "cycle-feat (cycle)") {
			t.Errorf("output missing cycle marker:\n%s", got)
		}
	})

	t.Run("nested children with indentation", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{
				Path:   "parent",
				Status: "Approved",
				Children: []*enrichedFeature{
					{Path: "parent/child", Status: "Draft"},
				},
			},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, []string{"status"}); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		if !strings.Contains(got, "parent") {
			t.Errorf("output missing parent:\n%s", got)
		}
		if !strings.Contains(got, "\tparent/child") {
			t.Errorf("output missing indented child:\n%s", got)
		}
	})

	t.Run("no fields produces plain path output", func(t *testing.T) {
		t.Parallel()

		features := []*enrichedFeature{
			{Path: "plain-feat"},
		}

		var buf bytes.Buffer
		if err := writeEnrichedText(&buf, features, nil); err != nil {
			t.Fatalf("writeEnrichedText error: %v", err)
		}

		got := buf.String()
		want := "plain-feat\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// boolPtr
// ---------------------------------------------------------------------------

func TestBoolPtr(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		p := boolPtr(true)
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *p != true {
			t.Errorf("got %v, want true", *p)
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		p := boolPtr(false)
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *p != false {
			t.Errorf("got %v, want false", *p)
		}
	})

	t.Run("returns distinct pointers", func(t *testing.T) {
		t.Parallel()
		p1 := boolPtr(true)
		p2 := boolPtr(true)
		if p1 == p2 {
			t.Error("expected distinct pointers for separate calls")
		}
	})
}

// ---------------------------------------------------------------------------
// validateFormat
// ---------------------------------------------------------------------------

func TestValidateFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{name: "text is valid", format: "text", wantErr: false},
		{name: "yaml is valid", format: "yaml", wantErr: false},
		{name: "json is valid", format: "json", wantErr: false},
		{name: "invalid format", format: "csv", wantErr: true},
		{name: "empty format", format: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateFormat(tt.format)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// writeEnrichedOutput dispatching
// ---------------------------------------------------------------------------

func TestWriteEnrichedOutput(t *testing.T) {
	t.Parallel()

	oq := 2
	features := []*enrichedFeature{
		{Path: "dispatch-feat", Status: "Approved", OQ: &oq},
	}

	t.Run("yaml format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := writeEnrichedOutput(&buf, features, []string{"status", "oq"}, "yaml"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "path: dispatch-feat") {
			t.Errorf("YAML output missing path:\n%s", output)
		}
	})

	t.Run("json format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := writeEnrichedOutput(&buf, features, []string{"status", "oq"}, "json"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, `"path": "dispatch-feat"`) {
			t.Errorf("JSON output missing path:\n%s", output)
		}
	})

	t.Run("text format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := writeEnrichedOutput(&buf, features, []string{"status", "oq"}, "text"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "dispatch-feat") {
			t.Errorf("text output missing path:\n%s", output)
		}
		if !strings.Contains(output, "status=Approved") {
			t.Errorf("text output missing status:\n%s", output)
		}
	})
}
