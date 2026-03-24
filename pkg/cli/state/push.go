package state

// Features implemented: cli/state/push
// Features depended on:  state-store

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local state to origin",
		Args:  cobra.NoArgs,
		RunE:  runPush,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runPush(cmd *cobra.Command, _ []string) error {
	// TODO: Resolve project, construct store, call store.State().Push(ctx)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "state push: not implemented yet")
	return &exitError{code: 10, msg: "synchestra state push is not yet implemented"}
}
