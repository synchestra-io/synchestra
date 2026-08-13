package state

// Features implemented: cli/state/wait
// Features depended on:  state-store/topology, state-store/backends/git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/resolve"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

func waitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Block until a replica durably applies events up to a cursor (mirror barrier)",
		Long: `Implements the mirror barrier from state-store/topology: block until the
named replica has durably applied events up to a given cursor, then return.

On the required Git mirror this proves REMOTE durability -- a live fetch and
ancestry check against the configured remote, not merely a local commit --
before reporting success. A timeout never rolls back or hides the active
write that already succeeded; it reports the exact cursor observed plus an
explicit "barrier unsatisfied" outcome (exit code 1).

The target cursor is either given directly with --cursor (epoch:sequence,
e.g. "1:42" -- typically the cursor a prior terminal command returned) or
resolved from a second local journal's current head with --source-dir.`,
		Args: cobra.NoArgs,
		RunE: runWait,
	}
	cmd.Flags().String("project", "", "project id the journal is scoped to (required)")
	cmd.Flags().String("replica", "", "replica endpoint id being waited on, for reporting (required)")
	cmd.Flags().String("replica-dir", "", "path to the replica's local Git state repository (default: the current project's resolved state repo path)")
	cmd.Flags().String("cursor", "", "target cursor as epoch:sequence")
	cmd.Flags().String("source-dir", "", "resolve the target cursor from this second local Git journal's current head, instead of --cursor")
	cmd.Flags().String("remote", "origin", "Git remote name to prove durability against")
	cmd.Flags().String("branch", "main", "Git branch to prove durability against")
	cmd.Flags().Duration("timeout", replication.DefaultWaitTimeout, "how long to wait before reporting the barrier unsatisfied")
	cmd.Flags().Duration("poll-interval", replication.DefaultWaitPollInterval, "how often to re-check the replica while waiting")
	cmd.Flags().String("format", "text", "output format: text or json")
	return cmd
}

func runWait(cmd *cobra.Command, _ []string) error {
	projectFlag, _ := cmd.Flags().GetString("project")
	replicaFlag, _ := cmd.Flags().GetString("replica")
	replicaDirFlag, _ := cmd.Flags().GetString("replica-dir")
	cursorFlag, _ := cmd.Flags().GetString("cursor")
	sourceDirFlag, _ := cmd.Flags().GetString("source-dir")
	remoteFlag, _ := cmd.Flags().GetString("remote")
	branchFlag, _ := cmd.Flags().GetString("branch")
	timeoutFlag, _ := cmd.Flags().GetDuration("timeout")
	pollFlag, _ := cmd.Flags().GetDuration("poll-interval")
	formatFlag, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(projectFlag) == "" {
		return exitcode.InvalidArgsError("--project is required")
	}
	if strings.TrimSpace(replicaFlag) == "" {
		return exitcode.InvalidArgsError("--replica is required")
	}
	if formatFlag != "text" && formatFlag != "json" {
		return exitcode.InvalidArgsErrorf("--format must be text or json, got %q", formatFlag)
	}
	hasCursor := strings.TrimSpace(cursorFlag) != ""
	hasSourceDir := strings.TrimSpace(sourceDirFlag) != ""
	if hasCursor == hasSourceDir {
		return exitcode.InvalidArgsError("exactly one of --cursor or --source-dir is required")
	}

	replicaDir := replicaDirFlag
	if replicaDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return exitcode.UnexpectedErrorf("getting working directory: %v", err)
		}
		repoPath, err := resolve.StateRepoPath(cwd)
		if err != nil {
			return err
		}
		replicaDir = repoPath
	}

	ctx := cmd.Context()
	replicaJournal, err := openReadOnlyGitJournal(ctx, replicaDir, projectFlag, replicaFlag, remoteFlag, branchFlag)
	if err != nil {
		return exitcode.UnexpectedErrorf("open replica journal at %q: %v", replicaDir, err)
	}

	var target replication.Cursor
	if hasCursor {
		target, err = parseCursor(cursorFlag)
		if err != nil {
			return exitcode.InvalidArgsError(err.Error())
		}
	} else {
		sourceJournal, err := openReadOnlyGitJournal(ctx, sourceDirFlag, projectFlag, replicaFlag+"-source", "", "")
		if err != nil {
			return exitcode.UnexpectedErrorf("open source journal at %q: %v", sourceDirFlag, err)
		}
		head, _, err := sourceJournal.Head(ctx)
		if err != nil {
			return exitcode.UnexpectedErrorf("read source head: %v", err)
		}
		if head.IsZero() {
			return exitcode.NotFoundErrorf("source journal at %q has no events yet", sourceDirFlag)
		}
		target = head
	}

	result, waitErr := replication.Wait(ctx, replicaJournal, replicaFlag, target, replication.WaitOptions{Timeout: timeoutFlag, PollInterval: pollFlag})

	if err := printWaitResult(cmd, formatFlag, result, waitErr); err != nil {
		return exitcode.UnexpectedErrorf("writing output: %v", err)
	}

	if waitErr == nil {
		return nil
	}
	var timeoutErr *replication.BarrierTimeoutError
	if errors.As(waitErr, &timeoutErr) {
		return exitcode.ConflictErrorf("mirror barrier unsatisfied: %v", timeoutErr)
	}
	return exitcode.UnexpectedErrorf("mirror barrier: %v", waitErr)
}

