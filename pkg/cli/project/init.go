package project

// Features implemented: cli/project/init, embedded-state
// Features depended on:  state-store, task-status-board

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"github.com/synchestra-io/synchestra/pkg/cli/gitops"
)

const (
	defaultStateBranch = "synchestra-state"
	worktreeDir        = ".synchestra"
)

func initCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize embedded Synchestra state in the current repo",
		Long: `Initializes Synchestra embedded state in the current git repository.
Creates an orphan branch for state management and sets up a git worktree
at .synchestra/. This is the zero-friction alternative to "synchestra project new".`,
		Args: cobra.NoArgs,
		RunE: runInit,
	}
	cmd.Flags().String("title", "", "project title (default: derived from README heading or directory name)")
	cmd.Flags().String("branch", defaultStateBranch, "name of the orphan branch for state")
	cmd.Flags().Bool("no-push", false, "skip pushing the branch to the remote (local-only mode)")
	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	titleFlag, _ := cmd.Flags().GetString("title")
	branch, _ := cmd.Flags().GetString("branch")
	noPush, _ := cmd.Flags().GetBool("no-push")

	cwd, err := os.Getwd()
	if err != nil {
		return exitcode.UnexpectedErrorf("getting working directory: %v", err)
	}

	// Step 1: Verify we're in a git repo.
	if !gitops.IsGitRepo(cwd) {
		return exitcode.Newf(exitcode.NotFound, "not a git repository: %s", cwd)
	}

	repoRoot, err := gitops.RepoRoot(cwd)
	if err != nil {
		return exitcode.UnexpectedErrorf("finding repository root: %v", err)
	}

	worktreePath := filepath.Join(repoRoot, worktreeDir)

	// Step 2: Check if already initialized (idempotent).
	if isValidWorktree(repoRoot, worktreePath) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Already initialized: %s (branch: %s)\n", worktreePath, branch)
		return nil
	}

	// Prune stale worktree entries before proceeding.
	_ = gitops.WorktreePrune(repoRoot)

	// Check for conflict with dedicated project setup.
	if _, err := os.Stat(filepath.Join(repoRoot, SpecConfigFile)); err == nil {
		return exitcode.ConflictError("this repository already has a dedicated project setup (" + SpecConfigFile + "); embedded state cannot be used alongside it")
	}

	// Remember the current branch to switch back after creating orphan.
	originalBranch, err := gitops.CurrentBranch(repoRoot)
	if err != nil {
		return exitcode.UnexpectedErrorf("detecting current branch: %v", err)
	}

	// Step 3: Check if branch exists on remote.
	if gitops.RemoteBranchExists(repoRoot, "origin", branch) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Found existing state branch on remote, fetching...\n")
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
		if err := ensureGitignoreEntry(repoRoot, worktreeDir); err != nil {
			return exitcode.UnexpectedErrorf("updating .gitignore: %v", err)
		}
		if err := ensureEmbeddedConfig(repoRoot, branch); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Embedded state connected (existing project)\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Worktree: %s\n  Branch:   %s\n", worktreePath, branch)
		return nil
	}

	// Step 4: Check if branch exists locally.
	if gitops.BranchExists(repoRoot, branch) {
		if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
			return exitcode.UnexpectedErrorf("creating worktree from local branch: %v", err)
		}
		if err := ensureGitignoreEntry(repoRoot, worktreeDir); err != nil {
			return exitcode.UnexpectedErrorf("updating .gitignore: %v", err)
		}
		if err := ensureEmbeddedConfig(repoRoot, branch); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Embedded state initialized (from local branch)\n")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Worktree: %s\n  Branch:   %s\n", worktreePath, branch)
		return nil
	}

	// Step 5: Create new orphan branch with initial state.
	title := deriveInitTitle(titleFlag, repoRoot)

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Creating state branch %q...\n", branch)

	if err := gitops.CreateOrphanBranch(repoRoot, branch); err != nil {
		return exitcode.UnexpectedErrorf("creating orphan branch: %v", err)
	}

	// Write initial state files onto the orphan branch.
	if err := writeInitialState(repoRoot, title, branch, originalBranch); err != nil {
		// Attempt to switch back to original branch on failure.
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("writing initial state: %v", err)
	}

	// Commit the initial state.
	if err := gitops.Commit(repoRoot, []string{EmbeddedStateFile, "tasks/README.md", "README.md"}, "Initialize Synchestra state"); err != nil {
		_ = gitops.CheckoutBranch(repoRoot, originalBranch)
		return exitcode.UnexpectedErrorf("committing initial state: %v", err)
	}

	// Switch back to the original branch.
	if err := gitops.CheckoutBranch(repoRoot, originalBranch); err != nil {
		return exitcode.UnexpectedErrorf("returning to branch %s: %v", originalBranch, err)
	}

	// Step 6: Create worktree.
	if err := gitops.WorktreeAdd(repoRoot, worktreePath, branch); err != nil {
		return exitcode.UnexpectedErrorf("creating worktree: %v", err)
	}

	// Step 7: Update .gitignore on the main branch.
	if err := ensureGitignoreEntry(repoRoot, worktreeDir); err != nil {
		return exitcode.UnexpectedErrorf("updating .gitignore: %v", err)
	}

	// Step 8: Write marker config on the main branch.
	if err := ensureEmbeddedConfig(repoRoot, branch); err != nil {
		return err
	}

	// Step 9: Push if requested.
	if !noPush {
		if err := gitops.PushNewBranch(repoRoot, "origin", branch); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not push state branch to remote: %v\n", err)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Run 'git push -u origin %s' later to sync.\n", branch)
		}
	}

	// Step 10: Print summary.
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Embedded state initialized\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Worktree: %s\n  Branch:   %s\n  Title:    %s\n", worktreePath, branch, title)
	return nil
}

