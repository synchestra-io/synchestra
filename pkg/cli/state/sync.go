package state

// Features implemented: cli/state/sync
// Features depended on:  state-store

import (
	"fmt"

	"github.com/spf13/cobra"
)

func syncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Full bidirectional sync -- pull then push",
		Args:  cobra.NoArgs,
		RunE:  runSync,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runSync(cmd *cobra.Command, _ []string) error {
	// TODO: Resolve project, construct store, call store.State().Sync(ctx)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "state sync: not implemented yet")
	return &exitError{code: 10, msg: "synchestra state sync is not yet implemented"}
}
