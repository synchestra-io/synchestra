package internal

import (
	"testing"
)

func TestParseRepoReference(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantHost  string
		wantOrg   string
		wantRepo  string
		wantError bool
	}{
		{
			name:     "https URL",
			ref:      "https://github.com/acme/acme-api",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "acme-api",
		},
		{
			name:     "https URL with .git",
			ref:      "https://github.com/acme/acme-api.git",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "acme-api",
		},
		{
			name:     "SSH URL",
			ref:      "git@github.com:acme/acme-api",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "acme-api",
		},
		{
			name:     "SSH URL with .git",
			ref:      "git@github.com:acme/acme-api.git",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "acme-api",
		},
		{
			name:     "short path",
			ref:      "github.com/acme/acme-api",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "acme-api",
		},
		{
			name:      "invalid reference",
			ref:       "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepoReference(tt.ref)
			if (err != nil) != tt.wantError {
				t.Fatalf("error: got %v, wantError %v", err, tt.wantError)
			}
			if err != nil {
				return
			}
			if got.Hosting != tt.wantHost || got.Org != tt.wantOrg || got.Repo != tt.wantRepo {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)",
					got.Hosting, got.Org, got.Repo,
					tt.wantHost, tt.wantOrg, tt.wantRepo)
			}
		})
	}
}

func TestResolveRepoPath_Security(t *testing.T) {
	tests := []struct {
		name      string
		reposDir  string
		ref       *RepoRef
		wantError bool
	}{
		{
			name:      "valid path",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: "github.com", Org: "acme", Repo: "api"},
			wantError: false,
		},
		{
			name:      "path traversal in hosting",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: "..", Org: "acme", Repo: "api"},
			wantError: true,
		},
		{
			name:      "path traversal in org",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: "github.com", Org: "..", Repo: "api"},
			wantError: true,
		},
		{
			name:      "path traversal in repo",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: "github.com", Org: "acme", Repo: "../evil"},
			wantError: true,
		},
		{
			name:      "slash in component",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: "github.com", Org: "acme/evil", Repo: "api"},
			wantError: true,
		},
		{
			name:      "dot prefix",
			reposDir:  "/repos",
			ref:       &RepoRef{Hosting: ".github", Org: "acme", Repo: "api"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveRepoPath(tt.reposDir, tt.ref)
			if (err != nil) != tt.wantError {
				t.Errorf("error: got %v, wantError %v", err, tt.wantError)
			}
			if err == nil && tt.wantError {
				t.Errorf("expected error for path %q", got)
			}
		})
	}
}

func TestNormalizeCloneURL(t *testing.T) {
	tests := []struct {
		name        string
		originalRef string
		parsed      *RepoRef
		expected    string
	}{
		{
			name:        "https URL passthrough",
			originalRef: "https://github.com/acme/api",
			parsed:      &RepoRef{Hosting: "github.com", Org: "acme", Repo: "api"},
			expected:    "https://github.com/acme/api",
		},
		{
			name:        "SSH URL passthrough",
			originalRef: "git@github.com:acme/api",
			parsed:      &RepoRef{Hosting: "github.com", Org: "acme", Repo: "api"},
			expected:    "git@github.com:acme/api",
		},
		{
			name:        "short path conversion",
			originalRef: "github.com/acme/api",
			parsed:      &RepoRef{Hosting: "github.com", Org: "acme", Repo: "api"},
			expected:    "https://github.com/acme/api.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCloneURL(tt.originalRef, tt.parsed)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
