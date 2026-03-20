package code

// Features implemented: cli/code
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/code

import (
	"github.com/spf13/cobra"
)

// Command returns the "code" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code",
		Short: "Query source code relationships to Synchestra resources",
		Long: `Commands for querying source code relationships to Synchestra resources.
Scans source files for synchestra: annotations and https://synchestra.io/ URLs
embedded in comments, showing the resources (features, plans, docs) that code
depends on. This complements synchestra feature commands which operate on the
specification graph.`,
	}
	cmd.AddCommand(
		depsCommand(),
	)
	return cmd
}
