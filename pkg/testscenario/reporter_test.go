package testscenario

// Features implemented: testing-framework/test-runner

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func TestFormatResult_passing(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "My Test",
		Passed:        true,
		StepResults: []StepResult{
			{StepName: "step-a", Passed: true, Duration: 200 * time.Millisecond},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "My Test") || !strings.Contains(out, "0.2s") {
		t.Errorf("output = %q", out)
	}
}

func TestFormatResult_failing(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "My Test",
		Passed:        false,
		StepResults: []StepResult{
			{StepName: "bad-step", Passed: false, Error: "exit code 1", Duration: 100 * time.Millisecond},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "FAIL") || !strings.Contains(out, "bad-step") {
		t.Errorf("output = %q", out)
	}
}

func TestFormatResult_withACResults(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "AC Test",
		Passed:        false,
		StepResults: []StepResult{
			{
				StepName: "remove",
				Passed:   false,
				Duration: 50 * time.Millisecond,
				ACResults: []ACResult{
					{FeaturePath: "cli/project/remove", ACSlug: "not-in-list", Passed: true},
					{FeaturePath: "cli/project/remove", ACSlug: "recreate", Passed: false, Error: "assertion failed"},
				},
			},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "not-in-list") || !strings.Contains(out, "recreate") {
		t.Errorf("output = %q", out)
	}
}

func TestLiveReporter_stepBeforeACs(t *testing.T) {
	var buf strings.Builder
	lr := NewLiveReporter(&buf)

	lr.StepStarted("parse-valid")
	lr.ACStarted("testing-framework/test-runner", "parses-valid-scenario")
	lr.ACFinished(ACResult{FeaturePath: "testing-framework/test-runner", ACSlug: "parses-valid-scenario", Passed: true})
	lr.StepFinished(StepResult{StepName: "parse-valid", Passed: true, Duration: 500 * time.Millisecond})

	// Strip ANSI escape codes and cursor movement for line content analysis.
	raw := buf.String()
	clean := ansiRe.ReplaceAllString(raw, "")

	// Extract visible lines (skip empty and cursor-control-only lines).
	var lines []string
	for _, line := range strings.Split(clean, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	// Find the step line and AC line positions.
	stepIdx, acIdx := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "parse-valid") && strings.Contains(line, "0.5s") {
			stepIdx = i
		}
		if strings.Contains(line, "parses-valid-scenario") && !strings.Contains(line, "0.5s") {
			acIdx = i
		}
	}

	if stepIdx == -1 {
		t.Fatalf("step result line not found in output:\n%s", clean)
	}
	if acIdx == -1 {
		t.Fatalf("AC result line not found in output:\n%s", clean)
	}
	if stepIdx >= acIdx {
		t.Errorf("step line (index %d) should appear before AC line (index %d) — parent before child\nlines: %v", stepIdx, acIdx, lines)
	}
}
