package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func handoffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Explicitly hand off a claim to a successor run",
		Long: "Handoff proves the outgoing run's current fence and moves the SAME claim to the " +
			"incoming run under a freshly minted fence in one audited event. The outgoing run's " +
			"old fence is refused by any later renew/release " +
			"(agent-coordination's \"Sequential cooperation is supported through explicit handoff\").",
		Args: cobra.NoArgs,
		RunE: runHandoff,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("claim", "", "claim ID to hand off (required)")
	addFenceFlags(cmd)
	cmd.Flags().String("to-run", "", "incoming run ID accepting ownership (required)")
	cmd.Flags().String("reason", "", "checkpoint summary / reason for the handoff (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runHandoff(cmd *cobra.Command, _ []string) error {
	claimID, _ := cmd.Flags().GetString("claim")
	toRun, _ := cmd.Flags().GetString("to-run")
	reason, _ := cmd.Flags().GetString("reason")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(claimID) == "" {
		return exitcode.InvalidArgsError("--claim is required")
	}
	if strings.TrimSpace(toRun) == "" {
		return exitcode.InvalidArgsError("--to-run is required")
	}
	fence, err := readFenceFlags(cmd)
	if err != nil {
		return err
	}

	store, _, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	// Handoff issues two sequential Appends (lease transfer, then a
	// worktree.handed_off follow-up event) — see claim.go's comment on why
	// this is not wrapped in state.CloseAfter.
	claim, err := store.Agent().Worktree().Handoff(cmd.Context(), claimID, state.WorktreeHandoffParams{
		Fence: fence, ToRunID: toRun, Reason: reason,
	})
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}
	return writeClaim(cmd.OutOrStdout(), format, claim)
}
