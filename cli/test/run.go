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

func runCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [path]",
		Short: "Run test scenario files",
		Long:  "Run one or more test scenario .md files. Pass a file path or directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runRun,
	}
	cmd.Flags().StringSlice("tag", nil, "filter scenarios by tag")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	specRoot := "spec" // TODO: read from synchestra-spec.yaml project_dirs.specifications
	tags, _ := cmd.Flags().GetStringSlice("tag")

	target := specRoot + "/tests"
	if len(args) > 0 {
		target = args[0]
	}

	files, err := collectScenarioFiles(target)
	if err != nil {
		return fmt.Errorf("collecting scenarios: %w", err)
	}

	anyFailed := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}
		scenario, err := testscenario.ParseScenario(data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", f, err)
		}
		if len(tags) > 0 && !matchesTags(scenario.Tags, tags) {
			continue
		}
		runner := testscenario.NewRunner(testscenario.RunnerConfig{SpecRoot: specRoot})
		result := runner.Run(scenario)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), testscenario.FormatResult(result))
		if !result.Passed {
			anyFailed = true
		}
	}
	if anyFailed {
		return fmt.Errorf("one or more scenarios failed")
	}
	return nil
}

func collectScenarioFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	var files []string
	return files, filepath.Walk(path, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !i.IsDir() && strings.HasSuffix(p, ".md") && i.Name() != "README.md" {
			files = append(files, p)
		}
		return nil
	})
}

func matchesTags(scenarioTags, filterTags []string) bool {
	tagSet := make(map[string]bool, len(scenarioTags))
	for _, t := range scenarioTags {
		tagSet[t] = true
	}
	for _, ft := range filterTags {
		if tagSet[ft] {
			return true
		}
	}
	return false
}
