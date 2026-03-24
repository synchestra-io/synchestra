package task

// Features implemented: cli/task/claim
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}
	if strings.TrimSpace(run) == "" {
		return &exitError{code: 2, msg: "--run is required"}
	}
	if strings.TrimSpace(model) == "" {
		return &exitError{code: 2, msg: "--model is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Claim(ctx, slug, ClaimParams{Run, Model})
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task claim: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task claim is not yet implemented"}
}
