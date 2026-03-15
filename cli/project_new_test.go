package cli

import (
	"errors"
	"testing"

	"github.com/synchesta-io/synchestra/internal"
)

func TestProjectNewRun_MissingTargetRepo(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	err := projectNewRun(mockHomeDir, "github.com/test/spec", "github.com/test/state", []string{}, "")
	if err == nil {
		t.Fatalf("expected error for missing --target-repo")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("exit code: got %d, want 2", exitErr.Code)
	}
}

func TestProjectNewRun_InvalidSpecRepo(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	err := projectNewRun(mockHomeDir, "invalid", "github.com/test/state", []string{"github.com/test/api"}, "")
	if err == nil {
		t.Fatalf("expected error for invalid spec repo")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("exit code: got %d, want 2", exitErr.Code)
	}
}

func TestProjectNewRun_InvalidStateRepo(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	err := projectNewRun(mockHomeDir, "github.com/test/spec", "invalid", []string{"github.com/test/api"}, "")
	if err == nil {
		t.Fatalf("expected error for invalid state repo")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("exit code: got %d, want 2", exitErr.Code)
	}
}

func TestProjectNewRun_InvalidTargetRepo(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	err := projectNewRun(mockHomeDir, "github.com/test/spec", "github.com/test/state", []string{"invalid"}, "")
	if err == nil {
		t.Fatalf("expected error for invalid target repo")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("exit code: got %d, want 2", exitErr.Code)
	}
}

func TestProjectNewRun_PathTraversalRejectsPathTraversalRepo(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	// Try to use a repo reference with path traversal
	err := projectNewRun(mockHomeDir, "github.com/../../../etc/spec", "github.com/test/state", []string{"github.com/test/api"}, "")
	if err == nil {
		t.Fatalf("expected error for path traversal in spec repo")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("exit code: got %d, want 2", exitErr.Code)
	}
}

func TestProjectNewRun_HomeDirError(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "", errors.New("permission denied")
	}

	err := projectNewRun(mockHomeDir, "github.com/test/spec", "github.com/test/state", []string{"github.com/test/api"}, "")
	if err == nil {
		t.Fatalf("expected error when home dir fails")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 10 {
		t.Errorf("exit code: got %d, want 10", exitErr.Code)
	}
}

func TestProjectNewRun_MultipleTargetRepos(t *testing.T) {
	mockHomeDir := func() (string, error) {
		return "/home/user", nil
	}

	// This should fail at clone stage (repos don't exist), but it tests that
	// multiple target repos are parsed and processed
	err := projectNewRun(
		mockHomeDir,
		"github.com/test/spec",
		"github.com/test/state",
		[]string{"github.com/test/api", "github.com/test/web"},
		"",
	)

	// Should fail at clone stage with exit code 3
	if err == nil {
		t.Fatalf("expected error when cloning nonexistent repos")
	}

	var exitErr *internal.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T", err)
	}

	if exitErr.Code != 3 {
		t.Errorf("exit code: got %d, want 3 (repo not found)", exitErr.Code)
	}
}
