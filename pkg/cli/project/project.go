package project

// Features implemented: cli/project

import (
	"github.com/spf13/cobra"
)

// Command returns the "project" command group. The group currently has
// no implemented subcommands — `init` and `new` were removed in favour
// of the top-level `synchestra init` (cli/init Feature). The group is
// retained as a namespace placeholder for future commands (`info`,
// `set`, `code add`, `code remove`) once they are implemented against
// the unified specscore.yaml + synchestra.yaml model.
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "project",
		Short: "Project queries and management (subcommands TBD)",
		Long: `The "project" command group reserves its namespace for future
subcommands that operate on a Synchestra-managed project's metadata.

Bootstrapping a project is now done by the top-level "synchestra init"
command, NOT "synchestra project init" (which has been removed).

Planned subcommands (not yet implemented):
  info               Display project configuration
  set                Update project settings
  code add/remove    Manage code repositories

See spec/features/cli/project/ for the design.`,
	}
}
