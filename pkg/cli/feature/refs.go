package feature

// Features implemented: cli/feature/refs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
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

	if !featureExists(featuresDir, featureID) {
		return exitcode.NotFoundErrorf("feature not found: %s", featureID)
	}

	w := cmd.OutOrStdout()

	if transitive {
		nodes := resolveTransitiveRefs(featuresDir, featureID)
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
	allFeatures, err := discoverFeatures(featuresDir)
	if err != nil {
		return exitcode.UnexpectedErrorf("discovering features: %v", err)
	}

	var refs []string
	for _, fID := range allFeatures {
		if fID == featureID {
			continue
		}
		readmePath := featureReadmePath(featuresDir, fID)
		deps, err := parseDependencies(readmePath)
		if err != nil {
			continue
		}
		for _, dep := range deps {
			if dep == featureID {
				refs = append(refs, fID)
				break
			}
		}
	}
	sort.Strings(refs)

	if len(fields) > 0 || format == "yaml" || format == "json" {
		var enriched []*enrichedFeature
		for _, ref := range refs {
			ef := resolveFields(featuresDir, ref, fields)
			enriched = append(enriched, ef)
		}
		return writeEnrichedOutput(w, enriched, fields, format)
	}

	for _, ref := range refs {
		_, _ = fmt.Fprintln(w, ref)
	}
	return nil
}
