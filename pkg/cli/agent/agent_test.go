package agent

import (
	"bytes"
	"errors"
	"testing"

	"github.com/synchestra-io/specscore/pkg/exitcode"
)

func TestCommand_Help(t *testing.T) {
	cmd := Command()
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
		{name: "missing repository", args: []string{"--run", "r", "--branch", "b"}, want: "--repository is required"},
		{name: "missing run", args: []string{"--repository", "repo", "--branch", "b"}, want: "--run is required"},
		{name: "missing branch", args: []string{"--repository", "repo", "--run", "r"}, want: "--branch is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(append([]string{"claim"}, tt.args...))
			assertInvalidArgs(t, cmd.Execute(), tt.want)
		})
	}
}

func TestRenewReleaseHandoff_MissingFlags(t *testing.T) {
	tests := []struct {
		verb string
		args []string
		want string
	}{
		{"renew", []string{}, "--claim is required"},
		{"renew", []string{"--claim", "c"}, "--fence-epoch and --fence-token are required"},
		{"release", []string{}, "--claim is required"},
		{"release", []string{"--claim", "c"}, "--fence-epoch and --fence-token are required"},
		{"handoff", []string{}, "--claim is required"},
		{"handoff", []string{"--claim", "c"}, "--to-run is required"},
		{"handoff", []string{"--claim", "c", "--to-run", "r"}, "--fence-epoch and --fence-token are required"},
	}
	for _, tt := range tests {
		t.Run(tt.verb+"/"+tt.want, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(append([]string{tt.verb}, tt.args...))
			assertInvalidArgs(t, cmd.Execute(), tt.want)
		})
	}
}

func TestMessageCommands_MissingFlags(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"message", "send", "--run", "r"}, "--thread is required"},
		{[]string{"message", "send", "--thread", "t"}, "--run is required"},
		{[]string{"message", "list"}, "--thread is required"},
		{[]string{"message", "ack", "--run", "r"}, "--message is required"},
		{[]string{"message", "ack", "--message", "m"}, "--run is required"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(tt.args)
			assertInvalidArgs(t, cmd.Execute(), tt.want)
		})
	}
}

func TestRunCommands_MissingFlags(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"run", "start", "--family", "claude"}, "--effort is required"},
		{[]string{"run", "start", "--effort", "e1"}, "--family is required"},
		{[]string{"run", "start", "--effort", "e1", "--family", "claude", "--model-provenance", "runtime_observed"}, "--model-provenance requires --model"},
		{[]string{"run", "start", "--effort", "e1", "--family", "claude", "--model", "m"}, "--model-provenance must be runtime_observed or caller_declared when --model is set"},
		{[]string{"run", "correct"}, "--run is required"},
		{[]string{"run", "correct", "--run", "r1", "--model", "m"}, "--model-provenance must be runtime_observed or caller_declared when --model is set"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			cmd := Command()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			cmd.SetArgs(tt.args)
			assertInvalidArgs(t, cmd.Execute(), tt.want)
		})
	}
}

// TestMessageSendCommand_InvalidEvidence proves --evidence entries missing a
// kind are rejected before ever reaching the store.
func TestMessageSendCommand_InvalidEvidence(t *testing.T) {
	cmd := Command()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"message", "send", "--thread", "t", "--run", "r", "--evidence", ":no-kind:ref"})
	assertInvalidArgs(t, cmd.Execute(), `--evidence ":no-kind:ref": kind is required (expected kind:description:reference)`)
}

func assertInvalidArgs(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitcode.Error, got %T (%v)", err, err)
	}
	if ee.ExitCode() != exitcode.InvalidArgs {
		t.Fatalf("exit code = %d, want %d (message %q)", ee.ExitCode(), exitcode.InvalidArgs, ee.Error())
	}
	if ee.Error() != want {
		t.Fatalf("message = %q, want %q", ee.Error(), want)
	}
}

func TestClaimCommand_NoProjectReturnsNotFound(t *testing.T) {
	cmd := Command()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"claim", "--repository", "r", "--run", "run-a", "--branch", "b"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no project/state repo can be resolved")
	}
	var ee *exitcode.Error
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitcode.Error, got %T", err)
	}
	// Depending on the working directory this test runs from, resolution can
	// fail at "no state repo" (NotFound, 3) or "no git remote to derive
	// --project from" (InvalidArgs, 2) — both are legitimate fail-closed
	// outcomes; a bare success is the only wrong answer.
	if ee.ExitCode() != exitcode.NotFound && ee.ExitCode() != exitcode.InvalidArgs {
		t.Fatalf("exit code = %d, want %d (not found) or %d (invalid args)", ee.ExitCode(), exitcode.NotFound, exitcode.InvalidArgs)
	}
}
