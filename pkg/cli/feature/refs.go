package feature

// Features implemented: cli/feature/refs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/feature"
)

func refsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs <feature_id>",
		Short: "Show features that reference a given feature as a dependency",
		Long: `Shows features that reference (depend on) a given feature. This is the
inverse of deps — it scans all features' ## Dependencies sections to find
those that list the given feature ID. Use --transitive to follow the full chain.`,
		Args: cobra.ExactArgs(1),
		RunE: runRefs,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("fields", "", "comma-separated metadata fields to include (e.g., status,oq)")
	cmd.Flags().String("format", "", "output format: yaml, json, text (auto-selects yaml when --fields is set)")
	cmd.Flags().Bool("transitive", false, "follow reference chain recursively")
	return cmd
}

func runRefs(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")
	fieldsFlag, _ := cmd.Flags().GetString("fields")
	transitive, _ := cmd.Flags().GetBool("transitive")

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

	if !feature.Exists(featuresDir, featureID) {
		return exitcode.NotFoundErrorf("feature not found: %s", featureID)
	}

	w := cmd.OutOrStdout()

	if transitive {
		nodes := feature.TransitiveRefs(featuresDir, featureID)
		if len(fields) > 0 {
			feature.EnrichTransitiveNodes(featuresDir, nodes, fields)
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
			feature.PrintTransitiveText(&sb, nodes, 0)
			_, _ = fmt.Fprint(w, sb.String())
		}
		return nil
	}

	// Non-transitive
	refs, err := feature.FindFeatureRefs(featuresDir, featureID)
	if err != nil {
		return exitcode.UnexpectedErrorf("finding references: %v", err)
	}

	if len(fields) > 0 || format == "yaml" || format == "json" {
		var enriched []*feature.EnrichedFeature
		for _, ref := range refs {
			ef, _ := feature.ResolveFields(featuresDir, ref, fields)
			enriched = append(enriched, ef)
		}
		return writeEnrichedOutput(w, enriched, fields, format)
	}

	for _, ref := range refs {
		_, _ = fmt.Fprintln(w, ref)
	}
	return nil
}
