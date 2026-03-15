// Package project implements the `synchestra project` command group.
package project

// Features implemented: cli/project/new

import (
	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
)

// GroupCommand returns the `project` command group with all subcommands registered.
func GroupCommand(homeDir func() (string, error), git gitops.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Create and manage Synchestra projects",
		Long: `Commands for creating and managing Synchestra projects — setting up spec,
state, and target repositories, viewing project configuration, and
modifying project settings.`,
	}
	cmd.AddCommand(NewCommand(homeDir, git))
	return cmd
}
