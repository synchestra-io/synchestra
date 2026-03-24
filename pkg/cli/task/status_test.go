package task

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func TestStatusCommand_Help(t *testing.T) {
	cmd := statusCommand()
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

func TestStatusCommand_MissingTask(t *testing.T) {
	cmd := Command()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"status"})
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

func TestStatusCommand_PartialUpdateFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "current without new", args: []string{"--task", "t", "--current", "claimed"}},
		{name: "new without current", args: []string{"--task", "t", "--new", "in_progress"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(append([]string{"status"}, tt.args...))
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
		})
	}
}

func TestStatusCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := statusCommand()
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
