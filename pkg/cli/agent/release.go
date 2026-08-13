package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

func releaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Explicitly release a claim",
		Args:  cobra.NoArgs,
		RunE:  runRelease,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("claim", "", "claim ID to release (required)")
	addFenceFlags(cmd)
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runRelease(cmd *cobra.Command, _ []string) error {
	claimID, _ := cmd.Flags().GetString("claim")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")

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
	// Release issues two sequential Appends (lease release, then a worktree
	// follow-up event) — see claim.go's comment on why this is not wrapped
	// in state.CloseAfter.
	err = store.Agent().Worktree().Release(cmd.Context(), claimID, fence)
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "claim %s released\n", claimID)
	return nil
}
