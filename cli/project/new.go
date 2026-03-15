package project

// Features implemented: cli/project/new
// Features depended on:  global-config, project-definition

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	return cmd
}

func runNew(cmd *cobra.Command, _ []string) error {
	specRepoStr, _ := cmd.Flags().GetString("spec-repo")
	stateRepoStr, _ := cmd.Flags().GetString("state-repo")
	targetRepoStrs, _ := cmd.Flags().GetStringArray("target-repo")
	titleFlag, _ := cmd.Flags().GetString("title")

	if err := validateRequiredRepoFlags(specRepoStr, stateRepoStr, targetRepoStrs); err != nil {
		return err
	}

	specRef, err := reporef.Parse(specRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --spec-repo: %v", err)}
	}
	stateRef, err := reporef.Parse(stateRepoStr)
	if err != nil {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid --state-repo: %v", err)}
	}

	targetRefs := make([]reporef.Ref, 0, len(targetRepoStrs))
	for _, s := range targetRepoStrs {
		ref, err := reporef.Parse(s)
		if err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("invalid --target-repo %q: %v", s, err)}
		}
		targetRefs = append(targetRefs, ref)
	}

	if err := validateDistinctRepoRoles(specRef, stateRef, targetRefs); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("cannot determine home directory: %v", err)}
	}
	cfg, err := globalconfig.Load(filepath.Join(homeDir, ".synchestra.yaml"))
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("reading global config: %v", err)}
	}
	reposDir := globalconfig.ResolveReposDir(cfg.ReposDir, homeDir)

	allRefs := append([]reporef.Ref{specRef, stateRef}, targetRefs...)
	allPaths := make([]string, len(allRefs))
	for i, ref := range allRefs {
		allPaths[i] = ref.DiskPath(reposDir)
		if err := validateResolvedRepoPath(reposDir, allPaths[i], ref.Identifier()); err != nil {
			return err
		}
	}
	specPath, statePath := allPaths[0], allPaths[1]
	targetPaths := allPaths[2:]

	for i, ref := range allRefs {
		p := allPaths[i]
		info, err := os.Stat(p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Cloning %s...\n", ref.Identifier())
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("creating directory for %s: %v", ref.Identifier(), err)}
			}
			if err := gitops.Clone(ref.OriginURL(), p); err != nil {
				return &exitError{code: 3, msg: fmt.Sprintf("cloning %s: %v", ref.Identifier(), err)}
			}
		case err != nil:
			return &exitError{code: 10, msg: fmt.Sprintf("checking %s: %v", ref.Identifier(), err)}
		case !info.IsDir():
			return &exitError{code: 1, msg: fmt.Sprintf("path for %s already exists and is not a directory: %s", ref.Identifier(), p)}
		}
	}

	for i, ref := range allRefs {
		if !gitops.IsGitRepo(allPaths[i]) {
			return &exitError{code: 3, msg: fmt.Sprintf("%s is not a git repository", ref.Identifier())}
		}
		if err := ensureCheckoutMatchesRef(allPaths[i], ref); err != nil {
			return err
		}
	}

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

	title := DeriveTitle(titleFlag, specPath, specRef.Repo)
	targetOriginURLs := make([]string, len(targetRefs))
	for i, ref := range targetRefs {
		targetOriginURLs[i] = ref.OriginURL()
	}

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

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project %q created successfully.\n", title)
	return nil
}

func validateRequiredRepoFlags(specRepoStr, stateRepoStr string, targetRepoStrs []string) error {
	switch {
	case strings.TrimSpace(specRepoStr) == "":
		return &exitError{code: 2, msg: "--spec-repo is required"}
	case strings.TrimSpace(stateRepoStr) == "":
		return &exitError{code: 2, msg: "--state-repo is required"}
	case len(targetRepoStrs) == 0:
		return &exitError{code: 2, msg: "at least one --target-repo is required"}
	default:
		return nil
	}
}

func validateDistinctRepoRoles(specRef, stateRef reporef.Ref, targetRefs []reporef.Ref) error {
	if specRef == stateRef {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid repository layout: state repo %s must differ from spec repo %s", stateRef.Identifier(), specRef.Identifier())}
	}

	seen := map[string]string{
		specRef.Identifier():  "spec repo",
		stateRef.Identifier(): "state repo",
	}
	for i, ref := range targetRefs {
		id := ref.Identifier()
		if prevRole, ok := seen[id]; ok {
			return &exitError{code: 2, msg: fmt.Sprintf("invalid repository layout: target repo %s must differ from %s", id, prevRole)}
		}
		seen[id] = fmt.Sprintf("target repo #%d", i+1)
	}
	return nil
}

func validateResolvedRepoPath(reposDir, path, identifier string) error {
	reposDirAbs, err := filepath.Abs(reposDir)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("resolving repos_dir: %v", err)}
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("resolving local path for %s: %v", identifier, err)}
	}

	rel, err := filepath.Rel(reposDirAbs, pathAbs)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("checking local path for %s: %v", identifier, err)}
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return &exitError{code: 2, msg: fmt.Sprintf("unsafe local path for %s: %s resolves outside repos_dir", identifier, pathAbs)}
	}
	if err := rejectSymlinkPath(reposDirAbs, pathAbs); err != nil {
		return &exitError{code: 1, msg: fmt.Sprintf("unsafe local path for %s: %v", identifier, err)}
	}
	return nil
}

func rejectSymlinkPath(rootAbs, pathAbs string) error {
	if err := rejectSymlink(rootAbs); err != nil {
		return err
	}

	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}

	current := rootAbs
	for _, segment := range strings.Split(rel, string(os.PathSeparator)) {
		if segment == "" || segment == "." {
			continue
		}
		current = filepath.Join(current, segment)
		if err := rejectSymlink(current); err != nil {
			return err
		}
	}
	return nil
}

func rejectSymlink(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}
	return nil
}

func ensureCheckoutMatchesRef(dir string, expected reporef.Ref) error {
	originURL, err := gitops.GetOriginURL(dir)
	if err != nil {
		return &exitError{code: 1, msg: fmt.Sprintf("cannot verify origin for %s in %s: %v", expected.Identifier(), dir, err)}
	}
	originRef, err := reporef.Parse(originURL)
	if err != nil {
		return &exitError{code: 1, msg: fmt.Sprintf("existing checkout for %s in %s has unsupported origin %q: %v", expected.Identifier(), dir, originURL, err)}
	}
	if originRef != expected {
		return &exitError{code: 1, msg: fmt.Sprintf("existing checkout in %s points to %s, not %s", dir, originRef.Identifier(), expected.Identifier())}
	}
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
