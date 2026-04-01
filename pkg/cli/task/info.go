package task

// Features implemented: cli/task/info
// Features depended on:  state-store/task-store

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
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
	format, _ := cmd.Flags().GetString("format")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	t, err := store.Task().Get(cmd.Context(), taskFlag)
	if err != nil {
		return mapStoreError(err)
	}

	return writeTask(cmd.OutOrStdout(), format, t)
}
