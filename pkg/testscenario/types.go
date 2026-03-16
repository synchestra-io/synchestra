package testscenario

// Features implemented: testing-framework/test-runner

import "time"

// OutputStore indicates where a step output is stored.
type OutputStore string

const (
	StoreContext OutputStore = "context"
	StoreStep    OutputStore = "step"
	StoreBoth    OutputStore = "both"
)

// Output defines a named value extracted from step execution.
type Output struct {
	Name    string
	Store   OutputStore
	Extract string // shell expression to extract value
}

// ACRef references acceptance criteria to verify after a step.
type ACRef struct {
	FeaturePath string // e.g., "cli/project/new"
	FeatureLink string // markdown link target
	ACs         string // "*" or comma-separated AC slugs
}

// Step is a named step in a test scenario.
type Step struct {
	Name      string
	DependsOn []string
	Parallel  bool
	Outputs   []Output
	ACs       []ACRef
	Include   string // path to sub-flow .md file, empty if inline code
	Code      string // code block content, empty if include
	Language  string // code block language annotation: "bash", "python", or "starlark"
}

// Scenario is a parsed test scenario.
type Scenario struct {
	Title            string
	Description      string
	Tags             []string
	Setup            string // code for setup block
	SetupLanguage    string // language annotation for setup block
	Teardown         string // code for teardown block
	TeardownLanguage string // language annotation for teardown block
	Steps            []Step
}

// ACFile is a parsed acceptance criteria file.
type ACFile struct {
	Slug         string
	Status       string
	FeaturePath  string
	Description  string
	Inputs       []ACInput
	Verification string // verification script content
	Language     string // verification script language: "bash", "python", or "starlark"
}

// ACInput is a named input for an AC verification script.
type ACInput struct {
	Name        string
	Required    bool
	Description string
}

// StepResult holds the outcome of executing a single step.
type StepResult struct {
	StepName  string
	Passed    bool
	Error     string
	Stdout    string
	Stderr    string
	ExitCode  int
	Duration  time.Duration
	ACResults []ACResult
}

// ACResult holds the outcome of a single AC verification.
type ACResult struct {
	FeaturePath string
	ACSlug      string
	Passed      bool
	Error       string
}

// ScenarioResult holds the outcome of a full scenario run.
type ScenarioResult struct {
	ScenarioTitle string
	Passed        bool
	StepResults   []StepResult
	Duration      time.Duration
	SetupError    string
	TeardownError string
}

// ProgressReporter receives live updates during scenario execution.
type ProgressReporter interface {
	ScenarioStarted(title string)
	SetupStarted()
	SetupFinished(err string)
	StepStarted(stepName string)
	StepFinished(result StepResult)
	ACStarted(featurePath, acSlug string)
	ACFinished(result ACResult)
	TeardownStarted()
	TeardownFinished(err string)
	ScenarioFinished(result ScenarioResult)
}
