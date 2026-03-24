package feature

// Features implemented: cli/feature/list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feature IDs, one per line",
		Long: `Lists all features in a project as full feature IDs, one per line,
sorted alphabetically. Use --fields to include metadata for each feature.`,
		Args: cobra.NoArgs,
		RunE: runList,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("fields", "", "comma-separated metadata fields to include (e.g., status,oq)")
	cmd.Flags().String("format", "", "output format: yaml, json, text (auto-selects yaml when --fields is set)")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")
	fieldsFlag, _ := cmd.Flags().GetString("fields")

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

	if len(fields) > 0 || format == "yaml" || format == "json" {
		var enriched []*enrichedFeature
		for _, id := range features {
			ef := resolveFields(featuresDir, id, fields)
			enriched = append(enriched, ef)
		}
		return writeEnrichedOutput(w, enriched, fields, format)
	}

	for _, id := range features {
		_, _ = fmt.Fprintln(w, id)
	}
	return nil
}
