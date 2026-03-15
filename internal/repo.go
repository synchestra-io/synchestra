package internal

// Features depended on: cli/project/new, repository-types, architecture/spec-to-execution

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ingitdb/ingitdb-cli/pkg/ingitdb/gitrepo"
)

// RepoRef represents a parsed repo reference.
type RepoRef struct {
	Hosting string // e.g., "github.com"
	Org     string // e.g., "acme"
	Repo    string // e.g., "acme-api"
}

// ToHTTPSURL returns the HTTPS git URL for this repo reference.
func (r *RepoRef) ToHTTPSURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", r.Hosting, r.Org, r.Repo)
}

// NormalizeCloneURL converts a repo reference to a valid git clone URL.
// If input is already a full URL, it's returned as-is.
// If input is a short path, it's converted to HTTPS URL.
func NormalizeCloneURL(originalRef string, parsedRef *RepoRef) string {
	// If the original input is a full URL (contains :// or git@), use it as-is
	if strings.Contains(originalRef, "://") || strings.Contains(originalRef, "@") {
		return originalRef
	}
	// Otherwise it's a short path; convert to HTTPS URL
	return parsedRef.ToHTTPSURL()
}

// ParseRepoReference parses a repo reference (full URL or short path)
// and returns the normalized {hosting}/{org}/{repo} form.
// Examples:
//   "https://github.com/acme/acme-api" -> github.com/acme/acme-api
//   "git@github.com:acme/acme-api" -> github.com/acme/acme-api
//   "github.com/acme/acme-api" -> github.com/acme/acme-api
func ParseRepoReference(ref string) (*RepoRef, error) {
	// Try to parse as URL
	parsedURL, err := url.Parse(ref)
	if err == nil && parsedURL.Scheme != "" {
		// It's a full URL
		hosting := parsedURL.Hostname()
		path := strings.TrimPrefix(parsedURL.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return &RepoRef{
				Hosting: hosting,
				Org:     parts[0],
				Repo:    parts[1],
			}, nil
		}
	}

	// Try to parse as SSH URL (git@github.com:org/repo)
	if strings.Contains(ref, "@") && strings.Contains(ref, ":") {
		parts := strings.Split(ref, "@")
		if len(parts) == 2 {
			hostPath := parts[1]
			hostParts := strings.Split(hostPath, ":")
			if len(hostParts) == 2 {
				hosting := hostParts[0]
				path := hostParts[1]
				path = strings.TrimSuffix(path, ".git")
				repoParts := strings.Split(path, "/")
				if len(repoParts) >= 2 {
					return &RepoRef{
						Hosting: hosting,
						Org:     repoParts[0],
						Repo:    repoParts[1],
					}, nil
				}
			}
		}
	}

	// Try to parse as short path (github.com/org/repo)
	parts := strings.Split(ref, "/")
	if len(parts) >= 3 {
		return &RepoRef{
			Hosting: parts[0],
			Org:     parts[1],
			Repo:    parts[2],
		}, nil
	}

	return nil, fmt.Errorf("invalid repo reference: %s", ref)
}

// ResolveRepoPath returns the local path for a repo reference.
// Returns an error if any component contains path traversal sequences.
func ResolveRepoPath(reposDir string, ref *RepoRef) (string, error) {
	// Validate components against path traversal: only allow safe characters
	for _, component := range []string{ref.Hosting, ref.Org, ref.Repo} {
		// Reject if contains .. or path separators
		if strings.Contains(component, "..") || strings.ContainsAny(component, "/\\") {
			return "", fmt.Errorf("path traversal detected in component: %s", component)
		}
		// Reject if starts with . (hidden files/relative paths)
		if strings.HasPrefix(component, ".") {
			return "", fmt.Errorf("invalid component starts with '.': %s", component)
		}
	}

	resolvedPath := filepath.Join(reposDir, ref.Hosting, ref.Org, ref.Repo)

	// Ensure the resolved path is still under reposDir
	cleanReposDir := filepath.Clean(reposDir)
	cleanResolved := filepath.Clean(resolvedPath)
	if !strings.HasPrefix(cleanResolved, cleanReposDir+string(os.PathSeparator)) && cleanResolved != cleanReposDir {
		return "", fmt.Errorf("path traversal detected: resolved path escapes repos_dir")
	}

	return cleanResolved, nil
}

// CloneRepo clones a repo if it doesn't exist at the given path.
// It auto-creates parent directories.
// Returns exit code 3 on failure, 0 on success.
func CloneRepo(repoURL string, localPath string) (int, error) {
	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return 3, fmt.Errorf("could not create parent directories: %w", err)
	}

	// Check if already exists, using Lstat to detect symlinks
	if fileInfo, err := os.Lstat(localPath); err == nil {
		// Path exists - check if it's a symlink (security issue)
		if (fileInfo.Mode() & os.ModeSymlink) != 0 {
			return 3, fmt.Errorf("path is a symlink, which is not allowed: %s", localPath)
		}
		// Regular directory or file exists
		return 0, nil
	} else if !os.IsNotExist(err) {
		// Some other error (permission denied, etc.)
		return 3, fmt.Errorf("could not check path existence: %w", err)
	}

	// Clone the repo
	cmd := exec.Command("git", "clone", repoURL, localPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return 3, fmt.Errorf("clone failed for %s: %s", repoURL, string(output))
	}

	return 0, nil
}

// ValidateGitRepo checks if a directory is a valid git repository.
// Returns exit code 3 if not a git repo, 0 if valid.
func ValidateGitRepo(path string) (int, error) {
	// Use ingitdb's FindRepoRoot to validate
	_, err := gitrepo.FindRepoRoot(path)
	if err != nil {
		return 3, fmt.Errorf("%s is not a git repository: %w", path, err)
	}
	return 0, nil
}

// GetOriginURL returns the origin URL of a git repo.
func GetOriginURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not get origin URL: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// GitCommitAndPush commits changes to all repos and pushes them.
// On push conflict, it pulls, re-checks, and retries once.
func GitCommitAndPush(repoPath string, commitMessage string) error {
	// Stage all changes
	cmd := exec.Command("git", "-C", repoPath, "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed in %s: %s", repoPath, string(output))
	}

	// Commit
	cmd = exec.Command("git", "-C", repoPath, "commit", "-m", commitMessage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if there's nothing to commit
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed in %s: %s", repoPath, string(output))
	}

	// Push with retry logic
	return gitPushWithRetry(repoPath, 1)
}

// gitPushWithRetry pushes and retries once on conflict.
func gitPushWithRetry(repoPath string, attemptsRemaining int) error {
	cmd := exec.Command("git", "-C", repoPath, "push")
	output, err := cmd.CombinedOutput()

	if err == nil {
		return nil // Success
	}

	if attemptsRemaining > 0 && (strings.Contains(string(output), "rejected") || strings.Contains(string(output), "conflict")) {
		// Pull with --rebase to avoid creating merge commits
		cmd = exec.Command("git", "-C", repoPath, "pull", "--rebase")
		if pullOutput, pullErr := cmd.CombinedOutput(); pullErr != nil {
			return fmt.Errorf("git pull --rebase failed: %s", string(pullOutput))
		}

		// Retry push once
		return gitPushWithRetry(repoPath, attemptsRemaining-1)
	}

	return fmt.Errorf("git push failed in %s: %s", repoPath, string(output))
}
