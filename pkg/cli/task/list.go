package task

// Features implemented: cli/task/list
// Features depended on:  state-store/task-store

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks with optional filtering",
		Args:  cobra.NoArgs,
		RunE:  runList,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("status", "", "filter by task status (e.g. planning, queued, claimed, in_progress, completed, failed, blocked, aborted)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json, md, csv")
	cmd.Flags().String("fields", "", "comma-separated list of fields to include (e.g. path,status,model)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	statusFlag, _ := cmd.Flags().GetString("status")
	format, _ := cmd.Flags().GetString("format")
	syncFlag, _ := cmd.Flags().GetString("sync")

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	var filter state.TaskFilter
	if s := strings.TrimSpace(statusFlag); s != "" {
		ts := state.TaskStatus(s)
		filter.Status = &ts
	}

	tasks, err := store.Task().List(cmd.Context(), filter)
	if err != nil {
		return mapStoreError(err)
	}

	return writeTaskList(cmd.OutOrStdout(), format, tasks)
}
