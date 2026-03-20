package feature

// Features implemented: cli/feature/list

import (
	"fmt"

	"github.com/spf13/cobra"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feature IDs, one per line",
		Long: `Lists all features in a project as full feature IDs, one per line,
sorted alphabetically. Each line is a feature ID — the path relative to
the project's features directory using / as separator.`,
		Args: cobra.NoArgs,
		RunE: runList,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	features, err := discoverFeatures(featuresDir)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("discovering features: %v", err)}
	}

	w := cmd.OutOrStdout()
	for _, id := range features {
		_, _ = fmt.Fprintln(w, id)
	}
	return nil
}
