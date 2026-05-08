// Package synchinit implements the top-level `synchestra init` command.
// Package name is `synchinit` (rather than the more direct `init`) to
// avoid visual collision with Go's builtin func init().
package synchinit

// Features implemented: cli/init, repo-config, state-repo-config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/gitops"
	"github.com/synchestra-io/synchestra/pkg/cli/project"
)

// Canonical synchestra.yaml schema-pointer comment per
// repo-config#req:schema-header-comment.
const synchestraSchemaHeader = "# Synchestra Repo Config Schema: https://synchestra.md/repo-config"

// SynchestraConfigFile is the canonical filename per repo-config#req:file-name.
const SynchestraConfigFile = "synchestra.yaml"

const (
	stateModeEmbedded    = "embedded"
	stateModeSeparate    = "separate-repo"
	stateModeHubManaged  = "hub-managed"
	defaultStateBranch   = "synchestra-state"
	worktreeDir          = ".synchestra"
)

// Command returns the cobra.Command for `synchestra init`.
//
// Top-level placement is intentional per cli/init#req:top-level-command —
// the only command that breaks the noun-verb convention, matching the
// muscle-memory of `git init` / `npm init` / `cargo init`.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a Synchestra-managed project (creates synchestra.yaml + state)",
		Long: `Bootstrap a Synchestra-managed project at the current repo root.

Creates synchestra.yaml at the repository root and sets up state per
the chosen mode. By default, mode=embedded creates an orphan branch
(default name "synchestra-state") and a git worktree at .synchestra/.

Idempotent: rerunning is safe. Mode-mismatch on rerun is a hard error
(exit 4); edit synchestra.yaml manually to change modes (or wait for
--force in a future release).

This is the unified bootstrap entry-point. It does NOT yet replace the
older "synchestra project init" / "synchestra project new" commands —
deprecation is tracked separately.`,
		Args: cobra.NoArgs,
		RunE: runInit,
	}
	cmd.Flags().String("state-mode", stateModeEmbedded, "state mode: embedded | separate-repo | hub-managed (only embedded is implemented in v1)")
	cmd.Flags().String("branch", defaultStateBranch, "name of the orphan branch for embedded state")
	cmd.Flags().String("project", "", "project root (autodetected from current directory if omitted)")
	cmd.Flags().Bool("no-push", false, "skip pushing the orphan branch to the remote (local-only mode)")
	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	stateMode, _ := cmd.Flags().GetString("state-mode")
	branch, _ := cmd.Flags().GetString("branch")
	projectFlag, _ := cmd.Flags().GetString("project")
	noPush, _ := cmd.Flags().GetBool("no-push")

	// 1. Validate --state-mode flag value (per cli/init#req:state-mode-flag).
	switch stateMode {
	case stateModeEmbedded:
		// fall through to embedded init below
	case stateModeSeparate, stateModeHubManaged:
		return exitcode.InvalidArgsErrorf(
			"--state-mode %q is recognized but not yet implemented in v1; only %q is implemented (cli/init#req:state-mode-flag)",
			stateMode, stateModeEmbedded,
		)
	default:
		return exitcode.InvalidArgsErrorf(
			"--state-mode %q is invalid; accepted values: %s, %s, %s (cli/init#req:state-mode-flag)",
			stateMode, stateModeEmbedded, stateModeSeparate, stateModeHubManaged,
		)
	}

	// 2. Resolve project root (per cli/init#req:project-flag).
	root, err := resolveProjectRoot(projectFlag)
	if err != nil {
		return err
	}

	// 3. Verify git repo (per cli/init#req:requires-git-repo).
	if !gitops.IsGitRepo(root) {
		return exitcode.Newf(exitcode.NotFound,
			"not a git repository: %s\nRun `git init` first, then re-run `synchestra init`. (cli/init#req:requires-git-repo)",
			root,
		)
	}
	repoRoot, err := gitops.RepoRoot(root)
	if err != nil {
		return exitcode.UnexpectedErrorf("finding repository root: %v", err)
	}

	// 4. Idempotence + mode-mismatch detection (per cli/init#req:idempotent-rerun).
	configPath := filepath.Join(repoRoot, SynchestraConfigFile)
	existingMode, hadExisting, err := readExistingMode(configPath)
	if err != nil {
		return err
	}
	if hadExisting {
		if existingMode != stateMode {
			return exitcode.Newf(exitcode.InvalidState,
				"existing synchestra.yaml has state.mode=%q; cannot re-initialize with --state-mode=%q. Edit synchestra.yaml manually to change modes (--force not available in v1). (cli/init#req:idempotent-rerun)",
				existingMode, stateMode,
			)
		}
		// Same-mode rerun: report no-op, then idempotently ensure state setup.
		fmt.Fprintf(cmd.OutOrStdout(), "Project already initialized at %s (state.mode=%s)\n", repoRoot, stateMode)
		if stateMode == stateModeEmbedded {
			if err := ensureEmbeddedStateOnRerun(repoRoot, branch, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return err
			}
		}
		return nil
	}

	// 5. Greenfield init.
	if stateMode == stateModeEmbedded {
		if err := provisionEmbeddedState(repoRoot, branch, noPush, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return err
		}
	}

	// 6. Write synchestra.yaml at repo root (per cli/init#req:writes-synchestra-yaml).
	if err := writeSynchestraConfig(configPath, stateMode, branch); err != nil {
		return exitcode.UnexpectedErrorf("writing %s: %v", SynchestraConfigFile, err)
	}

	// 7. Ensure .gitignore contains .synchestra (per cli/init#req:gitignore-synchestra-entry).
	if stateMode == stateModeEmbedded {
		if err := ensureGitignoreEntry(repoRoot, worktreeDir); err != nil {
			return exitcode.UnexpectedErrorf("updating .gitignore: %v", err)
		}
	}

	// 8. Success summary (per cli/init#req:success-output).
	worktreePath := filepath.Join(repoRoot, worktreeDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Synchestra project initialized\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Config:   %s (state.mode=%s)\n", configPath, stateMode)
	if stateMode == stateModeEmbedded {
		fmt.Fprintf(cmd.OutOrStdout(), "  Worktree: %s\n  Branch:   %s\n", worktreePath, branch)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Next: `synchestra task new` to create your first task.\n")
	return nil
}

// resolveProjectRoot resolves --project's value (or cwd if empty) and
// verifies the path exists and is a directory. Per cli/init#req:project-flag.
func resolveProjectRoot(projectFlag string) (string, error) {
	var path string
	if projectFlag != "" {
		abs, err := filepath.Abs(projectFlag)
		if err != nil {
			return "", exitcode.InvalidArgsErrorf("resolving --project path: %v", err)
		}
		path = abs
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", exitcode.UnexpectedErrorf("cannot determine working directory: %v", err)
		}
		path = cwd
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", exitcode.InvalidArgsErrorf("project root does not exist: %s", path)
		}
		return "", exitcode.UnexpectedErrorf("stat %s: %v", path, err)
	}
	if !st.IsDir() {
		return "", exitcode.InvalidArgsErrorf("project root is not a directory: %s", path)
	}
	return path, nil
}

