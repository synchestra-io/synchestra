package feature

// Features implemented: cli/feature/new

import (
	"strings"
	"testing"
)

func Test_generateReadme(t *testing.T) {
	t.Parallel()

	requiredSections := []string{
		"## Summary",
		"## Problem",
		"## Behavior",
		"## Acceptance Criteria",
		"## Outstanding Questions",
	}

	tests := []struct {
		name        string
		title       string
		status      string
		description string
		deps        []string
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "basic_feature_with_description_no_deps",
			title:       "My Feature",
			status:      "draft",
			description: "A cool feature.",
			deps:        nil,
			wantContain: []string{
				"# Feature: My Feature",
				"**Status:** draft",
				"A cool feature.",
			},
			wantAbsent: []string{
				"## Dependencies",
			},
		},
		{
			name:        "feature_with_deps_no_description",
			title:       "Task Board",
			status:      "approved",
			description: "",
			deps:        []string{"state-store", "cli"},
			wantContain: []string{
				"## Dependencies",
				"- state-store",
				"- cli",
				"TODO: Brief summary of the feature.",
			},
		},
		{
			name:        "feature_with_description_and_deps",
			title:       "Test",
			status:      "implemented",
			description: "Test desc.",
			deps:        []string{"dep-a"},
			wantContain: []string{
				"Test desc.",
				"## Dependencies",
				"- dep-a",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateReadme(tt.title, tt.status, tt.description, tt.deps)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("generateReadme() missing expected string %q\ngot:\n%s", want, got)
				}
			}

			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("generateReadme() should not contain %q\ngot:\n%s", absent, got)
				}
			}

			// All generated READMEs must contain the required sections.
			for _, section := range requiredSections {
				if !strings.Contains(got, section) {
					t.Errorf("generateReadme() missing required section %q\ngot:\n%s", section, got)
				}
			}

			// Must end with "None at this time.\n".
			if !strings.HasSuffix(got, "None at this time.\n") {
				t.Errorf("generateReadme() should end with %q\ngot suffix: %q",
					"None at this time.\n", got[len(got)-40:])
			}

			// Must end with a newline.
			if got[len(got)-1] != '\n' {
				t.Error("generateReadme() output must end with a newline")
			}
		})
	}
}

func Test_isValidStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   bool
	}{
		// Valid statuses.
		{"draft", true},
		{"approved", true},
		{"implemented", true},
		{"Draft", true},
		{"APPROVED", true},

		// Invalid statuses.
		{"conceptual", false},
		{"not-started", false},
		{"", false},
		{"in_progress", false},
	}

	for _, tt := range tests {
		t.Run("status_"+tt.status, func(t *testing.T) {
			t.Parallel()

			got := isValidStatus(tt.status)
			if got != tt.want {
				t.Errorf("isValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
