package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// RunnerConfig holds configuration for a Runner.
type RunnerConfig struct {
	SpecRoot string
}

// Runner executes test scenarios.
type Runner struct {
	config     RunnerConfig
	acResolver *ACResolver
}

// NewRunner creates a new Runner with the given config.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		config:     cfg,
		acResolver: NewACResolver(cfg.SpecRoot),
	}
}

// Run executes a full scenario and returns the result.
func (r *Runner) Run(s *Scenario) ScenarioResult {
	result := ScenarioResult{
		ScenarioTitle: s.Title,
		Passed:        true,
	}
	ctx := NewExecContext()

	// Run teardown via defer so it always runs.
	defer func() {
		if s.Teardown != "" {
			_, stderr, exitCode, err := execScript(s.TeardownLanguage, s.Teardown, nil)
			if err != nil || exitCode != 0 {
				msg := stderr
				if err != nil {
					msg = err.Error()
				}
				result.TeardownError = msg
			}
		}
	}()

	// Run setup if present.
	if s.Setup != "" {
		_, stderr, exitCode, err := execScript(s.SetupLanguage, s.Setup, nil)
		if err != nil || exitCode != 0 {
			msg := stderr
			if err != nil {
				msg = err.Error()
			}
			result.SetupError = msg
			result.Passed = false
			return result
		}
	}

	// Group steps into sequential and parallel groups.
	groups := groupSteps(s.Steps)

	for _, group := range groups {
		if len(group) == 1 && !group[0].Parallel {
			// Sequential single step.
			sr := r.runStep(group[0], ctx)
			result.StepResults = append(result.StepResults, sr)
			if !sr.Passed {
				result.Passed = false
			}
		} else {
			// Parallel group.
			stepResults := make([]StepResult, len(group))
			var wg sync.WaitGroup
			for i, step := range group {
				wg.Add(1)
				go func(idx int, st Step) {
					defer wg.Done()
					stepResults[idx] = r.runStep(st, ctx)
				}(i, step)
			}
			wg.Wait()
			for _, sr := range stepResults {
				result.StepResults = append(result.StepResults, sr)
				if !sr.Passed {
					result.Passed = false
				}
			}
		}
	}

	return result
}

// groupSteps groups consecutive parallel steps together.
// Non-parallel steps form single-element groups.
func groupSteps(steps []Step) [][]Step {
	var groups [][]Step
	var currentParallel []Step

	for _, step := range steps {
		if step.Parallel {
			currentParallel = append(currentParallel, step)
		} else {
			if len(currentParallel) > 0 {
				groups = append(groups, currentParallel)
				currentParallel = nil
			}
			groups = append(groups, []Step{step})
		}
	}
	if len(currentParallel) > 0 {
		groups = append(groups, currentParallel)
	}
	return groups
}

// runStep executes a single step and returns the result.
func (r *Runner) runStep(step Step, ctx *ExecContext) StepResult {
	sr := StepResult{
		StepName: step.Name,
		Passed:   true,
	}

	// Resolve ${{ }} references in the code.
	code, err := ctx.ResolveString(step.Code)
	if err != nil {
		sr.Passed = false
		sr.Error = fmt.Sprintf("resolving variables: %v", err)
		return sr
	}

	// Execute the step script.
	stdout, stderr, exitCode, err := execScript(step.Language, code, nil)
	sr.Stdout = stdout
	sr.Stderr = stderr
	sr.ExitCode = exitCode
	if err != nil {
		sr.Passed = false
		sr.Error = err.Error()
		return sr
	}
	if exitCode != 0 {
		sr.Passed = false
		sr.Error = fmt.Sprintf("exit code %d", exitCode)
	}

	// Extract outputs.
	for _, output := range step.Outputs {
		outVal, err := r.extractOutput(output, stdout, stderr, exitCode)
		if err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("extracting output %q: %v", output.Name, err)
			return sr
		}
		if err := ctx.StoreOutput(step.Name, output.Name, outVal, output.Store); err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("storing output %q: %v", output.Name, err)
			return sr
		}
	}

	// Resolve and verify ACs.
	for _, acRef := range step.ACs {
		acs, err := r.acResolver.Resolve(acRef.FeaturePath, acRef.ACs)
		if err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("resolving ACs for %s: %v", acRef.FeaturePath, err)
			return sr
		}
		for _, ac := range acs {
			acResult := r.verifyAC(ac, ctx, stdout, stderr, exitCode)
			sr.ACResults = append(sr.ACResults, acResult)
			if !acResult.Passed {
				sr.Passed = false
			}
		}
	}

	return sr
}

