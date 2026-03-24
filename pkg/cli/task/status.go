package task

// Features implemented: cli/task/status
// Features depended on:  state-store/task-store

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/state"
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
	reason, _ := cmd.Flags().GetString("reason")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return &exitError{code: 2, msg: "--task is required"}
	}

	hasCurrent := strings.TrimSpace(current) != ""
	hasNew := strings.TrimSpace(newStatus) != ""
	if hasCurrent != hasNew {
		return &exitError{code: 2, msg: "--current and --new must both be provided for update mode"}
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Query mode: just print the status
	if !hasNew {
		t, err := store.Task().Get(ctx, taskFlag)
		if err != nil {
			return mapStoreError(err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", t.Status)
		return nil
	}

	// Update mode: guard with --current, then perform transition
	t, err := store.Task().Get(ctx, taskFlag)
	if err != nil {
		return mapStoreError(err)
	}
	if string(t.Status) != current {
		return &exitError{code: 4, msg: fmt.Sprintf("status guard failed: expected %s, got %s", current, t.Status)}
	}

	target := state.TaskStatus(newStatus)
	if err := applyTransition(store, ctx, taskFlag, target, reason); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "task %s transitioned from %s to %s\n", taskFlag, current, newStatus)
	return nil
}

// applyTransition calls the appropriate TaskStore method for the target status.
func applyTransition(store state.Store, ctx context.Context, slug string, target state.TaskStatus, reason string) error {
	ts := store.Task()
	var err error
	switch target {
	case state.TaskStatusQueued:
		err = ts.Enqueue(ctx, slug)
	case state.TaskStatusClaimed:
		return &exitError{code: 2, msg: "use 'synchestra task claim' for claiming (requires --run and --model)"}
	case state.TaskStatusInProgress:
		err = ts.Start(ctx, slug)
	case state.TaskStatusCompleted:
		err = ts.Complete(ctx, slug, reason)
	case state.TaskStatusFailed:
		if reason == "" {
			return &exitError{code: 2, msg: "--reason is required when transitioning to failed"}
		}
		err = ts.Fail(ctx, slug, reason)
	case state.TaskStatusBlocked:
		if reason == "" {
			return &exitError{code: 2, msg: "--reason is required when transitioning to blocked"}
		}
		err = ts.Block(ctx, slug, reason)
	case state.TaskStatusAborted:
		err = ts.ConfirmAbort(ctx, slug)
	default:
		return &exitError{code: 2, msg: fmt.Sprintf("unsupported target status %q for status command", target)}
	}
	if err != nil {
		return mapStoreError(err)
	}
	return nil
}
