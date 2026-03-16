package testscenario

// Features implemented: testing-framework/test-runner

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

// ANSI color palette.
var (
	colorGreen  = lipgloss.Color("#34D399")
	colorRed    = lipgloss.Color("#F87171")
	colorYellow = lipgloss.Color("#FBBF24")
	colorCyan   = lipgloss.Color("#67E8F9")
	colorDim    = lipgloss.Color("#6B7280")
	colorWhite  = lipgloss.Color("#F9FAFB")
)

// Styles.
var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	stylePass = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	styleFail = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	styleStepName = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleDuration = lipgloss.NewStyle().
			Foreground(colorDim)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleACPath = lipgloss.NewStyle().
			Foreground(colorCyan)

	styleSummaryPass = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorGreen)

	styleSummaryFail = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorRed)

	styleWarn = lipgloss.NewStyle().
			Foreground(colorYellow)

	styleDivider = lipgloss.NewStyle().
			Foreground(colorDim)
)

// FormatResult formats a ScenarioResult as human-readable text with color.
func FormatResult(r ScenarioResult) string {
	var b strings.Builder

	// Header.
	b.WriteString("\n")
	b.WriteString(styleDivider.Render("─────────────────────────────────────────────"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(styleTitle.Render(r.ScenarioTitle))
	dur := fmt.Sprintf("%.1fs", r.Duration.Seconds())
	b.WriteString("  ")
	b.WriteString(styleDuration.Render(dur))
	b.WriteString("\n")
	b.WriteString(styleDivider.Render("─────────────────────────────────────────────"))
	b.WriteString("\n\n")

	// Setup error.
	if r.SetupError != "" {
		b.WriteString("  ")
		b.WriteString(styleFail.Render("✘ Setup"))
		b.WriteString("  ")
		b.WriteString(styleError.Render(r.SetupError))
		b.WriteString("\n\n")
	}

	// Steps.
	passed, failed := 0, 0
	acPassed, acFailed := 0, 0

	for _, sr := range r.StepResults {
		b.WriteString("  ")
		if sr.Passed {
			b.WriteString(stylePass.Render("✔"))
			passed++
		} else {
			b.WriteString(styleFail.Render("✘"))
			failed++
		}
		b.WriteString("  ")
		b.WriteString(styleStepName.Render(sr.StepName))

		dur := fmt.Sprintf("%.1fs", sr.Duration.Seconds())
		b.WriteString("  ")
		b.WriteString(styleDuration.Render(dur))

		if !sr.Passed && sr.Error != "" {
			b.WriteString("\n    ")
			b.WriteString(styleError.Render(sr.Error))
		}
		b.WriteString("\n")

		// AC results.
		for _, ac := range sr.ACResults {
			acSlug := ac.FeaturePath + "/" + ac.ACSlug
			b.WriteString("      ")
			if ac.Passed {
				b.WriteString(stylePass.Render("✔"))
				b.WriteString("  ")
				b.WriteString(styleACPath.Render(acSlug))
				acPassed++
			} else {
				b.WriteString(styleFail.Render("✘"))
				b.WriteString("  ")
				b.WriteString(styleACPath.Render(acSlug))
				acFailed++
				if ac.Error != "" {
					b.WriteString("\n        ")
					b.WriteString(styleError.Render(ac.Error))
				}
			}
			b.WriteString("\n")
		}
	}

	// Teardown warning.
	if r.TeardownError != "" {
		b.WriteString("\n  ")
		b.WriteString(styleWarn.Render("⚠ Teardown: " + r.TeardownError))
		b.WriteString("\n")
	}

	// Summary.
	b.WriteString("\n")
	b.WriteString(styleDivider.Render("─────────────────────────────────────────────"))
	b.WriteString("\n")

	total := passed + failed
	if r.Passed {
		b.WriteString("  ")
		b.WriteString(styleSummaryPass.Render(fmt.Sprintf("PASS  %d/%d steps", passed, total)))
	} else {
		b.WriteString("  ")
		b.WriteString(styleSummaryFail.Render(fmt.Sprintf("FAIL  %d/%d steps passed, %d failed", passed, total, failed)))
	}

	acTotal := acPassed + acFailed
	if acTotal > 0 {
		b.WriteString(styleDuration.Render(fmt.Sprintf("  ·  %d/%d ACs", acPassed, acTotal)))
	}

	b.WriteString("\n")
	b.WriteString(styleDivider.Render("─────────────────────────────────────────────"))
	b.WriteString("\n\n")

	return b.String()
}

// jsonResult is the JSON-serializable representation of a scenario result.
type jsonResult struct {
	Scenario jsonScenario `json:"scenario"`
	Result   string       `json:"result"`
	Steps    []jsonStep   `json:"steps"`
	Duration float64      `json:"duration_seconds"`
}

type jsonScenario struct {
	Name string `json:"name"`
}

type jsonStep struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
	ExitCode int      `json:"exit_code"`
	Duration float64  `json:"duration_seconds"`
	ACs      []jsonAC `json:"acs,omitempty"`
}

type jsonAC struct {
	Feature string `json:"feature"`
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// FormatResultJSON formats a ScenarioResult as JSON.
func FormatResultJSON(r ScenarioResult) string {
	result := "passed"
	if !r.Passed {
		result = "failed"
	}
	jr := jsonResult{
		Scenario: jsonScenario{Name: r.ScenarioTitle},
		Result:   result,
		Duration: r.Duration.Seconds(),
	}
	for _, sr := range r.StepResults {
		status := "passed"
		if !sr.Passed {
			status = "failed"
		}
		js := jsonStep{
			Name:     sr.StepName,
			Status:   status,
			Stdout:   sr.Stdout,
			Stderr:   sr.Stderr,
			ExitCode: sr.ExitCode,
			Duration: sr.Duration.Seconds(),
		}
		for _, ac := range sr.ACResults {
			acStatus := "passed"
			if !ac.Passed {
				acStatus = "failed"
			}
			js.ACs = append(js.ACs, jsonAC{
				Feature: ac.FeaturePath,
				Slug:    ac.ACSlug,
				Status:  acStatus,
				Error:   ac.Error,
			})
		}
		jr.Steps = append(jr.Steps, js)
	}
	data, _ := json.MarshalIndent(jr, "", "  ")
	return string(data) + "\n"
}

// LiveReporter prints real-time progress to a writer during scenario execution.
type LiveReporter struct {
	w      io.Writer
	hasACs bool // true if any AC lines were printed for the current step
}

// NewLiveReporter creates a LiveReporter that writes to w.
func NewLiveReporter(w io.Writer) *LiveReporter {
	return &LiveReporter{w: w}
}

func (lr *LiveReporter) ScenarioStarted(title string) {
	_, _ = fmt.Fprintf(lr.w, "\n%s\n", styleDivider.Render("─────────────────────────────────────────────"))
	_, _ = fmt.Fprintf(lr.w, "  %s\n", styleTitle.Render(title))
	_, _ = fmt.Fprintf(lr.w, "%s\n\n", styleDivider.Render("─────────────────────────────────────────────"))
}

func (lr *LiveReporter) SetupStarted() {
	_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", styleDuration.Render("▸"), styleDuration.Render("Setup"))
}

func (lr *LiveReporter) SetupFinished(err string) {
	// Move cursor up and overwrite the "in progress" line.
	_, _ = fmt.Fprintf(lr.w, "\033[1A\033[2K")
	if err == "" {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", stylePass.Render("✔"), styleStepName.Render("Setup"))
	} else {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s  %s\n", styleFail.Render("✘"), styleStepName.Render("Setup"), styleError.Render(err))
	}
}

func (lr *LiveReporter) StepStarted(stepName string) {
	lr.hasACs = false
	_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", styleDuration.Render("▸"), styleDuration.Render(stepName))
}

func (lr *LiveReporter) StepFinished(result StepResult) {
	if !lr.hasACs {
		// No AC lines printed — overwrite the "in progress" line.
		_, _ = fmt.Fprintf(lr.w, "\033[1A\033[2K")
	}
	dur := fmt.Sprintf("%.1fs", result.Duration.Seconds())
	if result.Passed {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s  %s\n", stylePass.Render("✔"), styleStepName.Render(result.StepName), styleDuration.Render(dur))
	} else {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s  %s\n", styleFail.Render("✘"), styleStepName.Render(result.StepName), styleDuration.Render(dur))
		if result.Error != "" {
			_, _ = fmt.Fprintf(lr.w, "    %s\n", styleError.Render(result.Error))
		}
	}
}

func (lr *LiveReporter) ACStarted(featurePath, acSlug string) {
	if !lr.hasACs {
		// First AC for this step — overwrite the step's "in progress" line.
		_, _ = fmt.Fprintf(lr.w, "\033[1A\033[2K")
		lr.hasACs = true
	}
	slug := featurePath + "/" + acSlug
	_, _ = fmt.Fprintf(lr.w, "      %s  %s\n", styleDuration.Render("▸"), styleDuration.Render(slug))
}

func (lr *LiveReporter) ACFinished(result ACResult) {
	// Move cursor up and overwrite the "in progress" line.
	_, _ = fmt.Fprintf(lr.w, "\033[1A\033[2K")
	slug := result.FeaturePath + "/" + result.ACSlug
	if result.Passed {
		_, _ = fmt.Fprintf(lr.w, "      %s  %s\n", stylePass.Render("✔"), styleACPath.Render(slug))
	} else {
		_, _ = fmt.Fprintf(lr.w, "      %s  %s\n", styleFail.Render("✘"), styleACPath.Render(slug))
		if result.Error != "" {
			_, _ = fmt.Fprintf(lr.w, "        %s\n", styleError.Render(result.Error))
		}
	}
}

func (lr *LiveReporter) TeardownStarted() {
	_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", styleDuration.Render("▸"), styleDuration.Render("Teardown"))
}

func (lr *LiveReporter) TeardownFinished(err string) {
	_, _ = fmt.Fprintf(lr.w, "\033[1A\033[2K")
	if err == "" {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", stylePass.Render("✔"), styleStepName.Render("Teardown"))
	} else {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s\n", styleWarn.Render("⚠"), styleStepName.Render("Teardown"))
		_, _ = fmt.Fprintf(lr.w, "    %s\n", styleWarn.Render(err))
	}
}

func (lr *LiveReporter) ScenarioFinished(result ScenarioResult) {
	passed, failed := 0, 0
	acPassed, acFailed := 0, 0
	for _, sr := range result.StepResults {
		if sr.Passed {
			passed++
		} else {
			failed++
		}
		for _, ac := range sr.ACResults {
			if ac.Passed {
				acPassed++
			} else {
				acFailed++
			}
		}
	}

	_, _ = fmt.Fprintf(lr.w, "\n%s\n", styleDivider.Render("─────────────────────────────────────────────"))
	total := passed + failed
	dur := fmt.Sprintf("%.1fs", result.Duration.Seconds())
	if result.Passed {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s", styleSummaryPass.Render(fmt.Sprintf("PASS  %d/%d steps", passed, total)), styleDuration.Render(dur))
	} else {
		_, _ = fmt.Fprintf(lr.w, "  %s  %s", styleSummaryFail.Render(fmt.Sprintf("FAIL  %d/%d steps passed, %d failed", passed, total, failed)), styleDuration.Render(dur))
	}
	acTotal := acPassed + acFailed
	if acTotal > 0 {
		_, _ = fmt.Fprintf(lr.w, "%s", styleDuration.Render(fmt.Sprintf("  ·  %d/%d ACs", acPassed, acTotal)))
	}
	_, _ = fmt.Fprintf(lr.w, "\n%s\n\n", styleDivider.Render("─────────────────────────────────────────────"))
}
