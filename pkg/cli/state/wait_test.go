package state

// Features implemented: cli/state/wait
// Features depended on:  state-store/topology, state-store/backends/git

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// newTestWaitCommand mirrors how the real `wait` subcommand behaves once
// attached to synchestra's root command: root sets SilenceUsage/
// SilenceErrors (pkg/cli/main.go), which cobra's own error path already
// consults for a leaf command run through it (see cobra's
// `!cmd.SilenceUsage && !c.SilenceUsage` check). Constructing waitCommand()
// standalone (as every test in this file does, to avoid pulling in the
// whole command tree) makes it its own root, so it must set the same flags
// itself to get an equivalent, representative signal -- otherwise a
// legitimate, well-formed "the barrier timed out" run would dump a full
// usage/flags block after the command's own JSON/text output, exactly as it
// must never do for a real user.
func newTestWaitCommand() *cobra.Command {
	cmd := waitCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd
}

func TestWaitCommand_Help(t *testing.T) {
	cmd := newTestWaitCommand()
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

func TestWaitCommand_RequiresProjectAndReplica(t *testing.T) {
	cases := [][]string{
		{},
		{"--project", "github.com/acme/service"},
		{"--replica", "git-mirror"},
	}
	for _, args := range cases {
		cmd := newTestWaitCommand()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs(append(append([]string(nil), args...), "--cursor", "1:1"))
		err := cmd.Execute()
		ec, ok := err.(*exitcode.Error)
		if !ok || ec.ExitCode() != exitcode.InvalidArgs {
			t.Errorf("args=%v: err = %v, want exit code %d", args, err, exitcode.InvalidArgs)
		}
	}
}

func TestWaitCommand_RequiresExactlyOneOfCursorOrSourceDir(t *testing.T) {
	cmd := newTestWaitCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "p", "--replica", "r"})
	err := cmd.Execute()
	ec, ok := err.(*exitcode.Error)
	if !ok || ec.ExitCode() != exitcode.InvalidArgs {
		t.Fatalf("err = %v, want InvalidArgs for neither --cursor nor --source-dir", err)
	}

	cmd = newTestWaitCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "p", "--replica", "r", "--cursor", "1:1", "--source-dir", "/tmp/x"})
	err = cmd.Execute()
	ec, ok = err.(*exitcode.Error)
	if !ok || ec.ExitCode() != exitcode.InvalidArgs {
		t.Fatalf("err = %v, want InvalidArgs for both --cursor and --source-dir", err)
	}
}

func TestWaitCommand_RejectsBadFormatAndBadCursor(t *testing.T) {
	cmd := newTestWaitCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "p", "--replica", "r", "--cursor", "1:1", "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error for an unsupported --format")
	}

	cmd = newTestWaitCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "p", "--replica", "r", "--cursor", "not-a-cursor", "--replica-dir", t.TempDir()})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error for a malformed --cursor")
	}
}

// gitTestRepo sets up a bare "origin" plus one clone with the journal schema
// committed and pushed, mirroring the fixture shape
// pkg/state/replication's own physical tests use.
func gitTestRepo(t *testing.T) (bare, clone string) {
	t.Helper()
	bare = filepath.Join(t.TempDir(), "origin.git")
	runGit(t, t.TempDir(), "init", "--bare", "--initial-branch=main", bare)
	clone = filepath.Join(t.TempDir(), "writer")
	runGit(t, "", "clone", bare, clone)
	runGit(t, clone, "config", "user.email", "test@example.com")
	runGit(t, clone, "config", "user.name", "Test")

	ctx := context.Background()
	database, err := dalgo2ingitdb.NewDatabase(clone, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	if err := replication.EnsureDALJournalSchema(ctx, database); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", "-A")
	runGit(t, clone, "commit", "-m", "journal schema")
	runGit(t, clone, "push", "origin", "HEAD:refs/heads/main")
	return bare, clone
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func seedEvents(t *testing.T, clone, projectID string) []replication.Event {
	t.Helper()
	ctx := context.Background()
	database, err := dalgo2ingitdb.NewDatabase(clone, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	journal, err := replication.NewDALJournal(database, replication.DALJournalOptions{
		ProjectID: projectID, EndpointID: "git-active", Role: replication.RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: &zero, MaxBatchDelayMS: &zero,
	})
	if err != nil {
		t.Fatal(err)
	}
	pushed, err := replication.NewGitPushJournal(journal, clone, "origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"body": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	event, err := replication.NewEvent(replication.Event{
		ProjectID: projectID, EventID: "event-1", Cursor: replication.Cursor{Epoch: 1, Sequence: 1},
		OccurredAt: time.Now().UTC(), ActorID: "agent:test", CommandID: "cmd-1", IdempotencyKey: "event-1",
		Kind: "message.sent", Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pushed.AppendAndPush(ctx, event); err != nil {
		t.Fatal(err)
	}
	return []replication.Event{event}
}

func TestWaitCommand_SatisfiesAndProvesRemoteDurability(t *testing.T) {
	bare, clone := gitTestRepo(t)
	events := seedEvents(t, clone, "github.com/fair-split/relay")
	target := events[0].Cursor

	cmd := newTestWaitCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--project", "github.com/fair-split/relay",
		"--replica", "git-mirror",
		"--replica-dir", clone,
		"--cursor", formatCursor(target),
		"--remote", "origin", "--branch", "main",
		"--timeout", "3s", "--poll-interval", "100ms",
		"--format", "json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("wait command: %v\noutput: %s", err, out.String())
	}

	var got waitOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", err, out.String())
	}
	if !got.Satisfied || !got.RemoteProven {
		t.Fatalf("output = %+v, want satisfied and remote-proven", got)
	}
	remoteHead := gitRevParse(t, bare, "refs/heads/main")
	if got.CommitSHA != remoteHead {
		t.Fatalf("commit_sha = %q, want the bare origin's actual head %q", got.CommitSHA, remoteHead)
	}
	if got.Observed != formatCursor(target) {
		t.Fatalf("observed_cursor = %q, want %q", got.Observed, formatCursor(target))
	}
}

func gitRevParse(t *testing.T, dir, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", ref)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v", ref, err)
	}
	return string(bytes.TrimSpace(out))
}

