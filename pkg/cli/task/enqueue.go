package task

// Features implemented: cli/task/enqueue
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
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
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	if err := store.Task().Enqueue(cmd.Context(), taskFlag); err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s enqueued\n", taskFlag)
	return nil
}
