package feature

// Features implemented: cli/feature/new
// Features depended on:  cli/feature, cli/feature/info, project-definition

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/specscore/pkg/feature"
	"github.com/synchestra-io/synchestra/pkg/cli/gitops"
)

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new feature directory with a README template",
		Long: `Creates a new feature directory with a README containing all required
sections (Summary, Problem, Behavior, Acceptance Criteria, Outstanding
Questions). Changes are local by default; use --commit to create a git
commit, or --push for atomic commit-and-push.`,
		RunE: runNew,
	}
	cmd.Flags().String("title", "", "human-readable feature title (required)")
	cmd.Flags().String("slug", "", "feature slug (directory name); auto-generated from title if omitted")
	cmd.Flags().String("parent", "", "parent feature ID for creating a sub-feature")
	cmd.Flags().String("status", "draft", "initial feature status: draft, approved, implemented")
	cmd.Flags().String("description", "", "short description placed in the Summary section")
	cmd.Flags().String("depends-on", "", "comma-separated list of feature IDs this feature depends on")
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json, text")
	cmd.Flags().Bool("commit", false, "create a git commit with the changes")
	cmd.Flags().Bool("push", false, "commit and push atomically (implies --commit)")
	return cmd
}

func runNew(cmd *cobra.Command, _ []string) error {
	title, _ := cmd.Flags().GetString("title")
	slugFlag, _ := cmd.Flags().GetString("slug")
	parentFlag, _ := cmd.Flags().GetString("parent")
	statusFlag, _ := cmd.Flags().GetString("status")
	descFlag, _ := cmd.Flags().GetString("description")
	depsFlag, _ := cmd.Flags().GetString("depends-on")
	projectFlag, _ := cmd.Flags().GetString("project")
	formatFlag, _ := cmd.Flags().GetString("format")
	commitFlag, _ := cmd.Flags().GetBool("commit")
	pushFlag, _ := cmd.Flags().GetBool("push")

	if formatFlag != "yaml" && formatFlag != "json" && formatFlag != "text" {
		return exitcode.InvalidArgsErrorf("invalid format: %s (supported: yaml, json, text)", formatFlag)
	}
	if pushFlag {
		commitFlag = true
	}

	// Parse --depends-on
	var deps []string
	if depsFlag != "" {
		for _, d := range strings.Split(depsFlag, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				deps = append(deps, d)
			}
		}
	}

	// Resolve features directory
	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	// Delegate to specscore's feature.New
	result, err := feature.New(featuresDir, feature.NewOptions{
		Title:       title,
		Slug:        slugFlag,
		Parent:      parentFlag,
		Status:      statusFlag,
		Description: descFlag,
		DependsOn:   deps,
	})
	if err != nil {
		return err
	}

	// Git operations (CLI-specific)
	if commitFlag {
		repoRoot := filepath.Dir(filepath.Dir(featuresDir)) // spec/features/ → repo root
		if !gitops.IsGitRepo(repoRoot) {
			return exitcode.UnexpectedError("not a git repository; cannot commit")
		}

		// Make paths relative to repo root for git
		relFiles := make([]string, 0, len(result.ChangedFiles))
		for _, f := range result.ChangedFiles {
			rel, relErr := filepath.Rel(repoRoot, f)
			if relErr != nil {
				rel = f
			}
			relFiles = append(relFiles, rel)
		}

		commitMsg := fmt.Sprintf("feat(spec): add feature %s", result.FeatureID)

		if pushFlag {
			if err := gitops.CommitAndPush(repoRoot, relFiles, commitMsg); err != nil {
				return exitcode.ConflictErrorf("commit and push failed: %v", err)
			}
		} else {
			if err := gitCommitOnly(repoRoot, relFiles, commitMsg); err != nil {
				return exitcode.UnexpectedErrorf("commit failed: %v", err)
			}
		}
	}

	// Output: same as feature info
	return writeFeatureInfo(cmd.OutOrStdout(), formatFlag, result.Info)
}

// gitCommitOnly stages files and creates a commit without pushing.
func gitCommitOnly(repoDir string, files []string, message string) error {
	addArgs := append([]string{"-C", repoDir, "add"}, files...)
	cmd := exec.Command("git", addArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}

	cmd = exec.Command("git", "-C", repoDir, "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	return nil
}
