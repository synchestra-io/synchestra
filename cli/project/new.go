package project

// Features implemented: cli/project/new
// Features depended on:  global-config, project-definition

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/cli/gitops"
	"github.com/synchesta-io/synchestra/cli/globalconfig"
	"github.com/synchesta-io/synchestra/cli/reporef"
	"gopkg.in/yaml.v3"
)

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new Synchestra project",
		Long: `Creates a new Synchestra project by linking a spec repo, state repo, and
one or more target repos. Resolves all repo references, clones any that are
not already on disk, validates they are git repos, writes config files to
each, and commits and pushes the changes.`,
		RunE: runNew,
	}
	cmd.Flags().String("spec-repo", "", "spec repository reference (required)")
	cmd.Flags().String("state-repo", "", "state repository reference (required)")
	cmd.Flags().StringArray("target-repo", nil, "target repository reference (repeatable, at least one required)")
	cmd.Flags().String("title", "", "project title (default: derived from spec repo README)")
	_ = cmd.MarkFlagRequired("spec-repo")
	_ = cmd.MarkFlagRequired("state-repo")
	_ = cmd.MarkFlagRequired("target-repo")
	return cmd
}

func runNew(cmd *cobra.Command, _ []string) error {
	specRepoStr, _ := cmd.Flags().GetString("spec-repo")
	stateRepoStr, _ := cmd.Flags().GetString("state-repo")
	targetRepoStrs, _ := cmd.Flags().GetStringArray("target-repo")
	titleFlag, _ := cmd.Flags().GetString("title")

	if len(targetRepoStrs) == 0 {
		return &exitError{code: 2, msg: "at least one --target-repo is required"}
	}

	// Parse repo references
	specRef, err := reporef.Parse(specRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --spec-repo: %v", err)}
	}
	stateRef, err := reporef.Parse(stateRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --state-repo: %v", err)}
	}
	var targetRefs []reporef.Ref
	for _, s := range targetRepoStrs {
		ref, err := reporef.Parse(s)
		if err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("invalid --target-repo %q: %v", s, err)}
		}
		targetRefs = append(targetRefs, ref)
	}

	// Load global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("cannot determine home directory: %v", err)}
	}
	cfg, err := globalconfig.Load(filepath.Join(homeDir, ".synchestra.yaml"))
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("reading global config: %v", err)}
	}
	reposDir := globalconfig.ResolveReposDir(cfg.ReposDir, homeDir)

	// Resolve disk paths
	allRefs := append([]reporef.Ref{specRef, stateRef}, targetRefs...)
	allPaths := make([]string, len(allRefs))
	for i, ref := range allRefs {
		allPaths[i] = ref.DiskPath(reposDir)
	}
	specPath, statePath := allPaths[0], allPaths[1]
	targetPaths := allPaths[2:]

	// Clone repos that don't exist on disk
	for i, ref := range allRefs {
		p := allPaths[i]
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Cloning %s...\n", ref.Identifier())
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("creating directory for %s: %v", ref.Identifier(), err)}
			}
			if err := gitops.Clone(ref.OriginURL(), p); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("cloning %s: %v", ref.Identifier(), err)}
			}
		}
	}

	// Validate all are git repos
	for i, ref := range allRefs {
		if !gitops.IsGitRepo(allPaths[i]) {
			return &exitError{code: 3, msg: fmt.Sprintf("%s is not a git repository", ref.Identifier())}
		}
	}

	// Check for existing config files pointing to a different project
	if err := checkSpecConflict(specPath, stateRef.OriginURL()); err != nil {
		return err
	}
	if err := checkBackrefConflict(statePath, StateConfigFile, specRef.OriginURL()); err != nil {
		return err
	}
	for _, tp := range targetPaths {
		if err := checkBackrefConflict(tp, TargetConfigFile, specRef.OriginURL()); err != nil {
			return err
		}
	}

	// Derive title
	title := DeriveTitle(titleFlag, specPath, specRef.Repo)

	// Collect target origin URLs
	targetOriginURLs := make([]string, len(targetRefs))
	for i, ref := range targetRefs {
		targetOriginURLs[i] = ref.OriginURL()
	}

	// Write config files
	specCfg := SpecConfig{
		Title:     title,
		StateRepo: stateRef.OriginURL(),
		Repos:     targetOriginURLs,
	}
	if err := WriteSpecConfig(specPath, specCfg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("writing spec config: %v", err)}
	}

	stateCfg := StateConfig{SpecRepo: specRef.OriginURL()}
	if err := WriteStateConfig(statePath, stateCfg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("writing state config: %v", err)}
	}

	for _, tp := range targetPaths {
		targetCfg := TargetConfig{SpecRepo: specRef.OriginURL()}
		if err := WriteTargetConfig(tp, targetCfg); err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("writing target config: %v", err)}
		}
	}

	// Commit and push
	commitMsg := fmt.Sprintf("synchestra: initialize project %q", title)

	if err := gitops.CommitAndPush(specPath, []string{SpecConfigFile}, commitMsg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("committing spec repo: %v", err)}
	}
	if err := gitops.CommitAndPush(statePath, []string{StateConfigFile}, commitMsg); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("committing state repo: %v", err)}
	}
	for i, tp := range targetPaths {
		if err := gitops.CommitAndPush(tp, []string{TargetConfigFile}, commitMsg); err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("committing target repo %s: %v", targetRefs[i].Identifier(), err)}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Project %q created successfully.\n", title)
	return nil
}

// checkSpecConflict checks if synchestra-spec.yaml exists and points to a
// different state repo (i.e., belongs to a different project).
func checkSpecConflict(dir, expectedStateRepo string) error {
	cfg, err := ReadSpecConfig(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return &exitError{code: 10, msg: fmt.Sprintf("reading existing spec config: %v", err)}
	}
	if cfg.StateRepo != "" && cfg.StateRepo != expectedStateRepo {
		return &exitError{
			code: 1,
			msg:  fmt.Sprintf("conflict: %s in %s already points to state repo %s", SpecConfigFile, dir, cfg.StateRepo),
		}
	}
	return nil
}

// checkBackrefConflict checks if a state or target config file exists and
// its spec_repo field points to a different spec repo.
func checkBackrefConflict(dir, filename, expectedSpecRepo string) error {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return &exitError{code: 10, msg: fmt.Sprintf("reading %s: %v", path, err)}
	}
	var backref struct {
		SpecRepo string `yaml:"spec_repo"`
	}
	if err := yaml.Unmarshal(data, &backref); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("parsing %s: %v", path, err)}
	}
	if backref.SpecRepo != "" && backref.SpecRepo != expectedSpecRepo {
		return &exitError{
			code: 1,
			msg:  fmt.Sprintf("conflict: %s in %s already points to spec repo %s", filename, dir, backref.SpecRepo),
		}
	}
	return nil
}

// exitError is an error that carries an exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// ExitCode returns the exit code for the error.
func (e *exitError) ExitCode() int { return e.code }
