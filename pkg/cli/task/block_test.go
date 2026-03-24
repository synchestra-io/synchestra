package task

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func TestBlockCommand_Help(t *testing.T) {
	cmd := blockCommand()
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

func TestBlockCommand_MissingFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing task", args: []string{"--reason", "waiting"}, want: "--task is required"},
		{name: "missing reason", args: []string{"--task", "t"}, want: "--reason is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(append([]string{"block"}, tt.args...))
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
			if ee.Error() != tt.want {
				t.Fatalf("message = %q, want %q", ee.Error(), tt.want)
			}
		})
	}
}

func TestBlockCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := blockCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--task", "t", "--reason", "waiting"})
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
