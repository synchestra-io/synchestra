package testscenario

// Features implemented: testing-framework/test-runner

import "testing"

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

func TestParseScenario_rejectsBareCodeFence(t *testing.T) {
	input := "# Scenario: T\n\n## bare-step\n\n```\necho hello\n```"
	_, err := ParseScenario([]byte(input))
	if err == nil {
		t.Fatal("expected error for code block without language annotation")
	}
}
