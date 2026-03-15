package reporef

// Features implemented: global-config

import (
	"testing"
)

func TestParse_ShortPath(t *testing.T) {
	ref, err := Parse("github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_HTTPSURL(t *testing.T) {
	ref, err := Parse("https://github.com/acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_SSHURL(t *testing.T) {
	ref, err := Parse("git@github.com:acme/acme-api")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hosting != "github.com" {
		t.Errorf("Hosting = %q, want github.com", ref.Hosting)
	}
	if ref.Org != "acme" {
		t.Errorf("Org = %q, want acme", ref.Org)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api", ref.Repo)
	}
}

func TestParse_TrailingDotGit(t *testing.T) {
	ref, err := Parse("https://github.com/acme/acme-api.git")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Repo != "acme-api" {
		t.Errorf("Repo = %q, want acme-api (should strip .git)", ref.Repo)
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := []string{
		"",
		"github.com",
		"github.com/acme",
		"just-a-name",
		"github.com/acme/api/extra/parts",
	}
	for _, input := range cases {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) should fail", input)
		}
	}
}

func TestRef_OriginURL(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.OriginURL()
	want := "https://github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_DiskPath(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.DiskPath("/home/user/synchestra/repos")
	want := "/home/user/synchestra/repos/github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_Identifier(t *testing.T) {
	ref := Ref{Hosting: "github.com", Org: "acme", Repo: "acme-api"}
	got := ref.Identifier()
	want := "github.com/acme/acme-api"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
