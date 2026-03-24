package task

// Features implemented: cli/task/start
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func startCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Begin work on a claimed task (claimed -> in_progress)",
		Args:  cobra.NoArgs,
		RunE:  runStart,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runStart(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Start(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task start: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task start is not yet implemented"}
}
