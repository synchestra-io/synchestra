package state

import (
	"bytes"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func TestSyncCommand_Help(t *testing.T) {
	cmd := syncCommand()
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

func TestSyncCommand_StubReturnsNotImplemented(t *testing.T) {
	cmd := syncCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "test-project"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from stub implementation")
	}
	ec, ok := err.(*exitcode.Error)
	if !ok {
		t.Fatalf("expected *exitcode.Error, got %T: %v", err, err)
	}
	if ec.ExitCode() != 10 {
		t.Errorf("expected exit code 10, got %d", ec.ExitCode())
	}
}

func TestSyncCommand_RejectsExtraArgs(t *testing.T) {
	cmd := syncCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"extra-arg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for extra args")
	}
}
