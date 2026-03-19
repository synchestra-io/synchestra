package feature

// Features implemented: cli/feature/deps

import (
	"fmt"

	"github.com/spf13/cobra"
)

func depsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps <feature_id>",
		Short: "Show features that a given feature depends on",
		Long: `Shows the features that a given feature depends on. Dependencies are
read from the ## Dependencies section in the feature's README.md. Each
dependency is output as a feature ID, one per line.`,
		Args: cobra.ExactArgs(1),
		RunE: runDeps,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runDeps(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	if !featureExists(featuresDir, featureID) {
		return &exitError{code: 3, msg: fmt.Sprintf("feature not found: %s", featureID)}
	}

	readmePath := featureReadmePath(featuresDir, featureID)
	deps, err := parseDependencies(readmePath)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("reading feature %s: %v", featureID, err)}
	}

	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()
	for _, dep := range deps {
		if !featureExists(featuresDir, dep) {
			fmt.Fprintf(errW, "%s (not found)\n", dep)
		} else {
			fmt.Fprintln(w, dep)
		}
	}
	return nil
}
