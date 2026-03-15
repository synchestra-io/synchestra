package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/internal"
)

// ProjectNewCommand returns the "project new" subcommand.
func ProjectNewCommand(osUserHomeDir func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Synchestra project",
		Long:  "Creates a new Synchestra project by linking a spec repo, state repo, and one or more target repos. The command resolves all repo references, clones any that are not already on disk, validates they are git repos, writes the appropriate config files to each, and commits and pushes the changes.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			specRepoRef, _ := cmd.Flags().GetString("spec-repo")
			stateRepoRef, _ := cmd.Flags().GetString("state-repo")
			targetRepoRefs, _ := cmd.Flags().GetStringSlice("target-repo")
			providedTitle, _ := cmd.Flags().GetString("title")

			err := projectNewRun(osUserHomeDir, specRepoRef, stateRepoRef, targetRepoRefs, providedTitle)
			if exitErr, ok := err.(*internal.ExitError); ok {
				return errors.New(exitErr.Error())
			}
			return err
		},
	}

	cmd.Flags().String("spec-repo", "", "Spec repository reference (required)")
	cmd.Flags().String("state-repo", "", "State repository reference (required)")
	cmd.Flags().StringSlice("target-repo", []string{}, "Target repository reference (repeatable, at least one required)")
	cmd.Flags().String("title", "", "Project title (optional; derived from spec repo README or identifier if omitted)")

	cmd.MarkFlagRequired("spec-repo")
	cmd.MarkFlagRequired("state-repo")

	return cmd
}

