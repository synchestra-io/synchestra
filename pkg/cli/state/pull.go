package state

// Features implemented: cli/state/pull
// Features depended on:  state-store

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
)

func pullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest state from origin to local main",
		Args:  cobra.NoArgs,
		RunE:  runPull,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runPull(cmd *cobra.Command, _ []string) error {
	// TODO: Resolve project, construct store, call store.State().Pull(ctx)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "state pull: not implemented yet")
	return exitcode.UnexpectedError("synchestra state pull is not yet implemented")
}
