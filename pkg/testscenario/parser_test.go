package testscenario

// Features implemented: testing-framework/test-runner

import (
	"os"
	"strings"
	"testing"
)

func TestParseScenario_header(t *testing.T) {
	input := `# Scenario: My test

**Description:** A test scenario.
**Tags:** e2e, cli

## setup-step

` + "```bash\necho hello\n```"

	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Title != "My test" {
		t.Errorf("title = %q, want %q", s.Title, "My test")
	}
	if s.Description != "A test scenario." {
		t.Errorf("description = %q, want %q", s.Description, "A test scenario.")
	}
	if len(s.Tags) != 2 || s.Tags[0] != "e2e" || s.Tags[1] != "cli" {
		t.Errorf("tags = %v, want [e2e cli]", s.Tags)
	}
}

func TestParseScenario_setupTeardown(t *testing.T) {
	input := "# Scenario: T\n\n## Setup\n\n```bash\nexport X=1\n```\n\n## do-thing\n\n```bash\necho ok\n```\n\n## Teardown\n\n```bash\nrm -rf /tmp/test\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Setup != "export X=1" {
		t.Errorf("setup = %q, want %q", s.Setup, "export X=1")
	}
	if s.Teardown != "rm -rf /tmp/test" {
		t.Errorf("teardown = %q, want %q", s.Teardown, "rm -rf /tmp/test")
	}
}

func TestParseScenario_stepWithOutputsAndACs(t *testing.T) {
	input := "# Scenario: T\n\n## create-project\n\n**Outputs:**\n\n| Name | Store | Extract |\n|---|---|---|\n| project_id | context | `echo test` |\n\n**ACs:**\n\n| Feature | ACs |\n|---|---|\n| [cli/project/new](spec/features/cli/project/new/) | * |\n\n```bash\nsynchestra project new\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(s.Steps))
	}
	step := s.Steps[0]
	if step.Name != "create-project" {
		t.Errorf("name = %q, want %q", step.Name, "create-project")
	}
	if len(step.Outputs) != 1 || step.Outputs[0].Name != "project_id" || step.Outputs[0].Store != StoreContext {
		t.Errorf("outputs = %+v, want [{project_id context echo test}]", step.Outputs)
	}
	if len(step.ACs) != 1 || step.ACs[0].FeaturePath != "cli/project/new" || step.ACs[0].ACs != "*" {
		t.Errorf("acs = %+v", step.ACs)
	}
	if step.Code != "synchestra project new" {
		t.Errorf("code = %q", step.Code)
	}
	if step.Language != "bash" {
		t.Errorf("language = %q, want %q", step.Language, "bash")
	}
}

func TestParseScenario_parallelStep(t *testing.T) {
	input := "# Scenario: T\n\n## step-a\n**Parallel:** true\n\n```bash\necho a\n```\n\n## step-b\n**Parallel:** true\n\n```bash\necho b\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Steps[0].Parallel || !s.Steps[1].Parallel {
		t.Errorf("parallel flags: step-a=%v, step-b=%v", s.Steps[0].Parallel, s.Steps[1].Parallel)
	}
}

func TestParseScenario_includeStep(t *testing.T) {
	input := "# Scenario: T\n\n## start-container\n\n**Include:** [flows/start.md](flows/start.md)\n"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Steps[0].Include != "flows/start.md" {
		t.Errorf("include = %q, want %q", s.Steps[0].Include, "flows/start.md")
	}
}

func TestParseScenario_dependsOn(t *testing.T) {
	input := "# Scenario: T\n\n## step-a\n\n```bash\necho a\n```\n\n## step-b\n**Depends on:** step-a\n\n```bash\necho b\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Steps[1].DependsOn) != 1 || s.Steps[1].DependsOn[0] != "step-a" {
		t.Errorf("depends_on = %v, want [step-a]", s.Steps[1].DependsOn)
	}
}

func TestParseScenario_duplicateStepNames(t *testing.T) {
	input := "# Scenario: T\n\n## same-name\n\n```bash\necho 1\n```\n\n## same-name\n\n```bash\necho 2\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for duplicate step names")
	}
}

func TestParseScenario_stepWithNeitherCodeNorInclude(t *testing.T) {
	input := "# Scenario: T\n\n## empty-step\n\n**Depends on:** (none)\n"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for step with neither code nor include")
	}
}

