package feature

// Features implemented: cli/feature/refs

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func refsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs <feature_id>",
		Short: "Show features that reference a given feature as a dependency",
		Long: `Shows features that reference (depend on) a given feature. This is the
inverse of deps — it scans all features' ## Dependencies sections to find
those that list the given feature ID.`,
		Args: cobra.ExactArgs(1),
		RunE: runRefs,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runRefs(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	if !featureExists(featuresDir, featureID) {
		return &exitError{code: 3, msg: fmt.Sprintf("feature not found: %s", featureID)}
	}

	allFeatures, err := discoverFeatures(featuresDir)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("discovering features: %v", err)}
	}

	var refs []string
	for _, fID := range allFeatures {
		if fID == featureID {
			continue
		}
		readmePath := featureReadmePath(featuresDir, fID)
		deps, err := parseDependencies(readmePath)
		if err != nil {
			continue // skip features with unreadable READMEs
		}
		for _, dep := range deps {
			if dep == featureID {
				refs = append(refs, fID)
				break
			}
		}
	}

	sort.Strings(refs)
	w := cmd.OutOrStdout()
	for _, ref := range refs {
		_, _ = fmt.Fprintln(w, ref)
	}
	return nil
}
