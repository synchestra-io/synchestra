package task

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func TestListCommand_Help(t *testing.T) {
	cmd := listCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected help output, got empty string")
	}
}

func TestListCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := listCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from stub")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitcode.Error, got %T", err)
	}
	if ee.ExitCode() != 3 {
		t.Errorf("exit code = %d, want 3", ee.ExitCode())
	}
}

func TestListCommand_RejectsExtraArgs(t *testing.T) {
	cmd := listCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"extra-arg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for extra args")
	}
}