// parseCursor accepts "epoch:sequence" (e.g. "1:42"), matching the same
// (authority_epoch, sequence) pair state-store/topology's Cursor vocabulary
// entry and the state-change JSON schema use.
func parseCursor(raw string) (replication.Cursor, error) {
	epochPart, seqPart, ok := strings.Cut(raw, ":")
	if !ok {
		return replication.Cursor{}, fmt.Errorf("--cursor must be formatted as epoch:sequence, got %q", raw)
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(epochPart), 10, 64)
	if err != nil {
		return replication.Cursor{}, fmt.Errorf("--cursor epoch %q is not a valid integer", epochPart)
	}
	sequence, err := strconv.ParseInt(strings.TrimSpace(seqPart), 10, 64)
	if err != nil {
		return replication.Cursor{}, fmt.Errorf("--cursor sequence %q is not a valid integer", seqPart)
	}
	cursor := replication.Cursor{Epoch: epoch, Sequence: sequence}
	if cursor.IsZero() {
		return replication.Cursor{}, fmt.Errorf("--cursor must not be 0:0")
	}
	return cursor, nil
}

func formatCursor(c replication.Cursor) string {
	return fmt.Sprintf("%d:%d", c.Epoch, c.Sequence)
}

// openReadOnlyGitJournal opens a Git-backed physical journal at dir through
// the same dalgo2ingitdb + DALJournal construction pkg/state/gitstore's
// Agent() wiring and pkg/state/replication's own physical tests use. This
// command only ever calls Head/After (via replication.Wait or a plain Head
// read) -- never Append/IngestReplica -- so the Role/AuthorityEpoch passed to
// NewDALJournal are harmless placeholders: role/epoch fencing exists to
// protect writes, and this command performs none. When remote/branch are
// both non-blank the journal is wrapped in *GitPushJournal so it implements
// replication.RemoteDurable, letting Wait prove Git remote durability rather
// than merely local apply
// (state-store/topology#ac:mirror-barrier-proves-git-durability).
func openReadOnlyGitJournal(ctx context.Context, dir, projectID, endpointID, remote, branch string) (replication.Journal, error) {
	database, err := dalgo2ingitdb.NewDatabase(dir, validator.NewCollectionsReader())
	if err != nil {
		return nil, fmt.Errorf("open Git DAL database: %w", err)
	}
	if err := replication.EnsureDALJournalSchema(ctx, database); err != nil {
		return nil, fmt.Errorf("ensure journal schema: %w", err)
	}
	zero := 0
	journal, err := replication.NewDALJournal(database, replication.DALJournalOptions{
		ProjectID: projectID, EndpointID: endpointID, Role: replication.RoleReplica, AuthorityEpoch: 1,
		MaxBatchItems: &zero, MaxBatchDelayMS: &zero,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Git journal: %w", err)
	}
	if remote == "" || branch == "" {
		return journal, nil
	}
	pushed, err := replication.NewGitPushJournal(journal, dir, remote, branch)
	if err != nil {
		return nil, err
	}
	return pushed, nil
}

type waitOutput struct {
	Replica      string `json:"replica"`
	Requested    string `json:"requested_cursor"`
	Observed     string `json:"observed_cursor"`
	Satisfied    bool   `json:"satisfied"`
	RemoteProven bool   `json:"remote_proven"`
	CommitSHA    string `json:"commit_sha,omitempty"`
	ElapsedMS    int64  `json:"elapsed_ms"`
	Error        string `json:"error,omitempty"`
}

func printWaitResult(cmd *cobra.Command, format string, result replication.BarrierResult, waitErr error) error {
	out := waitOutput{
		Replica: result.EndpointID, Requested: formatCursor(result.Requested), Observed: formatCursor(result.Observed),
		Satisfied: result.Satisfied, RemoteProven: result.RemoteProven, CommitSHA: result.CommitSHA,
		ElapsedMS: result.Elapsed.Milliseconds(),
	}
	if waitErr != nil {
		out.Error = waitErr.Error()
	}
	if format == "json" {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(out)
	}
	if out.Satisfied {
		durability := "locally applied"
		if out.RemoteProven {
			durability = "durably recorded on the Git remote at commit " + out.CommitSHA
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "mirror barrier satisfied: replica %s reached cursor %s (%s) in %s\n",
			out.Replica, out.Observed, durability, roundedDuration(result.Elapsed))
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "mirror barrier unsatisfied: replica %s requested %s, observed %s after %s\n",
		out.Replica, out.Requested, out.Observed, roundedDuration(result.Elapsed))
	return err
}

func roundedDuration(d time.Duration) time.Duration {
	return d.Round(time.Millisecond)
}
