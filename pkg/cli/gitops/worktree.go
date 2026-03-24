package gitops

// Features implemented: embedded-state
// Features depended on:  cli/project/init

import (
	"fmt"
	"os/exec"
	"strings"
)

// WorktreeAdd creates a git worktree at the given path, checked out to the named branch.
func WorktreeAdd(repoDir, worktreePath, branch string) error {
	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", worktreePath, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("adding worktree at %s for branch %s: %w\n%s", worktreePath, branch, err, out)
	}
	return nil
}

// WorktreeList returns the paths of all active worktrees in the repository.
func WorktreeList(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoDir, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees in %s: %w", repoDir, err)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths, nil
}

// WorktreePrune removes stale worktree entries.
func WorktreePrune(repoDir string) error {
	cmd := exec.Command("git", "-C", repoDir, "worktree", "prune")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pruning worktrees in %s: %w\n%s", repoDir, err, out)
	}
	return nil
}

// CreateOrphanBranch creates an orphan branch with no history.
// It checks out the orphan branch, removes all tracked files, and returns.
// The caller is responsible for adding files, committing, and switching back.
func CreateOrphanBranch(repoDir, branch string) error {
	cmd := exec.Command("git", "-C", repoDir, "checkout", "--orphan", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating orphan branch %s: %w\n%s", branch, err, out)
	}
	// Remove all tracked files from the index (clean slate for the orphan branch).
	cmd = exec.Command("git", "-C", repoDir, "rm", "-rf", "--quiet", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clearing index on orphan branch %s: %w\n%s", branch, err, out)
	}
	return nil
}

// RemoteBranchExists checks if a branch exists on the given remote.
func RemoteBranchExists(repoDir, remote, branch string) bool {
	cmd := exec.Command("git", "-C", repoDir, "ls-remote", "--heads", remote, "refs/heads/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// FetchBranch fetches a specific branch from the remote.
func FetchBranch(repoDir, remote, branch string) error {
	cmd := exec.Command("git", "-C", repoDir, "fetch", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w\n%s", remote, branch, err, out)
	}
	return nil
}

// CreateTrackingBranch creates a local branch tracking a remote branch.
func CreateTrackingBranch(repoDir, branch, remote string) error {
	cmd := exec.Command("git", "-C", repoDir, "branch", branch, remote+"/"+branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating tracking branch %s: %w\n%s", branch, err, out)
	}
	return nil
}

// PushNewBranch pushes a branch to the remote with upstream tracking.
func PushNewBranch(repoDir, remote, branch string) error {
	cmd := exec.Command("git", "-C", repoDir, "push", "-u", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pushing branch %s to %s: %w\n%s", branch, remote, err, out)
	}
	return nil
}

// RepoRoot returns the absolute path to the repository root.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("finding repo root for %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}
