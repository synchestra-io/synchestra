package cli

// Features implemented: cli/project/new

import (
	"context"
	"errors"
	"os"

	"charm.land/fang/v2"
	"github.com/ingitdb/ingitdb-cli/cmd/ingitdb/commands"
	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/cmd/project"
	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Run(
	args []string,
	osUserHomeDir func() (string, error),
	osGetwd func() (string, error),
	fatal func(error),
	logf func(...any),
	exit func(int),
) {
	_ = osGetwd
	_ = logf
	rootCmd := &cobra.Command{
		Use:           "synchestra",
		Short:         "Synchestra CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return errors.New("not implemented yet")
		},
	}
	rootCmd.Flags().String("path", "", "path to the database directory (default: current directory)")
	rootCmd.SetErr(os.Stderr)

	rootCmd.AddCommand(
		commands.Version(version, commit, date),
		commands.Pull(),
		commands.Setup(),
		commands.Resolve(),
		commands.Watch(),
		commands.Find(),
		commands.Migrate(),
		project.GroupCommand(osUserHomeDir, gitops.NewRunner()),
	)

	rootCmd.SetArgs(args[1:])
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		var exitErr *exitcode.Error
		if errors.As(err, &exitErr) {
			fatal(exitErr.Err)
			exit(exitErr.Code)
			return
		}
		fatal(err)
		exit(1)
	}
}
