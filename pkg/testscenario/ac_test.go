package testscenario

// Features implemented: testing-framework/test-runner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeACFile(t *testing.T, dir, slug, content string) {
	t.Helper()
	acsDir := filepath.Join(dir, "_acs")
	if err := os.MkdirAll(acsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(acsDir, slug+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleAC = `# AC: not-in-list

**Status:** implemented
**Feature:** [cli/project/remove](../README.md)

## Description

Deleted project absent from list.

## Inputs

| Name | Required | Description |
|---|---|---|
| project_id | Yes | ID of the deleted project |

## Verification

` + "```bash\n! echo $project_id\n```"

func TestParseACFile(t *testing.T) {
	ac, err := ParseACFile([]byte(sampleAC), "not-in-list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ac.Slug != "not-in-list" {
		t.Errorf("slug = %q", ac.Slug)
	}
	if ac.Status != "implemented" {
		t.Errorf("status = %q", ac.Status)
	}
	if len(ac.Inputs) != 1 || !ac.Inputs[0].Required {
		t.Errorf("inputs = %+v", ac.Inputs)
	}
	if ac.Verification != "! echo $project_id" {
		t.Errorf("verification = %q", ac.Verification)
	}
	if ac.Language != "bash" {
		t.Errorf("language = %q, want %q", ac.Language, "bash")
	}
}

func TestResolveACs_wildcard(t *testing.T) {
	specRoot := t.TempDir()
	featureDir := filepath.Join(specRoot, "features", "cli", "project", "remove")
	writeACFile(t, featureDir, "not-in-list", sampleAC)
	writeACFile(t, featureDir, "recreate", sampleAC)

	resolver := NewACResolver(specRoot)
	acs, err := resolver.Resolve("cli/project/remove", "*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acs) != 2 {
		t.Errorf("got %d ACs, want 2", len(acs))
	}
}

func TestResolveACs_specific(t *testing.T) {
	specRoot := t.TempDir()
	featureDir := filepath.Join(specRoot, "features", "cli", "project", "remove")
	writeACFile(t, featureDir, "not-in-list", sampleAC)
	writeACFile(t, featureDir, "recreate", sampleAC)

	resolver := NewACResolver(specRoot)
	acs, err := resolver.Resolve("cli/project/remove", "not-in-list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(acs) != 1 || acs[0].Slug != "not-in-list" {
		t.Errorf("got %+v", acs)
	}
}

func TestResolveACs_nonExistentFeature(t *testing.T) {
	specRoot := t.TempDir()
	resolver := NewACResolver(specRoot)
	_, err := resolver.Resolve("does/not/exist", "*")
	if err == nil {
		t.Fatal("expected error")
	}
}