func TestParseScenario_stepWithBothCodeAndInclude(t *testing.T) {
	input := "# Scenario: T\n\n## bad-step\n\n**Include:** [flows/x.md](flows/x.md)\n\n```bash\necho oops\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for step with both code and include")
	}
}

func TestParseScenario_languageAnnotation(t *testing.T) {
	input := "# Scenario: T\n\n## bash-step\n\n```bash\necho hello\n```\n\n## python-step\n\n```python\nprint('hello')\n```\n\n## starlark-step\n\n```starlark\nresult = True\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Steps[0].Language != "bash" {
		t.Errorf("step 0 language = %q, want %q", s.Steps[0].Language, "bash")
	}
	if s.Steps[1].Language != "python" {
		t.Errorf("step 1 language = %q, want %q", s.Steps[1].Language, "python")
	}
	if s.Steps[2].Language != "starlark" {
		t.Errorf("step 2 language = %q, want %q", s.Steps[2].Language, "starlark")
	}
}

func TestParseScenario_nestedCodeFences(t *testing.T) {
	// A step whose bash code block contains triple-backtick heredocs.
	// The outer fence uses 4+ backticks so inner ``` don't close it.
	input := "# Scenario: T\n\n## Setup\n\n````bash\ncat > /tmp/x.md << 'EOF'\n```bash\necho nested\n```\nEOF\n````\n\n## do-thing\n\n```bash\necho ok\n```"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "cat > /tmp/x.md << 'EOF'\n```bash\necho nested\n```\nEOF"
	if s.Setup != want {
		t.Errorf("setup = %q, want %q", s.Setup, want)
	}
}

func TestParseScenario_nestedCodeFencesInStep(t *testing.T) {
	input := "# Scenario: T\n\n## create-fixture\n\n````bash\ncat > /tmp/x.md << 'EOF'\n```bash\necho nested\n```\nEOF\n````"
	s, err := ParseScenario([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(s.Steps))
	}
	want := "cat > /tmp/x.md << 'EOF'\n```bash\necho nested\n```\nEOF"
	if s.Steps[0].Code != want {
		t.Errorf("code = %q, want %q", s.Steps[0].Code, want)
	}
}

func TestParseScenario_runnerCoreDogfood(t *testing.T) {
	data, err := os.ReadFile("../../spec/features/testing-framework/test-runner/_tests/runner-core.md")
	if err != nil {
		t.Fatalf("reading runner-core.md: %v", err)
	}
	s, err := ParseScenario(data)
	if err != nil {
		t.Fatalf("parsing runner-core.md: %v", err)
	}
	if s.Title != "Runner core behaviors" {
		t.Errorf("title = %q, want %q", s.Title, "Runner core behaviors")
	}
	if s.Setup == "" {
		t.Error("expected non-empty Setup")
	}
	if s.SetupLanguage != "bash" {
		t.Errorf("setup language = %q, want %q", s.SetupLanguage, "bash")
	}
	if s.Teardown == "" {
		t.Error("expected non-empty Teardown")
	}

	expectedSteps := []string{
		"build-binary",
		"parse-valid",
		"reject-malformed",
		"test-sequential",
		"test-context-outputs",
		"test-ac-wildcard",
		"test-teardown-on-failure",
		"test-exit-codes",
	}
	if len(s.Steps) != len(expectedSteps) {
		t.Fatalf("steps = %d, want %d", len(s.Steps), len(expectedSteps))
	}
	for i, name := range expectedSteps {
		if s.Steps[i].Name != name {
			t.Errorf("step[%d].Name = %q, want %q", i, s.Steps[i].Name, name)
		}
		if s.Steps[i].Code == "" {
			t.Errorf("step[%d] %q has empty code", i, name)
		}
		if s.Steps[i].Language != "bash" {
			t.Errorf("step[%d] %q language = %q, want %q", i, name, s.Steps[i].Language, "bash")
		}
	}

	// Verify nested code fences in Setup are preserved (heredoc content with ``` inside).
	if !strings.Contains(s.Setup, "```bash") {
		t.Error("setup code should contain nested ```bash from heredocs")
	}
}

func TestParseScenario_rejectsBareCodeFence(t *testing.T) {
	input := "# Scenario: T\n\n## bare-step\n\n```\necho hello\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for code block without language annotation")
	}
}
