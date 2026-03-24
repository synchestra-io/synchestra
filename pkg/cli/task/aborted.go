package task

// Features implemented: cli/task/aborted
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func abortedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aborted",
		Short: "Report a task has been aborted (terminal)",
		Args:  cobra.NoArgs,
		RunE:  runAborted,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "what was done before aborting (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runAborted(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().ConfirmAbort(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task aborted: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task aborted is not yet implemented"}
}
