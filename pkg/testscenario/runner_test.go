package testscenario

// Features implemented: testing-framework/test-runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunner_singleStep(t *testing.T) {
	s := &Scenario{
		Title: "simple",
		Steps: []Step{{Name: "echo-test", Code: "echo hello", Language: "bash"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if len(result.StepResults) != 1 || !result.StepResults[0].Passed {
		t.Errorf("step failed: %+v", result.StepResults)
	}
}

func TestRunner_failingStep(t *testing.T) {
	s := &Scenario{
		Title: "fail",
		Steps: []Step{{Name: "bad", Code: "exit 1", Language: "bash"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if result.Passed {
		t.Error("expected scenario to fail")
	}
	if result.StepResults[0].ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.StepResults[0].ExitCode)
	}
}

func TestRunner_setupAndTeardown(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "teardown-ran")
	s := &Scenario{
		Title:            "lifecycle",
		Setup:            "export MARKER=" + marker,
		SetupLanguage:    "bash",
		Teardown:         "touch " + marker,
		TeardownLanguage: "bash",
		Steps:            []Step{{Name: "noop", Code: "echo ok", Language: "bash"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	_ = r.Run(s)
	if _, err := os.Stat(marker); err != nil {
		t.Error("teardown did not run")
	}
}

func TestRunner_teardownRunsOnFailure(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "teardown-ran")
	s := &Scenario{
		Title:            "fail-teardown",
		Teardown:         "touch " + marker,
		TeardownLanguage: "bash",
		Steps:            []Step{{Name: "fail", Code: "exit 1", Language: "bash"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	_ = r.Run(s)
	if _, err := os.Stat(marker); err != nil {
		t.Error("teardown did not run after failure")
	}
}

func TestRunner_contextOutputPassthrough(t *testing.T) {
	s := &Scenario{
		Title: "context",
		Steps: []Step{
			{
				Name:     "produce",
				Code:     "echo myvalue",
				Language: "bash",
				Outputs:  []Output{{Name: "val", Store: StoreContext, Extract: "cat $STEP_STDOUT"}},
			},
			{
				Name:     "consume",
				Code:     "echo got-${{ context.val }}",
				Language: "bash",
			},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if result.StepResults[1].Stdout != "got-myvalue" {
		t.Errorf("stdout = %q, want %q", result.StepResults[1].Stdout, "got-myvalue")
	}
}
