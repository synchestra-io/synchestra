package task

// Features implemented: cli/task/new
// Features depended on:  state-store/task-store

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new task (in planning or queued)",
		Args:  cobra.NoArgs,
		RunE:  runNew,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("task", "", "task path using / as separator (required)")
	cmd.Flags().String("title", "", "human-readable title for the task (required)")
	cmd.Flags().String("description", "", "task description (optional)")
	cmd.Flags().String("depends-on", "", "comma-separated list of task paths this task depends on")
	cmd.Flags().Bool("enqueue", false, "create the task directly in queued status instead of planning")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runNew(cmd *cobra.Command, _ []string) error {
	taskFlag, _ := cmd.Flags().GetString("task")
	title, _ := cmd.Flags().GetString("title")
	description, _ := cmd.Flags().GetString("description")
	dependsOn, _ := cmd.Flags().GetString("depends-on")
	enqueue, _ := cmd.Flags().GetBool("enqueue")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(taskFlag) == "" {
		return exitcode.InvalidArgsError("--task is required")
	}
	if strings.TrimSpace(title) == "" {
		return exitcode.InvalidArgsError("--title is required")
	}

	store, err := resolveStore(syncFlag)
	if err != nil {
		return err
	}

	var deps []string
	if dependsOn != "" {
		for _, d := range strings.Split(dependsOn, ",") {
			if s := strings.TrimSpace(d); s != "" {
				deps = append(deps, s)
			}
		}
	}

	ctx := cmd.Context()
	params := state.TaskCreateParams{
		Slug:      taskFlag,
		Title:     title,
		DependsOn: deps,
	}
	_ = description // description is stored in the task file by the store if supported

	task, err := store.Task().Create(ctx, params)
	if err != nil {
		return mapStoreError(err)
	}

	if enqueue {
		if err := store.Task().Enqueue(ctx, taskFlag); err != nil {
			return mapStoreError(err)
		}
		task.Status = state.TaskStatusQueued
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created task %s (%s)\n", task.Slug, task.Status)
	return nil
}
