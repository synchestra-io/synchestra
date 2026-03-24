package state

// Features implemented: cli/state

import "github.com/spf13/cobra"

// Command returns the "state" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "State repository synchronization -- pull, push, sync",
	}
	cmd.AddCommand(
		pullCommand(),
		pushCommand(),
		syncCommand(),
	)
	return cmd
}
