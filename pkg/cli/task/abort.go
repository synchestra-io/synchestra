package task

// Features implemented: cli/task/abort
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func abortCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "abort",
		Short: "Request abortion of a task (sets abort_requested flag)",
		Args:  cobra.NoArgs,
		RunE:  runAbort,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "why the abort is being requested (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runAbort(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().RequestAbort(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task abort: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task abort is not yet implemented"}
}
