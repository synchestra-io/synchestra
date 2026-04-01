package code

// Features implemented: cli/code/deps
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/code/deps

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/sourceref"
)

func depsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps [flags]",
		Short: "Show Synchestra resources that source files depend on",
		Long: `Shows the Synchestra resources (features, plans, docs) that source files depend on.
Scans source files for synchestra: annotations and https://synchestra.io/ URLs
in comments and lists the referenced resources.

This is a read-only command that scans the working tree and does not mutate anything.`,
		RunE: runDeps,
	}

	cmd.Flags().String("path", "**/*", "Glob pattern to select files (e.g., pkg/**/*.go, src/*/*_test.go). Defaults to **/* (all files)")
	cmd.Flags().String("project", "", "Project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("type", "", "Filter results to a specific resource type: feature, plan, or doc")

	return cmd
}

func runDeps(cmd *cobra.Command, _ []string) error {
	pathPattern, _ := cmd.Flags().GetString("path")
	typeFilter, _ := cmd.Flags().GetString("type")

	// Validate type filter
	if typeFilter != "" && typeFilter != "feature" && typeFilter != "plan" && typeFilter != "doc" {
		return exitcode.InvalidArgsErrorf("invalid --type value: %s (must be feature, plan, or doc)", typeFilter)
	}

	// Expand glob pattern
	files, err := sourceref.ExpandGlobPattern(pathPattern)
	if err != nil {
		return exitcode.InvalidArgsErrorf("invalid glob pattern %q: %v", pathPattern, err)
	}

	if len(files) == 0 {
		// No files matched the pattern
		return nil
	}

	// Scan files for references
	result, err := sourceref.ScanFiles(files)
	if err != nil {
		return exitcode.UnexpectedErrorf("scanning files: %v", err)
	}

	// Determine if we have a single file match
	singleFile := len(result.FileRefs) == 1

	// Format and output
	w := cmd.OutOrStdout()
	output := sourceref.FormatOutput(result, singleFile, typeFilter)
	if output != "" {
		_, _ = fmt.Fprint(w, output)
	}

	return nil
}
