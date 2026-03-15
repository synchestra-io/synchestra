package project

// Features implemented: cli/project/new
// Features depended on:  global-config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/internal/exitcode"
	"github.com/synchesta-io/synchestra/cli/internal/gitops"
	"github.com/synchesta-io/synchestra/cli/internal/globalconfig"
	"github.com/synchesta-io/synchestra/cli/internal/reporef"
	"gopkg.in/yaml.v3"
)

// NewCommand returns the `project new` cobra command.
func NewCommand(homeDir func() (string, error), git gitops.Runner) *cobra.Command {
	var (
		specRepoRef   string
		stateRepoRef  string
		targetRepoRefs []string
		title         string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Synchestra project",
		Long: `Creates a new Synchestra project by linking a spec repo, state repo, and
one or more target repos. Resolves all repo references, clones any that
are not already on disk, writes config files to each, commits and pushes.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProjectNew(cmd, homeDir, git, specRepoRef, stateRepoRef, targetRepoRefs, title)
		},
	}

	cmd.Flags().StringVar(&specRepoRef, "spec-repo", "", "Spec repository reference (required)")
	cmd.Flags().StringVar(&stateRepoRef, "state-repo", "", "State repository reference (required)")
	cmd.Flags().StringArrayVar(&targetRepoRefs, "target-repo", nil, "Target repository reference (repeatable, at least one required)")
	cmd.Flags().StringVar(&title, "title", "", "Project title (derived from README.md or repo name if omitted)")

	// Note: cobra's MarkFlagRequired errors exit with code 1 (not 2) because they return
	// a generic error, not an exitcode.Error. --target-repo is validated manually in RunE
	// to correctly emit exit code 2. --spec-repo and --state-repo missing-flag errors
	// take the cobra path and exit 1.
	_ = cmd.MarkFlagRequired("spec-repo")
	_ = cmd.MarkFlagRequired("state-repo")

	return cmd
}

func runProjectNew(
	cmd *cobra.Command,
	homeDirFn func() (string, error),
	git gitops.Runner,
	specRepoRef, stateRepoRef string,
	targetRepoRefs []string,
	title string,
) error {
	if len(targetRepoRefs) == 0 {
		return exitcode.New(2, "--target-repo is required (at least one)")
	}

	homeDir, err := homeDirFn()
	if err != nil {
		return exitcode.New(10, "get home directory: %v", err)
	}

	cfg, err := globalconfig.Load(homeDir)
	if err != nil {
		return exitcode.New(10, "load global config: %v", err)
	}

	// Parse all repo references
	specRef, err := reporef.Parse(specRepoRef)
	if err != nil {
		return exitcode.New(2, "invalid --spec-repo: %v", err)
	}
	stateRef, err := reporef.Parse(stateRepoRef)
	if err != nil {
		return exitcode.New(2, "invalid --state-repo: %v", err)
	}

	targetRefs := make([]reporef.Ref, len(targetRepoRefs))
	for i, tr := range targetRepoRefs {
		targetRefs[i], err = reporef.Parse(tr)
		if err != nil {
			return exitcode.New(2, "invalid --target-repo %q: %v", tr, err)
		}
	}

	// Collect all refs: spec, state, targets
	allRefs := append([]reporef.Ref{specRef, stateRef}, targetRefs...)

	// Resolve local paths and clone if needed
	for _, ref := range allRefs {
		local := ref.LocalPath(cfg.ReposDir)
		origin := ref.OriginURL()

		exists, err := git.IsRepo(local)
		if err != nil {
			return exitcode.New(10, "check repo %s: %v", local, err)
		}
		if !exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "cloning %s...\n", origin)
			if err := git.Clone(origin, local); err != nil {
				return exitcode.New(3, "clone %s: %v", origin, err)
			}
			isRepo, err := git.IsRepo(local)
			if err != nil || !isRepo {
				return exitcode.New(3, "clone of %s did not produce a git repo", origin)
			}
		}
	}

	specLocal := specRef.LocalPath(cfg.ReposDir)
	stateLocal := stateRef.LocalPath(cfg.ReposDir)
	specOrigin := specRef.OriginURL()

	// Get authoritative origin URLs from git remote before conflict checks.
	stateOrigin, err := git.OriginURL(stateLocal)
	if err != nil {
		return exitcode.New(10, "get origin URL for state repo: %v", err)
	}

	// Check for conflicts: existing config files pointing to a different project.
	// This applies to all repos, including the spec repo itself.
	if err := checkNoConflict(specLocal, "synchestra-spec.yaml", "state_repo", stateOrigin); err != nil {
		return err
	}
	if err := checkNoConflict(stateLocal, "synchestra-state.yaml", "spec_repo", specOrigin); err != nil {
		return err
	}
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		if err := checkNoConflict(targetLocal, "synchestra-target.yaml", "spec_repo", specOrigin); err != nil {
			return err
		}
	}

	// Derive title
	if title == "" {
		title = deriveTitle(specLocal, specRef.Repo)
	}

	targetOrigins := make([]string, len(targetRefs))
	for i, tr := range targetRefs {
		targetOrigins[i], err = git.OriginURL(tr.LocalPath(cfg.ReposDir))
		if err != nil {
			return exitcode.New(10, "get origin URL for target repo %s: %v", tr.Repo, err)
		}
	}

	// Write synchestra-spec.yaml
	specConfig := map[string]any{
		"title":      title,
		"state_repo": stateOrigin,
		"repos":      targetOrigins,
	}
	if err := writeYAML(filepath.Join(specLocal, "synchestra-spec.yaml"), specConfig); err != nil {
		return exitcode.New(10, "write synchestra-spec.yaml: %v", err)
	}

	// Write synchestra-state.yaml
	stateConfig := map[string]any{"spec_repo": specOrigin}
	if err := writeYAML(filepath.Join(stateLocal, "synchestra-state.yaml"), stateConfig); err != nil {
		return exitcode.New(10, "write synchestra-state.yaml: %v", err)
	}

	// Write synchestra-target.yaml to each target
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		targetConfig := map[string]any{"spec_repo": specOrigin}
		if err := writeYAML(filepath.Join(targetLocal, "synchestra-target.yaml"), targetConfig); err != nil {
			return exitcode.New(10, "write synchestra-target.yaml to %s: %v", tr.Repo, err)
		}
	}

	// Commit and push all repos (with pull-retry on push conflict)
	commitMsg := fmt.Sprintf("chore: initialize Synchestra project %q", title)

	if err := commitPushWithRetry(git, specLocal, []string{"synchestra-spec.yaml"}, commitMsg, func() error {
		return checkNoConflict(specLocal, "synchestra-spec.yaml", "state_repo", stateOrigin)
	}); err != nil {
		return exitcode.New(10, "commit spec repo: %v", err)
	}
	if err := commitPushWithRetry(git, stateLocal, []string{"synchestra-state.yaml"}, commitMsg, func() error {
		return checkNoConflict(stateLocal, "synchestra-state.yaml", "spec_repo", specOrigin)
	}); err != nil {
		return exitcode.New(10, "commit state repo: %v", err)
	}
	for _, tr := range targetRefs {
		targetLocal := tr.LocalPath(cfg.ReposDir)
		if err := commitPushWithRetry(git, targetLocal, []string{"synchestra-target.yaml"}, commitMsg, func() error {
			return checkNoConflict(targetLocal, "synchestra-target.yaml", "spec_repo", specOrigin)
		}); err != nil {
			return exitcode.New(10, "commit target repo %s: %v", tr.Repo, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Project %q created.\n", title)
	return nil
}

// commitPushWithRetry runs CommitAndPush; on push failure it pulls, re-checks for
// conflicts, then retries with Push only. This implements the spec's "on push conflict:
// pull, re-check, retry or fail" requirement.
func commitPushWithRetry(git gitops.Runner, dir string, files []string, msg string, conflictCheck func() error) error {
	commitErr := git.CommitAndPush(dir, files, msg)
	if commitErr == nil {
		return nil
	}
	// Push may have failed — pull to get remote changes and retry
	if pullErr := git.Pull(dir); pullErr != nil {
		return fmt.Errorf("pull after push failure (original: %v): %w", commitErr, pullErr)
	}
	// Re-check: if a concurrent writer set a conflicting config, return exit code 1
	if checkErr := conflictCheck(); checkErr != nil {
		return checkErr
	}
	// Retry push (commit already recorded locally)
	return git.Push(dir)
}

// checkNoConflict returns an exitcode.Error (code 1) if the given config file
// exists and its field does not equal expectedValue.
func checkNoConflict(dir, filename, field, expectedValue string) error {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return exitcode.New(10, "read %s: %v", path, err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return exitcode.New(10, "parse %s: %v", path, err)
	}
	existing, ok := m[field]
	if !ok {
		return nil
	}
	existingStr, isStr := existing.(string)
	if isStr && existingStr != expectedValue {
		return exitcode.New(1, "%s already configured for a different project (%s: %q)", filename, field, existingStr)
	}
	return nil
}

// deriveTitle extracts the first `# Heading` from README.md, or falls back to repoName.
func deriveTitle(repoDir, repoName string) string {
	data, err := os.ReadFile(filepath.Join(repoDir, "README.md"))
	if err != nil {
		return repoName
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(title)
		}
	}
	return repoName
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
