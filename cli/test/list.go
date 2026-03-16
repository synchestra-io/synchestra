package test

// Features implemented: cli
// Features depended on:  testing-framework/test-scenario, testing-framework/test-runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/pkg/testscenario"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available test scenarios",
		RunE:  runList,
	}
	cmd.Flags().StringSlice("tag", nil, "filter scenarios by tag")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	specRoot := "spec" // TODO: read from synchestra-spec.yaml
	tags, _ := cmd.Flags().GetStringSlice("tag")

	var allFiles []string
	for _, dir := range []string{
		filepath.Join(specRoot, "tests"),
	} {
		files, err := collectScenarioFiles(dir)
		if err != nil {
			continue
		}
		allFiles = append(allFiles, files...)
	}
	featuresDir := filepath.Join(specRoot, "features")
	_ = filepath.Walk(featuresDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "_tests" {
			files, _ := collectScenarioFiles(p)
			allFiles = append(allFiles, files...)
		}
		return nil
	})

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n", "SCENARIO", "DESCRIPTION", "TAGS")
	for _, f := range allFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		scenario, err := testscenario.ParseScenario(data)
		if err != nil {
			continue
		}
		if len(tags) > 0 && !matchesTags(scenario.Tags, tags) {
			continue
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n",
			f, scenario.Description, strings.Join(scenario.Tags, ", "))
	}
	return nil
}
