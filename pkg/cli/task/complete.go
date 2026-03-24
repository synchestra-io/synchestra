package task

// Features implemented: cli/task/complete
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func completeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete",
		Short: "Mark a task as completed",
		Args:  cobra.NoArgs,
		RunE:  runComplete,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("summary", "", "completion summary (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runComplete(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Complete(ctx, slug, summary)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task complete: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task complete is not yet implemented"}
}
