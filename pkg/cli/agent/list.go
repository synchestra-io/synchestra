package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git

import (
	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List worktree claims with optional filtering",
		Args:  cobra.NoArgs,
		RunE:  runList,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("repository", "", "filter by canonical repository identifier")
	cmd.Flags().String("run", "", "filter by run ID")
	cmd.Flags().Bool("active-only", false, "exclude released claims")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runList(cmd *cobra.Command, _ []string) error {
	repository, _ := cmd.Flags().GetString("repository")
	run, _ := cmd.Flags().GetString("run")
	activeOnly, _ := cmd.Flags().GetBool("active-only")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	store, projectID, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	claims, err := store.Agent().Worktree().List(cmd.Context(), state.WorktreeFilter{
		ProjectID: projectID, RepositoryID: repository, RunID: run, ActiveOnly: activeOnly,
	})
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}
	return writeClaimList(cmd.OutOrStdout(), format, claims)
}
