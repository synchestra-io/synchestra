package task

// Features implemented: cli/task/block
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
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
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}
	if strings.TrimSpace(reason) == "" {
		return exitcode.InvalidArgsError("--reason is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	if err := store.Task().Block(cmd.Context(), taskFlag, reason); err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s blocked\n", taskFlag)
	return nil
}