// readExistingMode reads an existing synchestra.yaml's state.mode if any.
// Returns (mode, hadFile, err). hadFile is true iff the file exists; in
// that case mode is the parsed mode (or empty if no state block).
func readExistingMode(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, exitcode.UnexpectedErrorf("reading %s: %v", SynchestraConfigFile, err)
	}
	// Schema-header check: first line MUST be the canonical comment per
	// repo-config#req:schema-header-comment.
	idx := bytes.IndexByte(data, '\n')
	first := data
	if idx >= 0 {
		first = data[:idx]
	}
	if string(bytes.TrimRight(first, "\r")) != synchestraSchemaHeader {
		return "", true, exitcode.Newf(exitcode.InvalidState,
			"existing %s has malformed schema header on line 1; expected %q. Manual repair required. (repo-config#req:schema-header-comment)",
			path, synchestraSchemaHeader,
		)
	}
	// Lightweight scan for `mode:` inside a `state:` block. Avoids pulling
	// in a YAML dependency for a single field; the canonical parser is a
	// separate Feature.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inState := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasPrefix(trimmed, "state:") {
			inState = true
			continue
		}
		if inState {
			// Continued lines inside the state block are indented.
			if len(trimmed) > 0 && !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t") {
				inState = false
				continue
			}
			s := strings.TrimSpace(trimmed)
			if strings.HasPrefix(s, "mode:") {
				val := strings.TrimSpace(strings.TrimPrefix(s, "mode:"))
				val = strings.Trim(val, `"'`)
				return val, true, nil
			}
		}
	}
	// File exists but no state.mode found.
	return "", true, nil
}

