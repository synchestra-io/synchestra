package gitops

// Features depended on: cli/project/new

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsGitRepo returns true if dir is a git repository.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetOriginURL returns the URL of the "origin" remote.
func GetOriginURL(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting origin URL for %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Clone clones a git repository from url to dest.
func Clone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloning %s to %s: %w", url, dest, err)
	}
	return nil
}

// CommitAndPush stages the given files, commits with the message, and pushes.
// On push conflict, it pulls, re-stages, and retries once.
func CommitAndPush(dir string, files []string, message string) error {
	// Stage files
	args := []string{"-C", dir, "add"}
	args = append(args, files...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add in %s: %w\n%s", dir, err, out)
	}

	// Commit
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit in %s: %w\n%s", dir, err, out)
	}

	// Push with retry on conflict
	cmd = exec.Command("git", "-C", dir, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Pull and retry once
		if pullErr := Pull(dir); pullErr != nil {
			return fmt.Errorf("git push failed and pull also failed in %s: push: %w\n%s\npull: %v", dir, err, out, pullErr)
		}
		cmd = exec.Command("git", "-C", dir, "push")
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git push in %s failed after retry: %w\n%s", dir, err2, out2)
		}
	}

	return nil
}

// Pull performs a git pull in the given directory.
func Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull in %s: %w\n%s", dir, err, out)
	}
	return nil
}
