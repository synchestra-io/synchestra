package reporef

// Features implemented: global-config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Ref is a parsed repository reference.
type Ref struct {
	Hosting string // e.g., "github.com"
	Org     string // e.g., "acme"
	Repo    string // e.g., "acme-api"
}

// Parse parses a repo reference string into a Ref.
// Accepts:
//   - Short path: github.com/org/repo
//   - HTTPS URL: https://github.com/org/repo
//   - SSH URL: git@github.com:org/repo
func Parse(s string) (Ref, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Ref{}, fmt.Errorf("empty repo reference")
	}

	var hosting, org, repo string

	switch {
	case strings.HasPrefix(s, "git@"):
		// git@github.com:org/repo
		rest := strings.TrimPrefix(s, "git@")
		var path string
		var ok bool
		hosting, path, ok = strings.Cut(rest, ":")
		if !ok {
			return Ref{}, fmt.Errorf("invalid SSH repo reference: %q", s)
		}
		org, repo = splitOrgRepo(path)

	case strings.Contains(s, "://"):
		// https://github.com/org/repo
		u, err := url.Parse(s)
		if err != nil {
			return Ref{}, fmt.Errorf("invalid repo URL: %q: %w", s, err)
		}
		hosting = u.Host
		org, repo = splitOrgRepo(strings.TrimPrefix(u.Path, "/"))

	default:
		// github.com/org/repo
		parts := strings.Split(s, "/")
		if len(parts) != 3 {
			return Ref{}, fmt.Errorf("invalid repo reference %q: expected hosting/org/repo", s)
		}
		hosting, org, repo = parts[0], parts[1], parts[2]
	}

	repo = strings.TrimSuffix(repo, ".git")

	if hosting == "" || org == "" || repo == "" {
		return Ref{}, fmt.Errorf("invalid repo reference %q: missing hosting, org, or repo", s)
	}
	if strings.Contains(repo, "/") {
		return Ref{}, fmt.Errorf("invalid repo reference %q: too many path segments", s)
	}

	return Ref{Hosting: hosting, Org: org, Repo: repo}, nil
}

func splitOrgRepo(path string) (string, string) {
	path = strings.TrimSuffix(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return "", ""
	}
	if len(parts) > 2 {
		return "", "" // too many segments
	}
	return parts[0], parts[1]
}

// OriginURL returns the HTTPS URL for this repo.
func (r Ref) OriginURL() string {
	return "https://" + r.Hosting + "/" + r.Org + "/" + r.Repo
}

// DiskPath returns the local filesystem path for this repo under reposDir.
func (r Ref) DiskPath(reposDir string) string {
	return filepath.Join(reposDir, r.Hosting, r.Org, r.Repo)
}

// Identifier returns the short-form identifier: hosting/org/repo.
func (r Ref) Identifier() string {
	return r.Hosting + "/" + r.Org + "/" + r.Repo
}
