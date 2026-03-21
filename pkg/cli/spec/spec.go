package spec

// Features implemented: cli/spec

import (
	"github.com/spf13/cobra"
)

// Command returns the "spec" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Validate and search specification repositories",
	}
	cmd.AddCommand(
		lintCommand(),
	)
	return cmd
}
