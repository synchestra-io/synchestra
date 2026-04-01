package feature

// Features implemented: cli/feature/info
// Features depended on:  cli/feature

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/feature"
	"gopkg.in/yaml.v3"
)

func infoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <feature_id>",
		Short: "Show feature metadata, section TOC, and children",
		Long: `Returns structured metadata and a section table-of-contents with line
ranges for a feature's README.md, enabling agents to surgically read only
the sections they need. Default output is YAML; use --format for JSON or text.`,
		Args: cobra.ExactArgs(1),
		RunE: runInfo,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json, text")
	return cmd
}

func runInfo(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")
	formatFlag, _ := cmd.Flags().GetString("format")

	if formatFlag != "yaml" && formatFlag != "json" && formatFlag != "text" {
		return exitcode.InvalidArgsErrorf("invalid format: %s (supported: yaml, json, text)", formatFlag)
	}

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	if !feature.Exists(featuresDir, featureID) {
		return exitcode.NotFoundErrorf("feature not found: %s", featureID)
	}

	info, err := feature.GetInfo(featuresDir, featureID)
	if err != nil {
		return exitcode.UnexpectedErrorf("%v", err)
	}

	return writeFeatureInfo(cmd.OutOrStdout(), formatFlag, *info)
}

// writeFeatureInfo encodes info to w in the given format (yaml, json, or text).
func writeFeatureInfo(w io.Writer, formatFlag string, info feature.Info) error {
	switch formatFlag {
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(info); err != nil {
			return exitcode.UnexpectedErrorf("encoding yaml: %v", err)
		}
		return enc.Close()
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	case "text":
		return writeTextInfo(w, info)
	}
	return nil
}

// writeTextInfo writes the feature info in a human-readable text format.
func writeTextInfo(w io.Writer, info feature.Info) error {
	bw := bufio.NewWriter(w)

	_, _ = fmt.Fprintf(bw, "Feature: %s\n", info.Path)
	_, _ = fmt.Fprintf(bw, "Status:  %s\n", info.Status)

	if len(info.Deps) > 0 {
		_, _ = fmt.Fprintf(bw, "Deps:    %s\n", strings.Join(info.Deps, ", "))
	} else {
		_, _ = fmt.Fprintln(bw, "Deps:    (none)")
	}

	if len(info.Refs) > 0 {
		_, _ = fmt.Fprintf(bw, "Refs:    %s\n", strings.Join(info.Refs, ", "))
	} else {
		_, _ = fmt.Fprintln(bw, "Refs:    (none)")
	}

	if len(info.Children) > 0 {
		_, _ = fmt.Fprintln(bw, "\nChildren:")
		for _, c := range info.Children {
			marker := "✓"
			if !c.InReadme {
				marker = "✗"
			}
			_, _ = fmt.Fprintf(bw, "  %s %s (in_readme: %v)\n", marker, c.Path, c.InReadme)
		}
	}

	if len(info.Plans) > 0 {
		_, _ = fmt.Fprintf(bw, "\nPlans:   %s\n", strings.Join(info.Plans, ", "))
	}

	if len(info.Sections) > 0 {
		_, _ = fmt.Fprintln(bw, "\nSections:")
		printTextSections(bw, info.Sections, 1)
	}

	return bw.Flush()
}

func printTextSections(w *bufio.Writer, sections []feature.SectionInfo, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, s := range sections {
		itemsSuffix := ""
		if s.Items > 0 {
			itemsSuffix = fmt.Sprintf(" (%d items)", s.Items)
		}
		_, _ = fmt.Fprintf(w, "%s%s [%s]%s\n", indent, s.Title, s.Lines, itemsSuffix)
		if len(s.Children) > 0 {
			printTextSections(w, s.Children, depth+1)
		}
	}
}
