package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git, state-store/journal-batching

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func claimCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Claim exclusive write ownership of a repository/branch/worktree",
		Long: "Claim binds one run to one repository/branch/worktree under a fencing lease " +
			"(agent-coordination#ac:one-writer-claim-is-fenced). If the current holder's lease " +
			"has expired without an explicit release, Claim reclaims it for the caller instead " +
			"of conflicting (agent-coordination#ac:abandoned-run-is-resumable).",
		Args: cobra.NoArgs,
		RunE: runClaim,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("repository", "", "canonical repository identifier (required)")
	cmd.Flags().String("run", "", "unique identifier for this agent run (required)")
	cmd.Flags().String("worktree", "", "local worktree path")
	cmd.Flags().String("branch", "", "local branch this run writes to (required)")
	cmd.Flags().String("target-ref", "", "target/base ref, e.g. main")
	cmd.Flags().String("base-sha", "", "observed base SHA")
	cmd.Flags().String("head-sha", "", "observed head SHA")
	cmd.Flags().StringSlice("scope", nil, "declared scope areas: repository paths, module IDs, or SpecScore feature references")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runClaim(cmd *cobra.Command, _ []string) error {
	repository, _ := cmd.Flags().GetString("repository")
	run, _ := cmd.Flags().GetString("run")
	worktree, _ := cmd.Flags().GetString("worktree")
	branch, _ := cmd.Flags().GetString("branch")
	targetRef, _ := cmd.Flags().GetString("target-ref")
	baseSHA, _ := cmd.Flags().GetString("base-sha")
	headSHA, _ := cmd.Flags().GetString("head-sha")
	scope, _ := cmd.Flags().GetStringSlice("scope")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(repository) == "" {
		return exitcode.InvalidArgsError("--repository is required")
	}
	if strings.TrimSpace(run) == "" {
		return exitcode.InvalidArgsError("--run is required")
	}
	if strings.TrimSpace(branch) == "" {
		return exitcode.InvalidArgsError("--branch is required")
	}

	store, projectID, err := resolveStore(cmd, project, syncFlag, run)
	if err != nil {
		return err
	}
	// Claim issues two sequential journal Appends (a lease acquire, then a
	// worktree.claimed/reclaimed follow-up — see pkg/state/agentstore/
	// worktree.go and README.md's Open Questions), so this call is not
	// wrapped in state.CloseAfter: CloseAfter's contract requires at most
	// one Append per call (see its doc comment) — racing it here could
	// permanently fail the second Append with ErrJournalClosed instead of
	// merely delaying it. Close is still called before returning, purely as
	// shutdown hygiene (nothing is pending at this point either way).
	claim, err := store.Agent().Worktree().Claim(cmd.Context(), state.WorktreeClaimParams{
		ProjectID: projectID, RepositoryID: repository, RunID: run,
		WorktreePath: worktree, Branch: branch, TargetRef: targetRef,
		BaseSHA: baseSHA, HeadSHA: headSHA, ScopeAreas: scope,
	})
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}

	return writeClaim(cmd.OutOrStdout(), format, claim)
}
