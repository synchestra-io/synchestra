package task

// Features implemented: cli/task/info
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func infoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show full task details and context",
		Args:  cobra.NoArgs,
		RunE:  runInfo,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runInfo(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Get(ctx, slug)
	// Format output per --format flag using writeTask()
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task info: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task info is not yet implemented"}
}
