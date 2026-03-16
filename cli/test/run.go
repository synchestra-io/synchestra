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
	cmd.Flags().String("format", "text", "output format: text or json")
	cmd.Flags().String("spec-root", "", "override spec root directory")
	cmd.Flags().Bool("run-manual-tests", false, "include scenarios tagged 'manual'")
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	specRoot := "spec" // TODO: read from synchestra-spec.yaml project_dirs.specifications
	if sr, _ := cmd.Flags().GetString("spec-root"); sr != "" {
		specRoot = sr
	}
	tags, _ := cmd.Flags().GetStringSlice("tag")
	format, _ := cmd.Flags().GetString("format")
	runManual, _ := cmd.Flags().GetBool("run-manual-tests")

	// When a specific file is passed, always run it (even if manual).
	// When scanning a directory, skip manual scenarios unless --run-manual-tests.
	specificFile := false
	target := specRoot + "/tests"
	if len(args) > 0 {
		target = args[0]
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			specificFile = true
		}
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
		if !specificFile && !runManual && hasTag(scenario.Tags, "manual") {
			continue
		}
		if len(tags) > 0 && !matchesTags(scenario.Tags, tags) {
			continue
		}
		cfg := testscenario.RunnerConfig{SpecRoot: specRoot}
		if format == "text" {
			cfg.Progress = testscenario.NewLiveReporter(cmd.OutOrStdout())
		}
		runner := testscenario.NewRunner(cfg)
		result := runner.Run(scenario)
		if format == "json" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), testscenario.FormatResultJSON(result))
		}
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

func hasTag(scenarioTags []string, tag string) bool {
	for _, t := range scenarioTags {
		if t == tag {
			return true
		}
	}
	return false
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
