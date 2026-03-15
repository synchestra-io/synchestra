// Package gitops provides git operations used by Synchestra CLI commands.
package gitops

// Features implemented: cli/project/new

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner holds git operation implementations.
// Each field is a function so tests can substitute fakes.
type Runner struct {
	IsRepo        func(dir string) (bool, error)
	Clone         func(url, dir string) error
	OriginURL     func(dir string) (string, error)
	CommitAndPush func(dir string, files []string, msg string) error
	Push          func(dir string) error
	Pull          func(dir string) error
}

// NewRunner returns a Runner backed by real git operations.
func NewRunner() Runner {
	return Runner{
		IsRepo:        isRepo,
		Clone:         cloneRepo,
		OriginURL:     originURL,
		CommitAndPush: commitAndPush,
		Push:          push,
		Pull:          pull,
	}
}

func isRepo(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func cloneRepo(url, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	out, err := exec.Command("git", "clone", url, dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %q into %q: %w\n%s", url, dir, err, out)
	}
	return nil
}

func originURL(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func commitAndPush(dir string, files []string, msg string) error {
	addArgs := append([]string{"-C", dir, "add", "--"}, files...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}
	out, err := exec.Command("git", "-C", dir, "commit", "-m", msg).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	out, err = exec.Command("git", "-C", dir, "push", "--set-upstream", "origin", "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %w\n%s", err, out)
	}
	return nil
}

func push(dir string) error {
	out, err := exec.Command("git", "-C", dir, "push", "--set-upstream", "origin", "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %w\n%s", err, out)
	}
	return nil
}

func pull(dir string) error {
	out, err := exec.Command("git", "-C", dir, "pull").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull: %w\n%s", err, out)
	}
	return nil
}
