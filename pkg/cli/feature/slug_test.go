package feature

// Features implemented: cli/feature/new

import (
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "lowercase with spaces",
			title: "Task Status Board",
			want:  "task-status-board",
		},
		{
			name:  "all caps acronym",
			title: "CLI",
			want:  "cli",
		},
		{
			name:  "already hyphenated",
			title: "Cross-Repo Sync",
			want:  "cross-repo-sync",
		},
		{
			name:  "parenthetical suffix stripped to alphanumeric",
			title: "Outstanding Questions (OQ)",
			want:  "outstanding-questions-oq",
		},
		{
			name:  "extra leading trailing and internal spaces",
			title: "  Extra   Spaces  ",
			want:  "extra-spaces",
		},
		{
			name:  "underscores converted to hyphens",
			title: "Hello_World",
			want:  "hello-world",
		},
		{
			name:  "leading and trailing hyphens stripped",
			title: "---Leading---Trailing---",
			want:  "leading-trailing",
		},
		{
			name:  "empty string",
			title: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateSlug(tt.title)
			if got != tt.want {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{
			name:    "valid multi-word slug",
			slug:    "task-status-board",
			wantErr: false,
		},
		{
			name:    "valid single word",
			slug:    "cli",
			wantErr: false,
		},
		{
			name:    "valid nested path with slashes",
			slug:    "cli/task/claim",
			wantErr: false,
		},
		{
			name:    "valid single character",
			slug:    "a",
			wantErr: false,
		},
		{
			name:    "invalid empty string",
			slug:    "",
			wantErr: true,
		},
		{
			name:    "invalid uppercase letter",
			slug:    "Task",
			wantErr: true,
		},
		{
			name:    "invalid consecutive hyphens",
			slug:    "foo--bar",
			wantErr: true,
		},
		{
			name:    "invalid leading hyphen",
			slug:    "-foo",
			wantErr: true,
		},
		{
			name:    "invalid trailing hyphen",
			slug:    "foo-",
			wantErr: true,
		},
		{
			name:    "invalid trailing slash segment",
			slug:    "foo/bar/",
			wantErr: true,
		},
		{
			name:    "invalid contains spaces",
			slug:    "foo bar",
			wantErr: true,
		},
		{
			name:    "invalid contains underscores",
			slug:    "foo_bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}
