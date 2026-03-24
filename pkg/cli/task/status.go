package task

// Features implemented: cli/task/status
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func statusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Query or update task status",
		Long: `Query mode (no --current/--new): pulls latest state and prints the task's
current status including the abort_requested flag.

Update mode (with --current and --new): transitions the task from one status
to another. The --current parameter acts as a guard — the command fails if
the task's actual status does not match --current.`,
		Args: cobra.NoArgs,
		RunE: runStatus,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("current", "", "expected current status (guard for update mode)")
	cmd.Flags().String("new", "", "target status to transition to (update mode)")
	cmd.Flags().String("reason", "", "reason for the transition (required for failed and blocked)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")
	current, _ := cmd.Flags().GetString("current")
	newStatus, _ := cmd.Flags().GetString("new")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	// Validate update mode: --current and --new must both be provided or both omitted
	hasCurrent := strings.TrimSpace(current) != ""
	hasNew := strings.TrimSpace(newStatus) != ""
	if hasCurrent != hasNew {
		return &exitError{code: 2, msg: "--current and --new must both be provided for update mode"}
	}

	// TODO: Resolve project, construct store
	// Query mode: call store.Task().Get(ctx, slug) and print status
	// Update mode: validate transition, call appropriate TaskStore method
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "task status: not implemented yet")
	return &exitError{code: 10, msg: "synchestra task status is not yet implemented"}
}
