package task

// Features implemented: cli/task/unblock
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
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
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	if err := store.Task().Unblock(cmd.Context(), taskFlag); err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s unblocked\n", taskFlag)
	return nil
}
