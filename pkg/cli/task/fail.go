package task

// Features implemented: cli/task/fail
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func failCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fail",
		Short: "Mark a task as failed with reason",
		Args:  cobra.NoArgs,
		RunE:  runFail,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("reason", "", "why the task failed (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runFail(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")
	reason, _ := cmd.Flags().GetString("reason")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}
	if strings.TrimSpace(reason) == "" {
		return &exitError{code: 2, msg: "--reason is required"}
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	if err := store.Task().Fail(cmd.Context(), taskFlag, reason); err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s failed\n", taskFlag)
	return nil
}
