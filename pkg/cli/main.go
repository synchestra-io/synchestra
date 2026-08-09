package cli

// Features implemented: cli

import (
	"context"
	"errors"
	"os"

	"charm.land/fang/v2"
	"github.com/ingitdb/ingitdb-cli/cmd/ingitdb/commands"
	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/code"
	"github.com/synchestra-io/synchestra/pkg/cli/feature"
	"github.com/synchestra-io/synchestra/pkg/cli/project"
	"github.com/synchestra-io/synchestra/pkg/cli/runner"
	selfupdatecmd "github.com/synchestra-io/synchestra/pkg/cli/selfupdate"
	"github.com/synchestra-io/synchestra/pkg/cli/spec"
	statecmd "github.com/synchestra-io/synchestra/pkg/cli/state"
	"github.com/synchestra-io/synchestra/pkg/cli/synchinit"
	taskcmd "github.com/synchestra-io/synchestra/pkg/cli/task"
	testcmd "github.com/synchestra-io/synchestra/pkg/cli/test"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// newRootCmd builds the synchestra root command and its full subcommand
// tree. Split out from Run purely so tests can walk the resulting command
// tree (e.g. resolving "self-update" and its "update" alias) without going
// through fang.Execute or touching os.Args.
func newRootCmd(
	osUserHomeDir func() (string, error),
	osGetwd func() (string, error),
) *cobra.Command {
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
		synchinit.Command(),
		project.Command(),
		testcmd.Command(),
		feature.Command(),
		code.Command(),
		spec.Command(),
		runner.Command(runner.Dependencies{Getwd: osGetwd, UserHomeDir: osUserHomeDir}),
		statecmd.Command(),
		taskcmd.Command(),
		selfupdatecmd.Command(version),
	)

	return rootCmd
}

func Run(
	args []string,
	osUserHomeDir func() (string, error),
	osGetwd func() (string, error),
	fatal func(error),
	logf func(...any),
) {
	_ = logf
	rootCmd := newRootCmd(osUserHomeDir, osGetwd)

	rootCmd.SetArgs(args[1:])
	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version), fang.WithCommit(commit)); err != nil {
		fatal(err)
	}
}
