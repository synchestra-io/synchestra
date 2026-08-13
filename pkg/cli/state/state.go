package state

// Features implemented: cli/state, cli/state/wait

import "github.com/spf13/cobra"

// Command returns the "state" command group. Only real, working subcommands
// are registered here -- see main_test.go's
// TestRootCmdAdvertisesStateWaitButNotTheWithdrawnSyncStubs. pull/push/sync
// remain deliberately UNREGISTERED: they were withdrawn from the root
// command tree (see "fix(state): harden physical journal boundaries") while
// still stubs that print "not implemented yet" and fail with exit code 10.
// Their command constructors and tests stay in this package so they can be
// wired in the same way `wait` was the moment they have a real
// implementation behind them -- but advertising a broken command to users in
// the meantime is worse than the command not existing at all.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "State topology operations -- mirror barriers and (soon) repository synchronization",
	}
	cmd.AddCommand(
		waitCommand(),
	)
	return cmd
}
