package task

// Features implemented: cli/task/list
// Features depended on:  state-store/task-store

import (
	"fmt"

	"github.com/spf13/cobra"
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
	// TODO: Resolve project, construct store
	// Parse --status into *state.TaskStatus filter
	// Call store.Task().List(ctx, filter)
	// Format output per --format flag
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task list: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task list is not yet implemented"}
}
