package reporef_test

import (
	"path/filepath"
	"testing"

	"github.com/synchesta-io/synchestra/cli/internal/reporef"
)

func TestParse(t *testing.T) {
	cases := []struct {
		input       string
		wantHosting string
		wantOrg     string
		wantRepo    string
		wantErr     bool
	}{
		{
			input:       "github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		{
			input:       "https://github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		{
			input:       "git@github.com:acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// .git suffix stripped
		{
			input:       "https://github.com/acme/acme-spec.git",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// http:// (non-TLS)
		{
			input:       "http://github.com/acme/acme-spec",
			wantHosting: "github.com", wantOrg: "acme", wantRepo: "acme-spec",
		},
		// sub-paths are invalid
		{input: "github.com/org/repo/extra", wantErr: true},
		{input: "notaref", wantErr: true},
		{input: "github.com/only-one-segment", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := reporef.Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Hosting != tc.wantHosting {
				t.Errorf("Hosting = %q, want %q", got.Hosting, tc.wantHosting)
			}
			if got.Org != tc.wantOrg {
				t.Errorf("Org = %q, want %q", got.Org, tc.wantOrg)
			}
			if got.Repo != tc.wantRepo {
				t.Errorf("Repo = %q, want %q", got.Repo, tc.wantRepo)
			}
		})
	}
}

func TestRef_LocalPath(t *testing.T) {
	ref, err := reporef.Parse("github.com/acme/acme-spec")
	if err != nil {
		t.Fatal(err)
	}
	got := ref.LocalPath("/home/user/synchestra/repos")
	want := filepath.Join("/home/user/synchestra/repos", "github.com", "acme", "acme-spec")
	if got != want {
		t.Errorf("LocalPath = %q, want %q", got, want)
	}
}

func TestRef_OriginURL(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"github.com/acme/acme-spec", "https://github.com/acme/acme-spec"},
		{"https://github.com/acme/acme-spec", "https://github.com/acme/acme-spec"},
		{"git@github.com:acme/acme-spec", "https://github.com/acme/acme-spec"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			ref, err := reporef.Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got := ref.OriginURL(); got != tc.want {
				t.Errorf("OriginURL = %q, want %q", got, tc.want)
			}
		})
	}
}
