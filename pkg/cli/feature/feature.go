package feature

// Features implemented: cli/feature

import (
	"github.com/spf13/cobra"
)

// Command returns the "feature" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Query features — listing, hierarchy, dependencies, references",
	}
	cmd.AddCommand(
		listCommand(),
		treeCommand(),
		depsCommand(),
		refsCommand(),
	)
	return cmd
}
