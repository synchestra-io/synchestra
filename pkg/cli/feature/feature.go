package feature

// Features implemented: cli/feature

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/feature"
)

// Command returns the "feature" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Query features — listing, hierarchy, dependencies, references",
	}
	cmd.AddCommand(
		infoCommand(),
		listCommand(),
		treeCommand(),
		depsCommand(),
		refsCommand(),
		newCommand(),
	)
	return cmd
}

// resolveFeaturesDir returns the absolute path to the features directory.
// It finds the spec repo root from CWD, then appends spec/features/.
func resolveFeaturesDir(projectFlag string) (string, error) {
	if projectFlag != "" {
		return "", exitcode.InvalidArgsError("--project with project lookup is not yet implemented; run from within a spec repo directory")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", exitcode.UnexpectedErrorf("cannot determine working directory: %v", err)
	}

	root, err := feature.FindSpecRepoRoot(cwd)
	if err != nil {
		return "", err
	}

	featDir := filepath.Join(root, feature.DefaultSpecDir, feature.FeaturesSubDir)
	info, err := os.Stat(featDir)
	if err != nil || !info.IsDir() {
		return "", exitcode.NotFoundErrorf("features directory not found: %s", featDir)
	}

	return featDir, nil
}
