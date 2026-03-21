package feature

// Features implemented: cli/feature/deps

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func depsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps <feature_id>",
		Short: "Show features that a given feature depends on",
		Long: `Shows the features that a given feature depends on. Dependencies are
read from the ## Dependencies section in the feature's README.md. Use --transitive
to follow the full dependency chain. Use --fields to include metadata.`,
		Args: cobra.ExactArgs(1),
		RunE: runDeps,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("fields", "", "comma-separated metadata fields to include (e.g., status,oq)")
	cmd.Flags().String("format", "", "output format: yaml, json, text (auto-selects yaml when --fields is set)")
	cmd.Flags().Bool("transitive", false, "follow dependency chain recursively")
	return cmd
}

func runDeps(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")
	fieldsFlag, _ := cmd.Flags().GetString("fields")
	transitive, _ := cmd.Flags().GetBool("transitive")

	fields, err := parseFieldNames(fieldsFlag)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	format := effectiveFormat(cmd)
	if err := validateFormat(format); err != nil {
		return err
	}

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	if !featureExists(featuresDir, featureID) {
		return &exitError{code: 3, msg: fmt.Sprintf("feature not found: %s", featureID)}
	}

	w := cmd.OutOrStdout()

	if transitive {
		nodes := resolveTransitiveDeps(featuresDir, featureID)
		if len(fields) > 0 {
			enrichTransitiveNodes(featuresDir, nodes, fields)
		}
		switch format {
		case "yaml":
			return writeEnrichedYAML(w, nodes)
		case "json":
			return writeEnrichedJSON(w, nodes)
		default:
			if len(fields) > 0 {
				return writeEnrichedText(w, nodes, fields)
			}
			var sb strings.Builder
			printTransitiveText(&sb, nodes, 0)
			_, _ = fmt.Fprint(w, sb.String())
		}
		return nil
	}

	// Non-transitive
	readmePath := featureReadmePath(featuresDir, featureID)
	deps, err := parseDependencies(readmePath)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("reading feature %s: %v", featureID, err)}
	}

	if len(fields) > 0 || format == "yaml" || format == "json" {
		var enriched []*enrichedFeature
		for _, dep := range deps {
			ef := resolveFields(featuresDir, dep, fields)
			enriched = append(enriched, ef)
		}
		return writeEnrichedOutput(w, enriched, fields, format)
	}

	// Plain text output (original behavior)
	errW := cmd.ErrOrStderr()
	for _, dep := range deps {
		if !featureExists(featuresDir, dep) {
			_, _ = fmt.Fprintf(errW, "%s (not found)\n", dep)
		} else {
			_, _ = fmt.Fprintln(w, dep)
		}
	}
	return nil
}
