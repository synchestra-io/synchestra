// Package project implements the `synchestra project` command group.
package project

import "github.com/spf13/cobra"

// GroupCommand returns the `project` command group.
func GroupCommand(homeDir func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create and manage Synchestra projects",
		Long: `Commands for creating and managing Synchestra projects — setting up spec,
state, and target repositories, viewing project configuration, and
modifying project settings.`,
	}
	return cmd
}
