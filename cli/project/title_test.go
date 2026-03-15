package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveTitle_ExplicitFlag(t *testing.T) {
	got := DeriveTitle("My Project", t.TempDir(), "acme-spec")
	if got != "My Project" {
		t.Errorf("got %q, want My Project", got)
	}
}

func TestDeriveTitle_FromREADME(t *testing.T) {
	dir := t.TempDir()
	readme := "# Acme Platform\n\nSome description.\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatal(err)
	}
	got := DeriveTitle("", dir, "acme-spec")
	if got != "Acme Platform" {
		t.Errorf("got %q, want Acme Platform", got)
	}
}

func TestDeriveTitle_FromREADME_NoHeading(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("No heading here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := DeriveTitle("", dir, "acme-spec")
	if got != "acme-spec" {
		t.Errorf("got %q, want acme-spec", got)
	}
}

func TestDeriveTitle_NoREADME(t *testing.T) {
	got := DeriveTitle("", t.TempDir(), "acme-spec")
	if got != "acme-spec" {
		t.Errorf("got %q, want acme-spec", got)
	}
}

func TestExtractFirstHeading(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"# Hello World\n", "Hello World"},
		{"  # Trimmed  \n", "Trimmed"},
		{"## Not H1\n# Actual H1\n", "Actual H1"},
		{"no heading\n", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractFirstHeading([]byte(tc.input))
		if got != tc.want {
			t.Errorf("extractFirstHeading(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
