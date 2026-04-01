package feature

// Features implemented: cli/feature/list, cli/feature/tree, cli/feature/deps, cli/feature/refs
// Features depended on:  cli/feature

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"gopkg.in/yaml.v3"
)

// validFields lists all recognized --fields values.
var validFields = map[string]bool{
	"status":    true,
	"oq":        true,
	"deps":      true,
	"refs":      true,
	"children":  true,
	"plans":     true,
	"proposals": true,
}

// parseFieldNames validates and returns field names from a comma-separated string.
func parseFieldNames(fieldsStr string) ([]string, error) {
	if fieldsStr == "" {
		return nil, nil
	}
	parts := strings.Split(fieldsStr, ",")
	seen := make(map[string]bool)
	var fields []string
	for _, p := range parts {
		f := strings.TrimSpace(p)
		if f == "" {
			continue
		}
		if !validFields[f] {
			return nil, fmt.Errorf("unknown field %q (valid: status, oq, deps, refs, children, plans, proposals)", f)
		}
		if !seen[f] {
			seen[f] = true
			fields = append(fields, f)
		}
	}
	return fields, nil
}

// enrichedFeature holds a feature ID with optional metadata fields.
// Children can be []string (child paths for flat output) or []*enrichedFeature (tree nesting).
type enrichedFeature struct {
	Path      string      `yaml:"path" json:"path"`
	Focus     *bool       `yaml:"focus,omitempty" json:"focus,omitempty"`
	Cycle     *bool       `yaml:"cycle,omitempty" json:"cycle,omitempty"`
	Status    string      `yaml:"status,omitempty" json:"status,omitempty"`
	OQ        *int        `yaml:"oq,omitempty" json:"oq,omitempty"`
	Deps      []string    `yaml:"deps,omitempty" json:"deps,omitempty"`
	Refs      []string    `yaml:"refs,omitempty" json:"refs,omitempty"`
	Plans     []string    `yaml:"plans,omitempty" json:"plans,omitempty"`
	Proposals []string    `yaml:"proposals,omitempty" json:"proposals,omitempty"`
	Children  interface{} `yaml:"children,omitempty" json:"children,omitempty"`
}

// resolveFields computes the requested metadata fields for a feature.
func resolveFields(featuresDir, featureID string, fields []string) *enrichedFeature {
	ef := &enrichedFeature{Path: featureID}
	readmePath := featureReadmePath(featuresDir, featureID)

	for _, f := range fields {
		switch f {
		case "status":
			if s, err := parseFeatureStatus(readmePath); err == nil {
				ef.Status = s
			}
		case "oq":
			if n, err := countOutstandingQuestions(readmePath); err == nil {
				ef.OQ = &n
			}
		case "deps":
			if d, err := parseDependencies(readmePath); err == nil {
				ef.Deps = d
			}
		case "refs":
			if r, err := findFeatureRefs(featuresDir, featureID); err == nil {
				ef.Refs = r
			}
		case "children":
			if c, err := discoverChildFeatures(featuresDir, featureID, readmePath); err == nil {
				var paths []string
				for _, ch := range c {
					paths = append(paths, ch.Path)
				}
				if len(paths) > 0 {
					ef.Children = paths
				}
			}
		case "plans":
			specRoot := filepath.Dir(featuresDir)
			if p, err := findLinkedPlans(filepath.Dir(specRoot), featureID); err == nil {
				ef.Plans = p
			}
		case "proposals":
			// Proposals not yet implemented in the spec repo structure.
		}
	}
	return ef
}

// countOutstandingQuestions counts list items in the ## Outstanding Questions section.
func countOutstandingQuestions(readmePath string) (int, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }() // opened read-only; Close never flushes data so its error is irrelevant

	inOQ := false
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "## Outstanding Questions" {
			inOQ = true
			continue
		}
		if inOQ && strings.HasPrefix(line, "## ") {
			break
		}
		if inOQ && strings.HasPrefix(line, "- ") {
			count++
		}
	}
	return count, scanner.Err()
}

// effectiveFormat determines the output format from flags.
// With --fields and no explicit --format, auto-switches to YAML.
func effectiveFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString("format")
	if format != "" {
		return format
	}
	fields, _ := cmd.Flags().GetString("fields")
	if fields != "" {
		return "yaml"
	}
	return "text"
}

// validateFormat checks the format flag value is valid.
func validateFormat(format string) error {
	if format != "text" && format != "yaml" && format != "json" {
		return exitcode.InvalidArgsErrorf("invalid --format: %s (valid: text, yaml, json)", format)
	}
	return nil
}

// writeEnrichedYAML encodes enriched features as YAML.
func writeEnrichedYAML(w io.Writer, features []*enrichedFeature) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(features); err != nil {
		return err
	}
	return enc.Close()
}

// writeEnrichedJSON encodes enriched features as JSON.
func writeEnrichedJSON(w io.Writer, features []*enrichedFeature) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(features)
}

// writeEnrichedOutput writes enriched features in the specified format.
func writeEnrichedOutput(w io.Writer, features []*enrichedFeature, fields []string, format string) error {
	switch format {
	case "yaml":
		return writeEnrichedYAML(w, features)
	case "json":
		return writeEnrichedJSON(w, features)
	default:
		return writeEnrichedText(w, features, fields)
	}
}

// writeEnrichedText outputs enriched features as human-readable text.
func writeEnrichedText(w io.Writer, features []*enrichedFeature, fields []string) error {
	bw := bufio.NewWriter(w)
	for _, ef := range features {
		writeEnrichedTextNode(bw, ef, fields, 0)
	}
	return bw.Flush()
}

func writeEnrichedTextNode(w *bufio.Writer, ef *enrichedFeature, fields []string, depth int) {
	indent := strings.Repeat("\t", depth)
	prefix := ""
	if ef.Focus != nil && *ef.Focus {
		prefix = "* "
	}
	if ef.Cycle != nil && *ef.Cycle {
		_, _ = fmt.Fprintf(w, "%s%s (cycle)\n", indent, ef.Path)
		return
	}

	var meta []string
	for _, f := range fields {
		switch f {
		case "status":
			if ef.Status != "" {
				meta = append(meta, fmt.Sprintf("status=%s", ef.Status))
			}
		case "oq":
			if ef.OQ != nil {
				meta = append(meta, fmt.Sprintf("oq=%d", *ef.OQ))
			}
		case "deps":
			if len(ef.Deps) > 0 {
				meta = append(meta, fmt.Sprintf("deps=[%s]", strings.Join(ef.Deps, ",")))
			}
		case "refs":
			if len(ef.Refs) > 0 {
				meta = append(meta, fmt.Sprintf("refs=[%s]", strings.Join(ef.Refs, ",")))
			}
		case "plans":
			if len(ef.Plans) > 0 {
				meta = append(meta, fmt.Sprintf("plans=[%s]", strings.Join(ef.Plans, ",")))
			}
		case "proposals":
			if len(ef.Proposals) > 0 {
				meta = append(meta, fmt.Sprintf("proposals=[%s]", strings.Join(ef.Proposals, ",")))
			}
		}
	}

	suffix := ""
	if len(meta) > 0 {
		suffix = " " + strings.Join(meta, " ")
	}
	_, _ = fmt.Fprintf(w, "%s%s%s%s\n", indent, prefix, ef.Path, suffix)

	if children, ok := ef.Children.([]*enrichedFeature); ok {
		for _, child := range children {
			writeEnrichedTextNode(w, child, fields, depth+1)
		}
	}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
