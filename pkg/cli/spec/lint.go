package spec

// Features implemented: cli/spec/lint

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/lint"
	"gopkg.in/yaml.v3"
)

func lintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [PATH]",
		Short: "Validate spec tree for structural convention violations",
		Long: `Scans the specification tree and reports violations of structural
conventions (README.md files, Outstanding Questions sections, heading
levels, feature references, internal links, index entries).

Violations are categorized by severity: error (must fix), warning (should
fix), info (advisory). Exit code 0 = valid, 1 = violations found, 2 =
invalid arguments, 10+ = unexpected error.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine spec root
			specRoot := "./spec"
			if len(args) > 0 {
				specRoot = args[0]
			}

			// Parse flags
			rulesStr, _ := cmd.Flags().GetString("rules")
			ignoreStr, _ := cmd.Flags().GetString("ignore")
			severity, _ := cmd.Flags().GetString("severity")
			format, _ := cmd.Flags().GetString("format")

			// Parse rules and ignore
			if rulesStr != "" && ignoreStr != "" {
				return exitcode.InvalidArgsError("--rules and --ignore are mutually exclusive")
			}

			var rules []string
			if rulesStr != "" {
				rules = strings.Split(rulesStr, ",")
				for i := range rules {
					rules[i] = strings.TrimSpace(rules[i])
				}
			}

			var ignore []string
			if ignoreStr != "" {
				ignore = strings.Split(ignoreStr, ",")
				for i := range ignore {
					ignore[i] = strings.TrimSpace(ignore[i])
				}
			}

			// Validate severity
			if severity != "error" && severity != "warning" && severity != "info" {
				return exitcode.InvalidArgsErrorf("invalid severity level: %s", severity)
			}

			// Validate format
			if format != "text" && format != "json" && format != "yaml" {
				return exitcode.InvalidArgsErrorf("invalid format: %s", format)
			}

			// Validate rule names
			if err := lint.ValidateRuleNames(rules); err != nil {
				return exitcode.InvalidArgsError(err.Error())
			}
			if err := lint.ValidateRuleNames(ignore); err != nil {
				return exitcode.InvalidArgsError(err.Error())
			}

			return runLint(lint.Options{
				SpecRoot: specRoot,
				Rules:    rules,
				Ignore:   ignore,
				Severity: severity,
			}, format)
		},
	}

	cmd.Flags().String("rules", "", "enable only specified rules (comma-separated)")
	cmd.Flags().String("ignore", "", "disable specified rules (comma-separated)")
	cmd.Flags().String("severity", "error", "minimum severity: error, warning, info")
	cmd.Flags().String("format", "text", "output format: text, json, yaml")

	return cmd
}

func runLint(opts lint.Options, format string) error {
	violations, err := lint.Lint(opts)
	if err != nil {
		return exitcode.UnexpectedErrorf("linting error: %v", err)
	}

	// Output results
	if err := outputViolations(violations, format); err != nil {
		return exitcode.UnexpectedErrorf("output error: %v", err)
	}

	// Determine exit code
	if len(violations) > 0 {
		return exitcode.ConflictErrorf("%d violation(s) found", len(violations))
	}
	return nil
}

func outputViolations(violations []lint.Violation, format string) error {
	switch format {
	case "json":
		return outputJSON(violations)
	case "yaml":
		return outputYAML(violations)
	default:
		return outputText(violations)
	}
}

func outputJSON(violations []lint.Violation) error {
	if violations == nil {
		violations = []lint.Violation{}
	}
	data, err := json.MarshalIndent(violations, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputYAML(violations []lint.Violation) error {
	data, err := yaml.Marshal(violations)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		fmt.Println("[]")
	} else {
		fmt.Print(string(data))
	}
	return nil
}

func outputText(violations []lint.Violation) error {
	// Sort violations by file then line
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return violations[i].Line < violations[j].Line
	})

	for _, v := range violations {
		fmt.Printf("%s:%d [%s] %s: %s\n", v.File, v.Line, v.Severity, v.Rule, v.Message)
	}

	if len(violations) > 0 {
		// Count by severity
		errorCount := 0
		warningCount := 0
		infoCount := 0
		for _, v := range violations {
			switch v.Severity {
			case "error":
				errorCount++
			case "warning":
				warningCount++
			case "info":
				infoCount++
			}
		}

		fmt.Printf("\n%d violations found", len(violations))
		var parts []string
		if errorCount > 0 {
			parts = append(parts, fmt.Sprintf("%d error%s", errorCount, plural(errorCount)))
		}
		if warningCount > 0 {
			parts = append(parts, fmt.Sprintf("%d warning%s", warningCount, plural(warningCount)))
		}
		if infoCount > 0 {
			parts = append(parts, fmt.Sprintf("%d info", infoCount))
		}
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
	} else {
		fmt.Println("0 violations found")
	}

	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