// writeSynchestraConfig writes a fresh synchestra.yaml with the given
// mode and branch. The line-1 schema-pointer comment is mandatory per
// repo-config#req:schema-header-comment.
func writeSynchestraConfig(path, mode, branch string) error {
	var body strings.Builder
	body.WriteString(synchestraSchemaHeader)
	body.WriteString("\n\n")
	body.WriteString("state:\n")
	body.WriteString("  mode: ")
	body.WriteString(mode)
	body.WriteString("\n")
	if mode == stateModeEmbedded {
		body.WriteString("  branch: ")
		body.WriteString(branch)
		body.WriteString("\n")
	}
	return os.WriteFile(path, []byte(body.String()), 0o644)
}

// provisionEmbeddedState creates the orphan branch + worktree for
// embedded mode. Mirrors the steps in project.runInit but writes
// synchestra.yaml-style state config instead of the legacy spec config.
// Per cli/init#req:embedded-state-setup.
func provisionEmbeddedState(repoRoot, branch string, noPush bool, stdout, stderr io.Writer) error {
	worktreePath := filepath.Join(repoRoot, worktreeDir)

	// Skip if a valid worktree already exists.
	if isValidWorktree(repoRoot, worktreePath) {
		return nil
	}
	_ = gitops.WorktreePrune(repoRoot)

	originalBranch, err := gitops.CurrentBranch(repoRoot)
	if err != nil {
		return exitcode.UnexpectedErrorf("detecting current branch: %v", err)
	}

	// Branch already exists on remote? Fetch + worktree.
	if gitops.RemoteBranchExists(repoRoot, "origin", branch) {
		fmt.Fprintf(stderr, "Found existing state branch on remote, fetching...\n")
		if err := gitops.FetchBranch(repoRoot, "origin", branch); err != nil {
			return exitcode.UnexpectedErrorf("fetching remote branch: %v", err)
		}
		if !gitops.BranchExists(repoRoot, branch) {
			if err := gitops.CreateTrackingBranch(repoRoot, branch, "origin"); err != nil {
				return exitcode.UnexpectedErrorf("creating tracking branch: %v", err)
			}
		}
		if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
			return exitcode.UnexpectedErrorf("creating worktree: %v", err)
		}
		return nil
	}

	// Branch already exists locally? Just attach worktree.
	if gitops.BranchExists(repoRoot, branch) {
		if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
			return exitcode.UnexpectedErrorf("creating worktree from local branch: %v", err)
		}
		return nil
	}

	// Greenfield orphan-branch creation.
	title := filepath.Base(repoRoot)
	fmt.Fprintf(stderr, "Creating state branch %q...\n", branch)

	if err := gitops.CreateOrphanBranch(repoRoot, branch); err != nil {
		return exitcode.UnexpectedErrorf("creating orphan branch: %v", err)
	}

	cfg := project.EmbeddedStateConfig{
		Title:        title,
		Mode:         "embedded",
		SourceBranch: originalBranch,
		Sync: &project.EmbeddedSyncCfg{
			Pull: "on_commit",
			Push: "on_commit",
		},
	}
	if err := project.WriteEmbeddedStateConfig(repoRoot, cfg); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("writing embedded state config: %v", err)
	}

	tasksDir := filepath.Join(repoRoot, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("creating tasks directory: %v", err)
	}
	emptyBoard := []byte("# Tasks\n\n| Task | Status | Depends on | Branch | Agent | Requester | Time |\n|---|---|---|---|---|---|---|\n")
	if err := os.WriteFile(filepath.Join(tasksDir, "README.md"), emptyBoard, 0o644); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("writing task board: %v", err)
	}

	branchReadme := fmt.Sprintf("# %s — Synchestra State\n\nThis branch contains Synchestra coordination state for the project.\n\nSee `tasks/README.md` for the task board.\n", title)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(branchReadme), 0o644); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("writing state-branch README: %v", err)
	}

	if err := gitops.Commit(repoRoot, []string{project.EmbeddedStateFile, "tasks/README.md", "README.md"}, "Initialize Synchestra state"); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("committing initial state: %v", err)
	}

	if err := gitops.CheckoutBranch(repoRoot, originalBranch); err != nil {
		return exitcode.UnexpectedErrorf("returning to branch %s: %v", originalBranch, err)
	}

	if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
		return exitcode.UnexpectedErrorf("creating worktree: %v", err)
	}

	if !noPush {
		if err := gitops.PushNewBranch(repoRoot, "origin", branch); err != nil {
			fmt.Fprintf(stderr, "Warning: could not push state branch to remote: %v\n", err)
			fmt.Fprintf(stderr, "Run `git push -u origin %s` later to sync.\n", branch)
		}
	}
	_ = stdout // success summary printed by the caller
	return nil
}

