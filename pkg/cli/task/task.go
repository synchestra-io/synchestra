package task

// Features implemented: cli/task

import "github.com/spf13/cobra"

// Command returns the "task" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task lifecycle management — create, claim, start, complete, and more",
	}
	cmd.AddCommand(
		newCommand(),
		enqueueCommand(),
		claimCommand(),
		startCommand(),
		statusCommand(),
		completeCommand(),
		failCommand(),
		blockCommand(),
		unblockCommand(),
		releaseCommand(),
		abortCommand(),
		abortedCommand(),
		listCommand(),
		infoCommand(),
	)
	return cmd
}
