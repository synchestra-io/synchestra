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

func TestRunner_stepDuration(t *testing.T) {
	s := &Scenario{
		Title: "duration",
		Steps: []Step{{Name: "sleep", Code: "sleep 0.05", Language: "bash"}},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if result.StepResults[0].Duration == 0 {
		t.Error("step duration should be > 0")
	}
	if result.Duration == 0 {
		t.Error("scenario duration should be > 0")
	}
}

func TestRunner_includeExecution(t *testing.T) {
	// Create a sub-scenario file that the include step references.
	dir := t.TempDir()
	subScenario := filepath.Join(dir, "sub.md")
	err := os.WriteFile(subScenario, []byte(`# Scenario: Sub

## sub-step

`+"```bash"+`
echo included
`+"```"+`
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	s := &Scenario{
		Title: "include-test",
		Steps: []Step{
			{Name: "delegated", Include: subScenario},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: dir})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if len(result.StepResults) != 1 {
		t.Fatalf("expected 1 step result from include, got %d", len(result.StepResults))
	}
	if result.StepResults[0].StepName != "delegated.sub-step" {
		t.Errorf("step name = %q, want %q", result.StepResults[0].StepName, "delegated.sub-step")
	}
	if result.StepResults[0].Stdout != "included" {
		t.Errorf("stdout = %q, want %q", result.StepResults[0].Stdout, "included")
	}
}

func TestRunner_dependsOnInParallelGroup(t *testing.T) {
	// step-b depends on step-a. Both are parallel. step-b should wait for step-a.
	marker := filepath.Join(t.TempDir(), "order")
	s := &Scenario{
		Title: "depends",
		Steps: []Step{
			{
				Name:     "step-a",
				Code:     "sleep 0.05 && echo A >> " + marker,
				Language: "bash",
				Parallel: true,
			},
			{
				Name:      "step-b",
				Code:      "echo B >> " + marker,
				Language:  "bash",
				Parallel:  true,
				DependsOn: []string{"step-a"},
			},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	// A should appear before B because step-b depends on step-a.
	content := string(data)
	if content != "A\nB\n" {
		t.Errorf("execution order = %q, want A before B", content)
	}
}

func TestRunner_acInputValidation(t *testing.T) {
	// Create an AC file with a required input that won't be available.
	specRoot := t.TempDir()
	acDir := filepath.Join(specRoot, "features", "test", "feature", "_acs")
	if err := os.MkdirAll(acDir, 0755); err != nil {
		t.Fatal(err)
	}
	acContent := `# AC: needs-input

**Status:** implemented
**Feature:** [test/feature](../README.md)

## Inputs

| Name | Required | Description |
|---|---|---|
| MISSING_VAR | Yes | A variable that won't be set |

## Verification

` + "```bash" + `
echo "$MISSING_VAR"
` + "```" + `
`
	if err := os.WriteFile(filepath.Join(acDir, "needs-input.md"), []byte(acContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Scenario{
		Title: "ac-input-check",
		Steps: []Step{
			{
				Name:     "verify",
				Code:     "echo ok",
				Language: "bash",
				ACs:      []ACRef{{FeaturePath: "test/feature", ACs: "needs-input"}},
			},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: specRoot})
	result := r.Run(s)
	if result.Passed {
		t.Error("expected scenario to fail due to missing AC input")
	}
	if len(result.StepResults) == 0 || len(result.StepResults[0].ACResults) == 0 {
		t.Fatal("expected AC results")
	}
	acr := result.StepResults[0].ACResults[0]
	if acr.Passed {
		t.Error("expected AC to fail due to missing input")
	}
	if acr.Error == "" {
		t.Error("expected error message about missing input")
	}
}

func TestRunner_setupPropagatesVarsToContext(t *testing.T) {
	s := &Scenario{
		Title:         "setup-vars",
		Setup:         `echo "MY_VAR=hello_world"`,
		SetupLanguage: "bash",
		Steps: []Step{
			{
				Name:     "use-var",
				Code:     "echo ${{ context.MY_VAR }}",
				Language: "bash",
			},
		},
	}
	r := NewRunner(RunnerConfig{SpecRoot: t.TempDir()})
	result := r.Run(s)
	if !result.Passed {
		t.Errorf("scenario failed: %+v", result)
	}
	if result.StepResults[0].Stdout != "hello_world" {
		t.Errorf("stdout = %q, want %q", result.StepResults[0].Stdout, "hello_world")
	}
}
