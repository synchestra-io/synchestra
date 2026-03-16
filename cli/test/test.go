package test

// Features implemented: cli

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run and manage test scenarios",
	}
	cmd.AddCommand(
		runCommand(),
		listCommand(),
	)
	return cmd
}
