package project

// Features implemented: cli/project

import (
	"github.com/spf13/cobra"
)

// Command returns the "project" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project creation and management",
	}
	cmd.AddCommand(newCommand())
	return cmd
}
