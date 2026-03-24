package feature

// Features implemented: cli/feature/tree

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
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

	fields, err := parseFieldNames(fieldsFlag)
	if err != nil {
		return exitcode.InvalidArgsError(err.Error())
	}

	format := effectiveFormat(cmd)
	if err := validateFormat(format); err != nil {
		return err
	}

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	features, err := discoverFeatures(featuresDir)
	if err != nil {
		return exitcode.UnexpectedErrorf("discovering features: %v", err)
	}

	w := cmd.OutOrStdout()

	// Determine target feature ID (if any)
	var targetID string
	if len(args) > 0 {
		targetID = args[0]
		if !featureExists(featuresDir, targetID) {
			return exitcode.NotFoundErrorf("feature not found: %s", targetID)
		}
	}

	// Enriched output path (--fields or structured format)
	if len(fields) > 0 || format == "yaml" || format == "json" {
		return writeTreeWithFields(w, featuresDir, features, targetID, directionFlag, fields, format)
	}

	// Plain text tree
	if targetID == "" {
		roots := buildTree(features)
		var sb strings.Builder
		printTree(&sb, roots, 0)
		_, _ = fmt.Fprint(w, sb.String())
		return nil
	}

	// Focused text tree
	filtered := filterFocusedFeatures(features, targetID, directionFlag)
	roots := buildTree(filtered)
	markFocus(roots, targetID)
	var sb strings.Builder
	printTree(&sb, roots, 0)
	_, _ = fmt.Fprint(w, sb.String())
	return nil
}

// filterFocusedFeatures returns features relevant to a focused tree view.
func filterFocusedFeatures(allFeatures []string, targetID, direction string) []string {
	include := make(map[string]bool)
	include[targetID] = true

	if direction != "down" {
		parts := strings.Split(targetID, "/")
		for i := 1; i < len(parts); i++ {
			ancestor := strings.Join(parts[:i], "/")
			include[ancestor] = true
		}
	}

	if direction != "up" {
		prefix := targetID + "/"
		for _, f := range allFeatures {
			if strings.HasPrefix(f, prefix) {
				include[f] = true
			}
		}
	}

	var filtered []string
	for _, f := range allFeatures {
		if include[f] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// markFocus sets the focus flag on the target node in the tree.
func markFocus(nodes []*featureNode, targetID string) {
	for _, node := range nodes {
		if node.id == targetID {
			node.focus = true
			return
		}
		markFocus(node.children, targetID)
	}
}

// writeTreeWithFields outputs the tree as enriched YAML/JSON/text with field metadata.
func writeTreeWithFields(w io.Writer, featuresDir string, allFeatures []string, targetID, direction string, fields []string, format string) error {
	var filtered []string
	if targetID == "" {
		filtered = allFeatures
	} else {
		filtered = filterFocusedFeatures(allFeatures, targetID, direction)
	}

	roots := buildEnrichedTree(featuresDir, filtered, fields, targetID)

	switch format {
	case "yaml":
		return writeEnrichedYAML(w, roots)
	case "json":
		return writeEnrichedJSON(w, roots)
	default:
		return writeEnrichedText(w, roots, fields)
	}
}

// buildEnrichedTree builds a tree of enrichedFeature nodes with resolved fields.
func buildEnrichedTree(featuresDir string, featureIDs []string, fields []string, focusID string) []*enrichedFeature {
	// Filter out "children" from fields for tree output (tree structure IS children)
	treeFields := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "children" {
			treeFields = append(treeFields, f)
		}
	}

	nodeMap := make(map[string]*enrichedFeature)
	var roots []*enrichedFeature

	for _, id := range featureIDs {
		ef := resolveFields(featuresDir, id, treeFields)
		if id == focusID && focusID != "" {
			ef.Focus = boolPtr(true)
		}
		nodeMap[id] = ef

		parts := strings.Split(id, "/")
		if len(parts) == 1 {
			roots = append(roots, ef)
		} else {
			parentID := strings.Join(parts[:len(parts)-1], "/")
			if parent, ok := nodeMap[parentID]; ok {
				if children, ok := parent.Children.([]*enrichedFeature); ok {
					parent.Children = append(children, ef)
				} else {
					parent.Children = []*enrichedFeature{ef}
				}
			} else {
				roots = append(roots, ef)
			}
		}
	}

	return roots
}
