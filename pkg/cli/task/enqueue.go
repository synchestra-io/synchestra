package task

// Features implemented: cli/task/enqueue
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func enqueueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enqueue",
		Short: "Move a task from planning to queued",
		Args:  cobra.NoArgs,
		RunE:  runEnqueue,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runEnqueue(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Enqueue(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task enqueue: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task enqueue is not yet implemented"}
}
