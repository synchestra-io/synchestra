package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// RunnerConfig holds configuration for a Runner.
type RunnerConfig struct {
	SpecRoot string
	Progress ProgressReporter
}

// Runner executes test scenarios.
type Runner struct {
	config          RunnerConfig
	acResolver      *ACResolver
	includeResolver *IncludeResolver
}

// NewRunner creates a new Runner with the given config.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		config:          cfg,
		acResolver:      NewACResolver(cfg.SpecRoot),
		includeResolver: NewIncludeResolver(),
	}
}

// Run executes a full scenario and returns the result.
func (r *Runner) Run(s *Scenario) (result ScenarioResult) {
	scenarioStart := time.Now()
	result = ScenarioResult{
		ScenarioTitle: s.Title,
		Passed:        true,
	}
	ctx := NewExecContext()
	r.notify(func(p ProgressReporter) { p.ScenarioStarted(s.Title) })

	// Run teardown via defer so it always runs.
	defer func() {
		if s.Teardown != "" {
			r.notify(func(p ProgressReporter) { p.TeardownStarted() })
			_, stderr, exitCode, err := execScript(s.TeardownLanguage, s.Teardown, ctx.ContextVarsAsEnv())
			teardownErr := ""
			if err != nil || exitCode != 0 {
				msg := stderr
				if err != nil {
					msg = err.Error()
				}
				result.TeardownError = msg
				teardownErr = msg
			}
			r.notify(func(p ProgressReporter) { p.TeardownFinished(teardownErr) })
		}
		result.Duration = time.Since(scenarioStart)
		r.notify(func(p ProgressReporter) { p.ScenarioFinished(result) })
	}()

	// Run setup if present. Propagate exported variables to context.
	if s.Setup != "" {
		r.notify(func(p ProgressReporter) { p.SetupStarted() })
		stdout, stderr, exitCode, err := execScript(s.SetupLanguage, s.Setup, nil)
		if err != nil || exitCode != 0 {
			msg := stderr
			if err != nil {
				msg = err.Error()
			}
			result.SetupError = msg
			result.Passed = false
			r.notify(func(p ProgressReporter) { p.SetupFinished(msg) })
			return result
		}
		r.notify(func(p ProgressReporter) { p.SetupFinished("") })
		// Parse exported variables from setup stdout so subsequent steps can use them.
		parseExportedVars(stdout, ctx)
	}

	// Resolve includes before execution.
	if err := r.resolveIncludes(s); err != nil {
		result.SetupError = fmt.Sprintf("resolving includes: %v", err)
		result.Passed = false
		return result
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
			// Parallel group with DependsOn enforcement.
			r.runParallelGroup(group, ctx, &result)
		}
	}

	return result
}

// resolveIncludes inlines included sub-scenarios into the parent scenario's step list.
func (r *Runner) resolveIncludes(s *Scenario) error {
	var resolved []Step
	for _, step := range s.Steps {
		if step.Include == "" {
			resolved = append(resolved, step)
			continue
		}
		subScenario, err := r.includeResolver.Resolve(step.Include, nil)
		if err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
		for _, subStep := range subScenario.Steps {
			subStep.Name = step.Name + "." + subStep.Name
			resolved = append(resolved, subStep)
		}
	}
	s.Steps = resolved
	return nil
}

