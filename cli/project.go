package cli

// Features implemented: cli/project
// Features depended on:  global-config

import (
	"github.com/spf13/cobra"
)

// ProjectCommand returns the root "project" command group.
func ProjectCommand(osUserHomeDir func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create and manage Synchestra projects",
		Long:  "Commands for creating and managing Synchestra projects — setting up spec, state, and target repositories, viewing project configuration, and modifying project settings.",
	}

	cmd.AddCommand(
		ProjectNewCommand(osUserHomeDir),
	)

	return cmd
}