// ensureEmbeddedStateOnRerun ensures the worktree exists when an existing
// synchestra.yaml says embedded mode but the worktree is missing. Per
// cli/init#req:idempotent-rerun.
func ensureEmbeddedStateOnRerun(repoRoot, branch string, stdout, stderr io.Writer) error {
	worktreePath := filepath.Join(repoRoot, worktreeDir)
	if isValidWorktree(repoRoot, worktreePath) {
		return nil
	}
	_ = gitops.WorktreePrune(repoRoot)
	if !gitops.BranchExists(repoRoot, branch) {
		fmt.Fprintf(stderr, "Worktree missing AND state branch %q does not exist; run `synchestra init` from a clean state. (cli/init#req:idempotent-rerun)\n", branch)
		return exitcode.UnexpectedErrorf("state branch %s missing on rerun", branch)
	}
	if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
		return exitcode.UnexpectedErrorf("recreating worktree: %v", err)
	}
	fmt.Fprintf(stdout, "  Recreated missing worktree at %s\n", worktreePath)
	return nil
}

// isValidWorktree reports whether worktreePath is an active git worktree
// of repoRoot.
func isValidWorktree(repoRoot, worktreePath string) bool {
	info, err := os.Stat(worktreePath)
	if err != nil || !info.IsDir() {
		return false
	}
	paths, err := gitops.WorktreeList(repoRoot)
	if err != nil {
		return false
	}
	for _, p := range paths {
		if p == worktreePath {
			return true
		}
	}
	return false
}

// ensureGitignoreEntry adds the entry to .gitignore at repoRoot if not
// already present. Preserves any existing content. Per
// cli/init#req:gitignore-synchestra-entry.
func ensureGitignoreEntry(repoRoot, entry string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == entry {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer func() { _ = f.Close() }()
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("writing newline to .gitignore: %w", err)
		}
	}
	if _, err := f.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("writing to .gitignore: %w", err)
	}
	return nil
}
