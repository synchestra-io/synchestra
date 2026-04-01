package feature

// Features implemented: cli/feature/tree

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/feature"
)

func treeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tree [feature_id]",
		Short: "Display feature hierarchy as an indented tree",
		Long: `Displays the feature hierarchy as an indented tree. Without a feature ID,
shows the full project tree. With a feature ID, shows the feature in context —
ancestors (path to root) plus its subtree by default. Use --direction to narrow
to ancestors only (up) or subtree only (down).`,
		Args: cobra.MaximumNArgs(1),
		RunE: runTree,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("direction", "", "up (ancestors only) or down (subtree only); requires feature_id")
	cmd.Flags().String("fields", "", "comma-separated metadata fields to include (e.g., status,oq)")
	cmd.Flags().String("format", "", "output format: yaml, json, text (auto-selects yaml when --fields is set)")
	return cmd
}

func runTree(cmd *cobra.Command, args []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")
	directionFlag, _ := cmd.Flags().GetString("direction")
	fieldsFlag, _ := cmd.Flags().GetString("fields")

	if directionFlag != "" && directionFlag != "up" && directionFlag != "down" {
		return exitcode.InvalidArgsErrorf("invalid --direction: %s (valid: up, down)", directionFlag)
	}
	if directionFlag != "" && len(args) == 0 {
		return exitcode.InvalidArgsError("--direction requires a feature_id argument")
	}

	fields, err := feature.ParseFieldNames(fieldsFlag)
	if err != nil {
		return exitcode.InvalidArgsError(err.Error())
	}

	format := effectiveFormat(cmd)
	if err := feature.ValidateFormat(format); err != nil {
		return exitcode.InvalidArgsError(err.Error())
	}

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	discovered, err := feature.Discover(featuresDir)
	if err != nil {
		return exitcode.UnexpectedErrorf("discovering features: %v", err)
	}
	featureIDs := feature.FeatureIDs(discovered)

	w := cmd.OutOrStdout()

	// Determine target feature ID (if any)
	var targetID string
	if len(args) > 0 {
		targetID = args[0]
		if !feature.Exists(featuresDir, targetID) {
			return exitcode.NotFoundErrorf("feature not found: %s", targetID)
		}
	}

	// Enriched output path (--fields or structured format)
	if len(fields) > 0 || format == "yaml" || format == "json" {
		return writeTreeWithFields(w, featuresDir, featureIDs, targetID, directionFlag, fields, format)
	}

	// Plain text tree
	if targetID == "" {
		roots := feature.BuildTree(featureIDs)
		var sb strings.Builder
		feature.PrintTree(&sb, roots, 0)
		_, _ = fmt.Fprint(w, sb.String())
		return nil
	}

	// Focused text tree
	filtered := feature.FilterFocusedFeatures(featureIDs, targetID, directionFlag)
	roots := feature.BuildTree(filtered)
	feature.MarkFocus(roots, targetID)
	var sb strings.Builder
	feature.PrintTree(&sb, roots, 0)
	_, _ = fmt.Fprint(w, sb.String())
	return nil
}

// writeTreeWithFields outputs the tree as enriched YAML/JSON/text with field metadata.
func writeTreeWithFields(w io.Writer, featuresDir string, allFeatures []string, targetID, direction string, fields []string, format string) error {
	var filtered []string
	if targetID == "" {
		filtered = allFeatures
	} else {
		filtered = feature.FilterFocusedFeatures(allFeatures, targetID, direction)
	}

	roots := feature.BuildEnrichedTree(featuresDir, filtered, fields, targetID)

	switch format {
	case "yaml":
		return writeEnrichedYAML(w, roots)
	case "json":
		return writeEnrichedJSON(w, roots)
	default:
		return writeEnrichedText(w, roots, fields)
	}
}
