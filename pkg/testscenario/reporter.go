package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"strings"
)

// FormatResult formats a ScenarioResult as human-readable text.
func FormatResult(r ScenarioResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Scenario: %s ===\n", r.ScenarioTitle)
	if r.SetupError != "" {
		fmt.Fprintf(&b, "  ✗ Setup: %s\n", r.SetupError)
	}
	passed, total := 0, len(r.StepResults)
	for _, sr := range r.StepResults {
		if sr.Passed {
			fmt.Fprintf(&b, "  ✓ %s\n", sr.StepName)
			passed++
		} else {
			fmt.Fprintf(&b, "  ✗ %s: %s\n", sr.StepName, sr.Error)
		}
		for _, ac := range sr.ACResults {
			if ac.Passed {
				fmt.Fprintf(&b, "    ✓ AC %s/%s\n", ac.FeaturePath, ac.ACSlug)
			} else {
				fmt.Fprintf(&b, "    ✗ AC %s/%s: %s\n", ac.FeaturePath, ac.ACSlug, ac.Error)
			}
		}
	}
	if r.TeardownError != "" {
		fmt.Fprintf(&b, "  ✗ Teardown: %s\n", r.TeardownError)
	}
	if r.Passed {
		fmt.Fprintf(&b, "\nPASS (%d/%d steps passed)\n", passed, total)
	} else {
		fmt.Fprintf(&b, "\nFAIL (%d/%d steps passed)\n", passed, total)
	}
	return b.String()
}
