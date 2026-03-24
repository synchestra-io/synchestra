package spec

// Features implemented: cli/spec/lint

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"gopkg.in/yaml.v3"
)

// Violation represents a single linting violation.
type Violation struct {
	File     string `json:"file" yaml:"file"`
	Line     int    `json:"line" yaml:"line"`
	Severity string `json:"severity" yaml:"severity"`
	Rule     string `json:"rule" yaml:"rule"`
	Message  string `json:"message" yaml:"message"`
}

// LintOptions holds command options.
type LintOptions struct {
	SpecRoot string
	Rules    []string // enabled rules; nil = all rules
	Ignore   []string // disabled rules
	Severity string   // minimum severity: error, warning, info
	Format   string   // output format: text (default), json, yaml
}

func lintCommand() *cobra.Command {
	var opts LintOptions

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
			opts.Severity, _ = cmd.Flags().GetString("severity")
			opts.Format, _ = cmd.Flags().GetString("format")
			opts.SpecRoot = specRoot

			// Parse rules and ignore
			if rulesStr != "" && ignoreStr != "" {
				return exitcode.InvalidArgsError("--rules and --ignore are mutually exclusive")
			}

			if rulesStr != "" {
				opts.Rules = strings.Split(rulesStr, ",")
				for i := range opts.Rules {
					opts.Rules[i] = strings.TrimSpace(opts.Rules[i])
				}
			}

			if ignoreStr != "" {
				opts.Ignore = strings.Split(ignoreStr, ",")
				for i := range opts.Ignore {
					opts.Ignore[i] = strings.TrimSpace(opts.Ignore[i])
				}
			}

			// Validate severity
			if opts.Severity != "error" && opts.Severity != "warning" && opts.Severity != "info" {
				return exitcode.InvalidArgsErrorf("invalid severity level: %s", opts.Severity)
			}

			// Validate format
			if opts.Format != "text" && opts.Format != "json" && opts.Format != "yaml" {
				return exitcode.InvalidArgsErrorf("invalid format: %s", opts.Format)
			}

			// Validate rule names
			if err := validateRuleNames(opts.Rules); err != nil {
				return exitcode.InvalidArgsError(err.Error())
			}
			if err := validateRuleNames(opts.Ignore); err != nil {
				return exitcode.InvalidArgsError(err.Error())
			}

			return runLint(opts)
		},
	}

	cmd.Flags().String("rules", "", "enable only specified rules (comma-separated)")
	cmd.Flags().String("ignore", "", "disable specified rules (comma-separated)")
	cmd.Flags().String("severity", "error", "minimum severity: error, warning, info")
	cmd.Flags().String("format", "text", "output format: text, json, yaml")

	return cmd
}

func runLint(opts LintOptions) error {
	// Check spec root exists
	info, err := os.Stat(opts.SpecRoot)
	if err != nil {
		return exitcode.UnexpectedErrorf("spec root not found: %s", opts.SpecRoot)
	}
	if !info.IsDir() {
		return exitcode.UnexpectedErrorf("spec root is not a directory: %s", opts.SpecRoot)
	}

	// Create linter and run checks
	linter := newLinter(opts)
	violations, err := linter.lint()
	if err != nil {
		return exitcode.UnexpectedErrorf("linting error: %v", err)
	}

	// Filter violations by severity
	filtered := filterBySeverity(violations, opts.Severity)

	// Output results
	if err := outputViolations(filtered, opts.Format); err != nil {
		return exitcode.UnexpectedErrorf("output error: %v", err)
	}

	// Determine exit code
	if len(filtered) > 0 {
		return exitcode.ConflictErrorf("%d violation(s) found", len(filtered))
	}
	return nil
}

func filterBySeverity(violations []Violation, minSeverity string) []Violation {
	severityOrder := map[string]int{"error": 0, "warning": 1, "info": 2}
	minLevel := severityOrder[minSeverity]

	var filtered []Violation
	for _, v := range violations {
		if severityOrder[v.Severity] <= minLevel {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func outputViolations(violations []Violation, format string) error {
	switch format {
	case "json":
		return outputJSON(violations)
	case "yaml":
		return outputYAML(violations)
	default:
		return outputText(violations)
	}
}

func outputJSON(violations []Violation) error {
	if violations == nil {
		violations = []Violation{}
	}
	data, err := json.MarshalIndent(violations, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputYAML(violations []Violation) error {
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

func outputText(violations []Violation) error {
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

// allRuleNames is the canonical list of known rule names.
var allRuleNames = map[string]bool{
	"readme-exists":      true,
	"oq-section":         true,
	"oq-not-empty":       true,
	"index-entries":      true,
	"heading-levels":     true,
	"feature-ref-syntax": true,
	"internal-links":     true,
	"forward-refs":       true,
	"code-annotations":   true,
	"plan-hierarchy":     true,
}

func validateRuleNames(names []string) error {
	for _, name := range names {
		if !allRuleNames[name] {
			return fmt.Errorf("unknown rule %q", name)
		}
	}
	return nil
}
