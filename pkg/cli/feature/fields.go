package feature

// Features implemented: cli/feature/list, cli/feature/tree, cli/feature/deps, cli/feature/refs
// Features depended on:  cli/feature
//
// This file contains CLI-specific output formatting for enriched features.
// Field parsing, validation, and resolution are now in specscore/pkg/feature.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/feature"
	"gopkg.in/yaml.v3"
)

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

// writeEnrichedYAML encodes enriched features as YAML.
func writeEnrichedYAML(w io.Writer, features []*feature.EnrichedFeature) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(features); err != nil {
		return err
	}
	return enc.Close()
}

// writeEnrichedJSON encodes enriched features as JSON.
func writeEnrichedJSON(w io.Writer, features []*feature.EnrichedFeature) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(features)
}

// writeEnrichedOutput writes enriched features in the specified format.
func writeEnrichedOutput(w io.Writer, features []*feature.EnrichedFeature, fields []string, format string) error {
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
func writeEnrichedText(w io.Writer, features []*feature.EnrichedFeature, fields []string) error {
	bw := bufio.NewWriter(w)
	for _, ef := range features {
		writeEnrichedTextNode(bw, ef, fields, 0)
	}
	return bw.Flush()
}

func writeEnrichedTextNode(w *bufio.Writer, ef *feature.EnrichedFeature, fields []string, depth int) {
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

	for _, child := range ef.ChildNodes {
		writeEnrichedTextNode(w, child, fields, depth+1)
	}
}