// extractOutput runs the extract expression and returns the captured value.
func (r *Runner) extractOutput(output Output, stdout, stderr string, exitCode int) (string, error) {
	env := []string{
		"STEP_STDOUT=" + stdout,
		"STEP_STDERR=" + stderr,
		fmt.Sprintf("STEP_EXIT_CODE=%d", exitCode),
	}
	// Write stdout to a temp file so `cat $STEP_STDOUT` works as a file path pattern.
	// Actually, STEP_STDOUT is the content. Let's write it to a temp file.
	tmpFile, err := os.CreateTemp("", "step-stdout-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	if _, err := tmpFile.WriteString(stdout); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("closing temp file: %w", err)
	}
	env[0] = "STEP_STDOUT=" + tmpFile.Name()

	tmpErrFile, err := os.CreateTemp("", "step-stderr-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpErrFile.Name()) }()
	if _, err := tmpErrFile.WriteString(stderr); err != nil {
		_ = tmpErrFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpErrFile.Close(); err != nil {
		return "", fmt.Errorf("closing temp file: %w", err)
	}
	env[1] = "STEP_STDERR=" + tmpErrFile.Name()

	val, _, extractExitCode, err := execScript("bash", output.Extract, env)
	if err != nil {
		return "", err
	}
	if extractExitCode != 0 {
		return "", fmt.Errorf("extract command exited with code %d", extractExitCode)
	}
	return val, nil
}

// verifyAC runs an AC verification script and returns the result.
func (r *Runner) verifyAC(ac ACFile, ctx *ExecContext, stdout, stderr string, exitCode int) ACResult {
	acr := ACResult{
		FeaturePath: ac.FeaturePath,
		ACSlug:      ac.Slug,
		Passed:      true,
	}

	if ac.Verification == "" {
		return acr
	}

	env := ctx.ContextVarsAsEnv()
	env = append(env,
		"STEP_STDOUT="+stdout,
		"STEP_STDERR="+stderr,
		fmt.Sprintf("STEP_EXIT_CODE=%d", exitCode),
	)

	_, verifyStderr, verifyExitCode, err := execScript(ac.Language, ac.Verification, env)
	if err != nil {
		acr.Passed = false
		acr.Error = err.Error()
		return acr
	}
	if verifyExitCode != 0 {
		acr.Passed = false
		acr.Error = fmt.Sprintf("AC verification failed (exit %d): %s", verifyExitCode, verifyStderr)
	}
	return acr
}

// execScript runs a script in the specified language and returns stdout, stderr, exit code.
func execScript(language, script string, env []string) (stdout, stderr string, exitCode int, err error) {
	var cmd *exec.Cmd
	switch language {
	case "bash":
		cmd = exec.Command("bash", "-c", script)
	case "python":
		cmd = exec.Command("python3", "-c", script)
	case "starlark":
		return "", "", 1, fmt.Errorf("starlark execution not yet implemented")
	default:
		return "", "", 1, fmt.Errorf("unsupported language: %s", language)
	}
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode = cmd.ProcessState.ExitCode()
	stdout = strings.TrimRight(outBuf.String(), "\n")
	stderr = errBuf.String()
	// Only return the error if the process failed to start (not just non-zero exit).
	if runErr != nil && cmd.ProcessState == nil {
		return stdout, stderr, exitCode, runErr
	}
	return stdout, stderr, exitCode, nil
}
