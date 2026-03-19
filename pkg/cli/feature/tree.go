package feature

// Features implemented: cli/feature/tree

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func treeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tree",
		Short: "Display feature hierarchy as an indented tree",
		Long: `Displays the feature hierarchy as an indented tree. Top-level features
are printed at the root, and nested features are indented with tabs to
show parent-child relationships.`,
		Args: cobra.NoArgs,
		RunE: runTree,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runTree(cmd *cobra.Command, _ []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	features, err := discoverFeatures(featuresDir)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("discovering features: %v", err)}
	}

	roots := buildTree(features)
	var sb strings.Builder
	printTree(&sb, roots, 0)

	fmt.Fprint(cmd.OutOrStdout(), sb.String())
	return nil
}