// writeInitialState writes the initial files for the embedded state branch.
// Must be called while the orphan branch is checked out.
func writeInitialState(repoRoot, title, branch, sourceBranch string) error {
	// Write synchestra-state.yaml
	cfg := EmbeddedStateConfig{
		Title:        title,
		Mode:         "embedded",
		SourceBranch: sourceBranch,
		Sync: &EmbeddedSyncCfg{
			Pull: "on_commit",
			Push: "on_commit",
		},
	}
	if err := WriteEmbeddedStateConfig(repoRoot, cfg); err != nil {
		return fmt.Errorf("writing state config: %w", err)
	}

	// Write tasks/README.md with empty board.
	tasksDir := filepath.Join(repoRoot, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return fmt.Errorf("creating tasks directory: %w", err)
	}

	board := emptyBoard()
	if err := os.WriteFile(filepath.Join(tasksDir, "README.md"), board, 0o644); err != nil {
		return fmt.Errorf("writing task board: %w", err)
	}

	// Write root README.md (auto-generated overview).
	readme := fmt.Sprintf("# %s — Synchestra State\n\nThis branch contains Synchestra coordination state for the project.\n\nSee `tasks/README.md` for the task board.\n", title)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte(readme), 0o644); err != nil {
		return fmt.Errorf("writing README: %w", err)
	}

	return nil
}

// emptyBoard returns the markdown for an empty task status board.
func emptyBoard() []byte {
	var buf bytes.Buffer
	buf.WriteString("# Tasks\n\n")
	buf.WriteString("| Task | Status | Depends on | Branch | Agent | Requester | Time |\n")
	buf.WriteString("|---|---|---|---|---|---|---|\n")
	return buf.Bytes()
}

// isValidWorktree checks if the worktree directory exists and is a valid git worktree.
func isValidWorktree(repoRoot, worktreePath string) bool {
	info, err := os.Stat(worktreePath)
	if err != nil || !info.IsDir() {
		return false
	}
	// Check if it's listed as an active worktree.
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

// ensureGitignoreEntry adds the entry to .gitignore if not already present.
func ensureGitignoreEntry(repoRoot, entry string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	// Check if already present.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == entry {
			return nil
		}
	}

	// Append the entry.
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer f.Close()

	// Add newline before entry if file doesn't end with one.
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

// ensureEmbeddedConfig writes the synchestra.yaml marker on the main branch if not present.
func ensureEmbeddedConfig(repoRoot, branch string) error {
	path := filepath.Join(repoRoot, EmbeddedConfigFile)
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	cfg := EmbeddedConfig{
		State:       "embedded",
		StateBranch: branch,
	}
	if err := WriteEmbeddedConfig(repoRoot, cfg); err != nil {
		return exitcode.UnexpectedErrorf("writing embedded config: %v", err)
	}
	return nil
}

// deriveInitTitle derives a project title for embedded init.
func deriveInitTitle(explicit, repoRoot string) string {
	if explicit != "" {
		return explicit
	}
	// Try README heading.
	data, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err == nil {
		if h := extractFirstHeading(data); h != "" {
			return h
		}
	}
	// Fall back to directory name.
	return filepath.Base(repoRoot)
}
