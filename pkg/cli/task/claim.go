package task

// Features implemented: cli/task/claim
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func claimCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Claim a queued task for work",
		Args:  cobra.NoArgs,
		RunE:  runClaim,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("run", "", "unique identifier for this agent run (required)")
	cmd.Flags().String("model", "", "model being used, e.g. haiku, sonnet, opus (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runClaim(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")
	run, _ := cmd.Flags().GetString("run")
	model, _ := cmd.Flags().GetString("model")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}
	if strings.TrimSpace(run) == "" {
		return exitcode.InvalidArgsError("--run is required")
	}
	if strings.TrimSpace(model) == "" {
		return exitcode.InvalidArgsError("--model is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	if err := store.Task().Claim(cmd.Context(), taskFlag, state.ClaimParams{Run: run, Model: model}); err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s claimed by %s (%s)\n", taskFlag, run, model)
	return nil
}
