package task

// Features implemented: cli/task/block
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func blockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Mark a task as blocked with reason",
		Args:  cobra.NoArgs,
		RunE:  runBlock,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "what is blocking the task (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runBlock(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")
	reason, _ := cmd.Flags().GetString("reason")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}
	if strings.TrimSpace(reason) == "" {
		return &exitError{code: 2, msg: "--reason is required"}
	}

	// TODO: Resolve project, construct store, call store.Task().Block(ctx, slug, reason)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task block: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task block is not yet implemented"}
}
