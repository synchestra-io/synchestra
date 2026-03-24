package task

// Features implemented: cli/task/unblock
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func unblockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unblock",
		Short: "Resume a blocked task (blocked -> in_progress)",
		Args:  cobra.NoArgs,
		RunE:  runUnblock,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "what resolved the blocker (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runUnblock(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Unblock(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task unblock: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task unblock is not yet implemented"}
}