// projectNewRun contains the logic for project new, separated from Cobra for testability.
func projectNewRun(
	osUserHomeDir func() (string, error),
	specRepoRef string,
	stateRepoRef string,
	targetRepoRefs []string,
	providedTitle string,
) error {
	// Validate: at least one target repo
	if len(targetRepoRefs) == 0 {
		return internal.NewExitError(2, "at least one --target-repo is required")
	}

	// Load global config
	cfg, err := internal.LoadGlobalConfig(osUserHomeDir)
	if err != nil {
		return internal.WrapExitError(10, "could not load config", err)
	}

	// Parse repo references
	specRef, err := internal.ParseRepoReference(specRepoRef)
	if err != nil {
		return internal.WrapExitError(2, "invalid --spec-repo", err)
	}

	stateRef, err := internal.ParseRepoReference(stateRepoRef)
	if err != nil {
		return internal.WrapExitError(2, "invalid --state-repo", err)
	}

	targetRefs := make([]*internal.RepoRef, len(targetRepoRefs))
	for i, ref := range targetRepoRefs {
		parsed, err := internal.ParseRepoReference(ref)
		if err != nil {
			return internal.WrapExitError(2, fmt.Sprintf("invalid --target-repo[%d]", i), err)
		}
		targetRefs[i] = parsed
	}

	// Resolve paths
	specPath, err := internal.ResolveRepoPath(cfg.ReposDir, specRef)
	if err != nil {
		return internal.WrapExitError(2, "invalid spec repo path", err)
	}

	statePath, err := internal.ResolveRepoPath(cfg.ReposDir, stateRef)
	if err != nil {
		return internal.WrapExitError(2, "invalid state repo path", err)
	}

	targetPaths := make([]string, len(targetRefs))
	for i, ref := range targetRefs {
		path, err := internal.ResolveRepoPath(cfg.ReposDir, ref)
		if err != nil {
			return internal.WrapExitError(2, fmt.Sprintf("invalid target repo[%d] path", i), err)
		}
		targetPaths[i] = path
	}

	// Clone repos
	specCloneURL := internal.NormalizeCloneURL(specRepoRef, specRef)
	if _, err := internal.CloneRepo(specCloneURL, specPath); err != nil {
		return internal.WrapExitError(3, "could not clone spec repo", err)
	}

	stateCloneURL := internal.NormalizeCloneURL(stateRepoRef, stateRef)
	if _, err := internal.CloneRepo(stateCloneURL, statePath); err != nil {
		return internal.WrapExitError(3, "could not clone state repo", err)
	}

	for i, targetRef := range targetRepoRefs {
		targetCloneURL := internal.NormalizeCloneURL(targetRef, targetRefs[i])
		if _, err := internal.CloneRepo(targetCloneURL, targetPaths[i]); err != nil {
			return internal.WrapExitError(3, fmt.Sprintf("could not clone target repo[%d]", i), err)
		}
	}

	// Validate all are git repos
	if _, err := internal.ValidateGitRepo(specPath); err != nil {
		return internal.WrapExitError(3, "spec repo is not a valid git repo", err)
	}

	if _, err := internal.ValidateGitRepo(statePath); err != nil {
		return internal.WrapExitError(3, "state repo is not a valid git repo", err)
	}

	for i, targetPath := range targetPaths {
		if _, err := internal.ValidateGitRepo(targetPath); err != nil {
			return internal.WrapExitError(3, fmt.Sprintf("target repo[%d] is not a valid git repo", i), err)
		}
	}

	// Get origin URL upfront for conflict checking
	specOriginURL, err := internal.GetOriginURL(specPath)
	if err != nil {
		return internal.WrapExitError(10, "could not get spec repo origin URL", err)
	}

	// Check for existing conflicting configs
	existingSpec, err := internal.ReadSpecConfig(specPath)
	if err != nil {
		return internal.WrapExitError(10, "could not check existing spec config", err)
	}
	if existingSpec != nil {
		return internal.NewExitError(1, "spec repo already has config for a different project")
	}

	existingState, err := internal.ReadStateConfig(statePath)
	if err != nil {
		return internal.WrapExitError(10, "could not check existing state config", err)
	}
	if existingState != nil {
		if existingState.SpecRepo != specOriginURL {
			return internal.NewExitError(1, "state repo already has config for different spec repo")
		}
	}

	for i, targetPath := range targetPaths {
		existingTarget, err := internal.ReadTargetConfig(targetPath)
		if err != nil {
			return internal.WrapExitError(10, fmt.Sprintf("could not check target repo[%d] config", i), err)
		}
		if existingTarget != nil {
			if existingTarget.SpecRepo != specOriginURL {
				return internal.NewExitError(1, fmt.Sprintf("target repo[%d] already has config for different spec repo", i))
			}
		}
	}

	// Get state and target origin URLs
	stateOriginURL, err := internal.GetOriginURL(statePath)
	if err != nil {
		return internal.WrapExitError(10, "could not get state repo origin URL", err)
	}

	targetOriginURLs := make([]string, len(targetPaths))
	for i, targetPath := range targetPaths {
		url, err := internal.GetOriginURL(targetPath)
		if err != nil {
			return internal.WrapExitError(10, fmt.Sprintf("could not get target repo[%d] origin URL", i), err)
		}
		targetOriginURLs[i] = url
	}

	// Derive title
	title := internal.DeriveTitle(specPath, specRef, providedTitle)

	// Write config files
	specConfig := &internal.SynchstraSpecYaml{
		Title:     title,
		StateRepo: stateOriginURL,
		Repos:     targetOriginURLs,
	}
	if err := internal.WriteSpecConfig(specPath, specConfig); err != nil {
		return internal.WrapExitError(10, "could not write spec config", err)
	}

	stateConfig := &internal.SynchstraStateYaml{
		SpecRepo: specOriginURL,
	}
	if err := internal.WriteStateConfig(statePath, stateConfig); err != nil {
		return internal.WrapExitError(10, "could not write state config", err)
	}

	for i, targetPath := range targetPaths {
		targetConfig := &internal.SynchstraTargetYaml{
			SpecRepo: specOriginURL,
		}
		if err := internal.WriteTargetConfig(targetPath, targetConfig); err != nil {
			return internal.WrapExitError(10, fmt.Sprintf("could not write target repo[%d] config", i), err)
		}
	}

	// Commit and push to all repos
	commitMessage := fmt.Sprintf("synchestra: project new - %s", title)

	if err := internal.GitCommitAndPush(specPath, commitMessage); err != nil {
		return internal.WrapExitError(10, "could not commit/push to spec repo", err)
	}

	if err := internal.GitCommitAndPush(statePath, commitMessage); err != nil {
		return internal.WrapExitError(10, "could not commit/push to state repo", err)
	}

	for i, targetPath := range targetPaths {
		if err := internal.GitCommitAndPush(targetPath, commitMessage); err != nil {
			return internal.WrapExitError(10, fmt.Sprintf("could not commit/push to target repo[%d]", i), err)
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Project created: %s\n", title)
	return nil
}
