package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

func renewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "renew",
		Short: "Renew a claim's lease without changing ownership",
		Args:  cobra.NoArgs,
		RunE:  runRenew,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("claim", "", "claim ID to renew (required)")
	addFenceFlags(cmd)
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runRenew(cmd *cobra.Command, _ []string) error {
	claimID, _ := cmd.Flags().GetString("claim")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(claimID) == "" {
		return exitcode.InvalidArgsError("--claim is required")
	}
	fence, err := readFenceFlags(cmd)
	if err != nil {
		return err
	}

	store, _, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	// Renew issues two sequential Appends (lease renew, then a worktree
	// follow-up event) — see claim.go's comment on why this is not wrapped
	// in state.CloseAfter.
	claim, err := store.Agent().Worktree().Renew(cmd.Context(), claimID, fence)
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}
	return writeClaim(cmd.OutOrStdout(), format, claim)
}
