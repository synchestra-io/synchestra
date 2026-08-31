package cli

// Features implemented: cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"charm.land/fang/v2"
	"github.com/charmbracelet/x/term"
	"github.com/ingitdb/ingitdb-cli/cmd/ingitdb/commands"
	"github.com/ingitdb/ingitdb-cli/cmd/ingitdb/tui"
	"github.com/ingitdb/ingitdb-go/ingitdb/materializer"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
	"github.com/spf13/cobra"
	"github.com/strongo/buildinfo"
	"github.com/strongo/buildinfo/fangcmd"
	agentcmd "github.com/synchestra-io/synchestra/pkg/cli/agent"
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

// newRootCmd builds the synchestra root command and its full subcommand
// tree. Split out from Run purely so tests can walk the resulting command
// tree (e.g. resolving "self-update" and its "update" alias) without going
// through fang.Execute or touching os.Args.
func newRootCmd(
	osUserHomeDir func() (string, error),
	osGetwd func() (string, error),
) *cobra.Command {
	logf := func(args ...any) {
		_, _ = fmt.Fprintln(os.Stderr, args...)
	}
	viewBuilderLogf := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		logf(message)
	}
	viewBuilder := materializer.NewViewBuilder(materializer.NewFileRecordsReader(), viewBuilderLogf)
	isTerminal := func() bool {
		return term.IsTerminal(os.Stdout.Fd())
	}
	runConflictsTUI := func(ctx context.Context, files []string) error {
		width, height, sizeErr := term.GetSize(os.Stdout.Fd())
		if sizeErr != nil || width == 0 {
			width, height = 120, 40
		}
		return tui.RunConflicts(ctx, files, width, height)
	}

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

	info := buildinfo.Get("synchestra")

	rootCmd.AddCommand(
		commands.Pull(osUserHomeDir, osGetwd, validator.ReadDefinition, viewBuilder, logf, isTerminal, runConflictsTUI),
		commands.Setup(),
		commands.Resolve(osUserHomeDir, osGetwd, validator.ReadDefinition, logf, isTerminal, runConflictsTUI),
		synchinit.Command(),
		project.Command(),
		testcmd.Command(),
		feature.Command(),
		code.Command(),
		spec.Command(),
		runner.Command(runner.Dependencies{Getwd: osGetwd, UserHomeDir: osUserHomeDir}),
		taskcmd.Command(),
		agentcmd.Command(),
		selfupdatecmd.Command(info.Version),
		statecmd.Command(),
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

	info := buildinfo.Get("synchestra")
	fangOpts := fangcmd.Wire(rootCmd, info)

	rootCmd.SetArgs(args[1:])
	if err := fang.Execute(context.Background(), rootCmd, fangOpts...); err != nil {
		fatal(err)
	}
}
