package task

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func TestClaimCommand_Help(t *testing.T) {
	cmd := claimCommand()
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

func TestClaimCommand_MissingFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing task", args: []string{"--run", "42", "--model", "sonnet"}, want: "--task is required"},
		{name: "missing run", args: []string{"--task", "t", "--model", "sonnet"}, want: "--run is required"},
		{name: "missing model", args: []string{"--task", "t", "--run", "42"}, want: "--model is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(append([]string{"claim"}, tt.args...))
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

func TestClaimCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := claimCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--task", "t", "--run", "42", "--model", "sonnet"})
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
