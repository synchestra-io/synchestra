package task

import (
	"bytes"
	"sort"
	"testing"
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

func TestCommand_AllSubcommandsRegistered(t *testing.T) {
	cmd := Command()
	expected := []string{
		"new", "enqueue", "claim", "start", "status",
		"complete", "fail", "block", "unblock", "release",
		"abort", "aborted", "list", "info",
	}
	sort.Strings(expected)

	var actual []string
	for _, sub := range cmd.Commands() {
		actual = append(actual, sub.Name())
	}
	sort.Strings(actual)

	if len(actual) != len(expected) {
		t.Fatalf("expected %d subcommands, got %d: %v", len(expected), len(actual), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Errorf("subcommand mismatch at %d: got %q, want %q", i, actual[i], expected[i])
		}
	}
}
