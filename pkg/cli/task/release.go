package task

// Features implemented: cli/task/release
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func releaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release a claimed task back to queued",
		Args:  cobra.NoArgs,
		RunE:  runRelease,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "why the task is being released (optional)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runRelease(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Release(ctx, slug)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task release: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task release is not yet implemented"}
}
