package feature

// Features implemented: cli/feature

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	specfeature "github.com/synchestra-io/specscore/pkg/feature"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// enrichedFeature YAML output
// ---------------------------------------------------------------------------

func TestWriteEnrichedYAML(t *testing.T) {
	t.Parallel()

	t.Run("full feature with all fields", func(t *testing.T) {
		t.Parallel()

		oq := 3
		features := []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
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
		for _, absent := range []string{"status:", "oq:", "deps:", "refs:", "plans:", "proposals:", "focus:", "cycle:"} {
			if strings.Contains(output, absent) {
				t.Errorf("YAML output should not contain %q (omitempty):\n%s", absent, output)
			}
		}
	})

	t.Run("multiple features", func(t *testing.T) {
		t.Parallel()

		features := []*specfeature.EnrichedFeature{
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
		features := []*specfeature.EnrichedFeature{
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

		var decoded []specfeature.EnrichedFeature
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
		features := []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
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
		for _, absent := range []string{`"status"`, `"oq"`, `"deps"`, `"refs"`, `"plans"`, `"proposals"`, `"focus"`, `"cycle"`} {
			if strings.Contains(output, absent) {
				t.Errorf("JSON output should not contain %q (omitempty):\n%s", absent, output)
			}
		}
	})

	t.Run("roundtrip JSON decode", func(t *testing.T) {
		t.Parallel()

		oq := 1
		cycle := true
		features := []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
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
		features := []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
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
		features := []*specfeature.EnrichedFeature{
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
		features := []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
			{
				Path:   "parent",
				Status: "Approved",
				ChildNodes: []*specfeature.EnrichedFeature{
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

		features := []*specfeature.EnrichedFeature{
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
// writeEnrichedOutput dispatching
// ---------------------------------------------------------------------------

func TestWriteEnrichedOutput(t *testing.T) {
	t.Parallel()

	oq := 2
	features := []*specfeature.EnrichedFeature{
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
