package task

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
)

func TestReleaseCommand_Help(t *testing.T) {
	cmd := releaseCommand()
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

func TestReleaseCommand_MissingTask(t *testing.T) {
	cmd := Command()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"release"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitcode.Error, got %T (%v)", err, err)
	}
	if ee.ExitCode() != 2 {
		t.Fatalf("exit code = %d, want 2", ee.ExitCode())
	}
}

func TestReleaseCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := releaseCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--task", "test"})
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
