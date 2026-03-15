// Package reporef parses and resolves Synchestra repository references.
package reporef

// Features implemented: cli/project/new

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Ref is a parsed repository reference.
type Ref struct {
	Hosting string // e.g. "github.com"
	Org     string // e.g. "acme"
	Repo    string // e.g. "acme-spec"
}

// Parse parses a repo reference in any supported format:
//   - Short form:  "github.com/org/repo"
//   - HTTPS URL:   "https://github.com/org/repo" or "https://github.com/org/repo.git"
//   - HTTP URL:    "http://github.com/org/repo"
//   - SSH URL:     "git@github.com:org/repo" or "git@github.com:org/repo.git"
func Parse(ref string) (Ref, error) {
	var path string
	switch {
	case strings.HasPrefix(ref, "https://"):
		path = strings.TrimPrefix(ref, "https://")
	case strings.HasPrefix(ref, "http://"):
		path = strings.TrimPrefix(ref, "http://")
	case strings.HasPrefix(ref, "git@"):
		// git@github.com:org/repo -> github.com/org/repo
		rest := strings.TrimPrefix(ref, "git@")
		path = strings.Replace(rest, ":", "/", 1)
	default:
		path = ref
	}

	// Strip trailing .git
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Ref{}, fmt.Errorf("invalid repo reference %q: expected hosting/org/repo", ref)
	}
	return Ref{Hosting: parts[0], Org: parts[1], Repo: parts[2]}, nil
}

// LocalPath returns the absolute local filesystem path for this repo under reposDir.
func (r Ref) LocalPath(reposDir string) string {
	return filepath.Join(reposDir, r.Hosting, r.Org, r.Repo)
}

// OriginURL returns the canonical HTTPS origin URL for this repo.
func (r Ref) OriginURL() string {
	return fmt.Sprintf("https://%s/%s/%s", r.Hosting, r.Org, r.Repo)
}
