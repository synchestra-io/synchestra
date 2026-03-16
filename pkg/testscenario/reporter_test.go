package testscenario

// Features implemented: testing-framework/test-runner

import (
	"strings"
	"testing"
	"time"
)

func TestFormatResult_passing(t *testing.T) {
	r := ScenarioResult{
		ScenarioTitle: "My Test",
		Passed:        true,
		StepResults: []StepResult{
			{StepName: "step-a", Passed: true, Duration: 200 * time.Millisecond},
		},
	}
	out := FormatResult(r)
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "My Test") || !strings.Contains(out, "(0.2s)") {
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
