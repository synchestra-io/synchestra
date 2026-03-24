package task

import (
	"bytes"
	"errors"
	"testing"
)

func TestInfoCommand_Help(t *testing.T) {
	cmd := infoCommand()
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

func TestInfoCommand_MissingTask(t *testing.T) {
	cmd := Command()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"info"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitError, got %T (%v)", err, err)
	}
	if ee.code != 2 {
		t.Fatalf("exit code = %d, want 2", ee.code)
	}
}

func TestInfoCommand_StubReturnsNotImplemented(t *testing.T) {
	cmd := infoCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--task", "test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from stub")
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected exitError, got %T", err)
	}
	if ee.ExitCode() != 10 {
		t.Errorf("exit code = %d, want 10", ee.ExitCode())
	}
}