func TestWaitCommand_TimesOutAndReportsConflictExitCode(t *testing.T) {
	_, clone := gitTestRepo(t)
	// Never seed any events -- the requested cursor can never be reached.

	cmd := newTestWaitCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--project", "github.com/fair-split/relay",
		"--replica", "git-mirror",
		"--replica-dir", clone,
		"--cursor", "1:1",
		"--remote", "origin", "--branch", "main",
		"--timeout", "80ms", "--poll-interval", "10ms",
		"--format", "json",
	})
	err := cmd.Execute()
	ec, ok := err.(*exitcode.Error)
	if !ok || ec.ExitCode() != exitcode.Conflict {
		t.Fatalf("err = %v, want exit code %d (Conflict)", err, exitcode.Conflict)
	}

	var got waitOutput
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", jsonErr, out.String())
	}
	if got.Satisfied {
		t.Fatalf("output = %+v, want unsatisfied on timeout", got)
	}
}

func TestWaitCommand_SourceDirResolvesTargetCursor(t *testing.T) {
	_, clone := gitTestRepo(t)
	events := seedEvents(t, clone, "github.com/fair-split/relay")

	cmd := newTestWaitCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--project", "github.com/fair-split/relay",
		"--replica", "git-mirror",
		"--replica-dir", clone,
		"--source-dir", clone,
		"--remote", "origin", "--branch", "main",
		"--timeout", "3s", "--poll-interval", "100ms",
		"--format", "json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("wait command: %v\noutput: %s", err, out.String())
	}
	var got waitOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if got.Requested != formatCursor(events[0].Cursor) {
		t.Fatalf("requested_cursor = %q, want the source's head %q", got.Requested, formatCursor(events[0].Cursor))
	}
	if !got.Satisfied {
		t.Fatalf("output = %+v, want satisfied", got)
	}
}

func TestWaitCommand_TextFormatIsHumanReadable(t *testing.T) {
	bare, clone := gitTestRepo(t)
	_ = bare
	events := seedEvents(t, clone, "github.com/fair-split/relay")

	cmd := newTestWaitCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--project", "github.com/fair-split/relay",
		"--replica", "git-mirror",
		"--replica-dir", clone,
		"--cursor", formatCursor(events[0].Cursor),
		"--remote", "origin", "--branch", "main",
		"--timeout", "3s", "--poll-interval", "100ms",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("wait command: %v\noutput: %s", err, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("mirror barrier satisfied")) {
		t.Fatalf("text output = %q, want a human-readable satisfied message", out.String())
	}
}

// TestWaitCommand_DefaultsReplicaDirToResolvedProjectPath proves the
// convenience default: with --replica-dir omitted, the command falls back to
// resolve.StateRepoPath(cwd), matching every other `synchestra state`
// command's --project resolution convention.
func TestWaitCommand_DefaultsReplicaDirToResolvedProjectPath(t *testing.T) {
	_, clone := gitTestRepo(t)
	events := seedEvents(t, clone, "github.com/fair-split/relay")
	if err := os.WriteFile(filepath.Join(clone, "synchestra-state-repo.yaml"), []byte("project: github.com/fair-split/relay\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestWaitCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--project", "github.com/fair-split/relay",
		"--replica", "git-mirror",
		"--cursor", formatCursor(events[0].Cursor),
		"--remote", "origin", "--branch", "main",
		"--timeout", "3s", "--poll-interval", "100ms",
		"--format", "json",
	})

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(clone); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("wait command: %v\noutput: %s", err, out.String())
	}
	var got waitOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if !got.Satisfied {
		t.Fatalf("output = %+v, want satisfied using the auto-resolved replica dir", got)
	}
}