// runParallelGroup executes a parallel group with DependsOn enforcement.
func (r *Runner) runParallelGroup(group []Step, ctx *ExecContext, result *ScenarioResult) {
	stepResults := make([]StepResult, len(group))

	// Build a map of step name -> done channel for dependency coordination.
	done := make(map[string]chan struct{}, len(group))
	for _, step := range group {
		done[step.Name] = make(chan struct{})
	}

	var wg sync.WaitGroup
	for i, step := range group {
		wg.Add(1)
		go func(idx int, st Step) {
			defer wg.Done()
			defer close(done[st.Name])

			// Wait for dependencies within this parallel group.
			for _, dep := range st.DependsOn {
				if ch, ok := done[dep]; ok {
					<-ch
				}
			}

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

// parseExportedVars extracts KEY=VALUE lines from setup output and stores them in context.
// This handles the common pattern where Setup uses `echo KEY=VALUE` to communicate state.
func parseExportedVars(stdout string, ctx *ExecContext) {
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok && k != "" {
			_ = ctx.StoreOutput("setup", k, v, StoreContext)
		}
	}
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

// notify calls fn with the progress reporter if one is configured.
func (r *Runner) notify(fn func(ProgressReporter)) {
	if r.config.Progress != nil {
		fn(r.config.Progress)
	}
}

// runStep executes a single step and returns the result.
func (r *Runner) runStep(step Step, ctx *ExecContext) StepResult {
	stepStart := time.Now()
	r.notify(func(p ProgressReporter) { p.StepStarted(step.Name) })
	sr := StepResult{
		StepName: step.Name,
		Passed:   true,
	}

	// Resolve ${{ }} references in the code.
	code, err := ctx.ResolveString(step.Code)
	if err != nil {
		sr.Passed = false
		sr.Error = fmt.Sprintf("resolving variables: %v", err)
		sr.Duration = time.Since(stepStart)
		return sr
	}

	// Execute the step script.
	stdout, stderr, exitCode, err := execScript(step.Language, code, ctx.ContextVarsAsEnv())
	sr.Stdout = stdout
	sr.Stderr = stderr
	sr.ExitCode = exitCode
	if err != nil {
		sr.Passed = false
		sr.Error = err.Error()
		sr.Duration = time.Since(stepStart)
		return sr
	}
	if exitCode != 0 {
		sr.Passed = false
		sr.Error = fmt.Sprintf("exit code %d", exitCode)
	}

	// Extract outputs.
	for _, output := range step.Outputs {
		outVal, err := r.extractOutput(output, ctx, stdout, stderr, exitCode)
		if err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("extracting output %q: %v", output.Name, err)
			sr.Duration = time.Since(stepStart)
			return sr
		}
		if err := ctx.StoreOutput(step.Name, output.Name, outVal, output.Store); err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("storing output %q: %v", output.Name, err)
			sr.Duration = time.Since(stepStart)
			return sr
		}
	}

	// Resolve and verify ACs.
	for _, acRef := range step.ACs {
		acs, err := r.acResolver.Resolve(acRef.FeaturePath, acRef.ACs)
		if err != nil {
			sr.Passed = false
			sr.Error = fmt.Sprintf("resolving ACs for %s: %v", acRef.FeaturePath, err)
			sr.Duration = time.Since(stepStart)
			return sr
		}
		for _, ac := range acs {
			r.notify(func(p ProgressReporter) { p.ACStarted(ac.FeaturePath, ac.Slug) })
			acResult := r.verifyAC(ac, ctx, stdout, stderr, exitCode)
			sr.ACResults = append(sr.ACResults, acResult)
			r.notify(func(p ProgressReporter) { p.ACFinished(acResult) })
			if !acResult.Passed {
				sr.Passed = false
			}
		}
	}

	sr.Duration = time.Since(stepStart)
	r.notify(func(p ProgressReporter) { p.StepFinished(sr) })
	return sr
}

// extractOutput runs the extract expression and returns the captured value.
// STEP_STDOUT and STEP_STDERR are written to temp files so extract expressions
// can use them as file paths (e.g., `cat $STEP_STDOUT`).
func (r *Runner) extractOutput(output Output, ctx *ExecContext, stdout, stderr string, exitCode int) (string, error) {
	stdoutFile, err := writeTempFile("step-stdout-*", stdout)
	if err != nil {
		return "", fmt.Errorf("creating stdout temp file: %w", err)
	}
	defer func() { _ = os.Remove(stdoutFile) }()

	stderrFile, err := writeTempFile("step-stderr-*", stderr)
	if err != nil {
		return "", fmt.Errorf("creating stderr temp file: %w", err)
	}
	defer func() { _ = os.Remove(stderrFile) }()

	env := ctx.ContextVarsAsEnv()
	env = append(env,
		"STEP_STDOUT="+stdoutFile,
		"STEP_STDERR="+stderrFile,
		fmt.Sprintf("STEP_EXIT_CODE=%d", exitCode),
	)

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
// STEP_STDOUT and STEP_STDERR are written to temp files for consistency with extractOutput.
func (r *Runner) verifyAC(ac ACFile, ctx *ExecContext, stdout, stderr string, exitCode int) ACResult {
	acr := ACResult{
		FeaturePath: ac.FeaturePath,
		ACSlug:      ac.Slug,
		Passed:      true,
	}

	if ac.Verification == "" {
		return acr
	}

	// Validate required AC inputs are available.
	env := ctx.ContextVarsAsEnv()
	for _, input := range ac.Inputs {
		if !input.Required {
			continue
		}
		found := false
		for _, e := range env {
			if k, _, ok := strings.Cut(e, "="); ok && k == input.Name {
				found = true
				break
			}
		}
		if !found {
			// Check OS environment as fallback.
			if _, ok := os.LookupEnv(input.Name); !ok {
				acr.Passed = false
				acr.Error = fmt.Sprintf("missing required AC input %q", input.Name)
				return acr
			}
		}
	}

	// Write stdout/stderr to temp files for consistent semantics with extractOutput.
	stdoutFile, err := writeTempFile("ac-stdout-*", stdout)
	if err != nil {
		acr.Passed = false
		acr.Error = fmt.Sprintf("creating temp file: %v", err)
		return acr
	}
	defer func() { _ = os.Remove(stdoutFile) }()

	stderrFile, err := writeTempFile("ac-stderr-*", stderr)
	if err != nil {
		acr.Passed = false
		acr.Error = fmt.Sprintf("creating temp file: %v", err)
		return acr
	}
	defer func() { _ = os.Remove(stderrFile) }()

	env = append(env,
		"STEP_STDOUT="+stdoutFile,
		"STEP_STDERR="+stderrFile,
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

// writeTempFile writes content to a temporary file and returns its path.
func writeTempFile(pattern, content string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
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
