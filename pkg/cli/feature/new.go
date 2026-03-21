package feature

// Features implemented: cli/feature/new
// Features depended on:  cli/feature, cli/feature/info, project-definition

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchesta-io/synchestra/pkg/cli/gitops"
	"gopkg.in/yaml.v3"
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

	// Step 1: Validate arguments
	if title == "" {
		return &exitError{code: 2, msg: "missing required flag: --title"}
	}
	if formatFlag != "yaml" && formatFlag != "json" && formatFlag != "text" {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid format: %s (supported: yaml, json, text)", formatFlag)}
	}
	if !isValidStatus(statusFlag) {
		return &exitError{code: 2, msg: fmt.Sprintf("invalid status: %s (supported: draft, approved, implemented)", statusFlag)}
	}
	if pushFlag {
		commitFlag = true
	}

	// Step 2: Generate or validate slug
	slug := slugFlag
	if slug == "" {
		slug = generateSlug(title)
	} else {
		if err := validateSlug(slug); err != nil {
			return &exitError{code: 2, msg: fmt.Sprintf("invalid slug: %v", err)}
		}
	}

	// Validate --parent and slash-in-slug mutual exclusion
	if parentFlag != "" && strings.Contains(slug, "/") {
		return &exitError{code: 2, msg: "cannot use --parent with a slug containing slashes; use one or the other"}
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

	// Validate --depends-on feature IDs exist
	for _, dep := range deps {
		if !featureExists(featuresDir, dep) {
			return &exitError{code: 2, msg: fmt.Sprintf("dependency feature not found: %s", dep)}
		}
	}

	// Step 3: Resolve full feature path
	var featureID string
	var parentID string

	switch {
	case parentFlag != "":
		featureID = parentFlag + "/" + slug
		parentID = parentFlag
	case strings.Contains(slug, "/"):
		featureID = slug
		parts := strings.Split(slug, "/")
		parentID = strings.Join(parts[:len(parts)-1], "/")
	default:
		featureID = slug
	}

	featureDir := filepath.Join(featuresDir, filepath.FromSlash(featureID))

	// Step 4: Validate parent exists (for sub-features)
	if parentID != "" {
		if !featureExists(featuresDir, parentID) {
			return &exitError{code: 3, msg: fmt.Sprintf("parent feature not found: %s", parentID)}
		}
	}

	// Step 5: Verify target doesn't exist
	if _, err := os.Stat(featureDir); err == nil {
		return &exitError{code: 4, msg: fmt.Sprintf("feature already exists at: %s", featureID)}
	}

	// Step 6: Create feature directory and README
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("creating feature directory: %v", err)}
	}

	readme := generateReadme(title, statusFlag, descFlag, deps)
	readmePath := filepath.Join(featureDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("writing README.md: %v", err)}
	}

	// Track files changed for git commit
	changedFiles := []string{readmePath}

	// Step 7: Update parent's Contents section (sub-features)
	if parentID != "" {
		parentReadme := featureReadmePath(featuresDir, parentID)
		changed, err := updateParentContents(parentReadme, filepath.Base(featureDir), descFlag)
		if err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("updating parent contents: %v", err)}
		}
		if changed {
			changedFiles = append(changedFiles, parentReadme)
		}
	}

	// Step 8: Update feature index for top-level features
	if parentID == "" {
		indexPath := filepath.Join(featuresDir, "README.md")
		changed, err := updateFeatureIndex(indexPath, featureID, descFlag)
		if err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("updating feature index: %v", err)}
		}
		if changed {
			changedFiles = append(changedFiles, indexPath)
		}
	}

	// Step 9-10: Git operations
	if commitFlag {
		repoRoot := filepath.Dir(filepath.Dir(featuresDir)) // spec/features/ → repo root
		if !gitops.IsGitRepo(repoRoot) {
			return &exitError{code: 10, msg: "not a git repository; cannot commit"}
		}

		// Make paths relative to repo root for git
		relFiles := make([]string, 0, len(changedFiles))
		for _, f := range changedFiles {
			rel, err := filepath.Rel(repoRoot, f)
			if err != nil {
				rel = f
			}
			relFiles = append(relFiles, rel)
		}

		commitMsg := fmt.Sprintf("feat(spec): add feature %s", featureID)

		if pushFlag {
			if err := gitops.CommitAndPush(repoRoot, relFiles, commitMsg); err != nil {
				return &exitError{code: 1, msg: fmt.Sprintf("commit and push failed: %v", err)}
			}
		} else {
			if err := gitCommitOnly(repoRoot, relFiles, commitMsg); err != nil {
				return &exitError{code: 10, msg: fmt.Sprintf("commit failed: %v", err)}
			}
		}
	}

	// Output: same as feature info
	info, err := buildNewFeatureInfo(featuresDir, featureID, readmePath, statusFlag, deps)
	if err != nil {
		return &exitError{code: 10, msg: fmt.Sprintf("building output: %v", err)}
	}

	w := cmd.OutOrStdout()
	switch formatFlag {
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(info); err != nil {
			return &exitError{code: 10, msg: fmt.Sprintf("encoding yaml: %v", err)}
		}
		return enc.Close()
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	case "text":
		return writeTextInfo(w, info)
	}
	return nil
}

// buildNewFeatureInfo constructs a featureInfo for the newly created feature.
func buildNewFeatureInfo(featuresDir, featureID, readmePath, status string, deps []string) (featureInfo, error) {
	sections, err := parseSections(readmePath)
	if err != nil {
		return featureInfo{}, err
	}

	if deps == nil {
		deps = []string{}
	}

	return featureInfo{
		Path:     featureID,
		Status:   status,
		Deps:     deps,
		Refs:     []string{},
		Children: nil,
		Plans:    nil,
		Sections: sections,
	}, nil
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

// updateParentContents adds or updates the ## Contents section in the parent's README.
// Returns true if the file was modified.
func updateParentContents(parentReadmePath, childSlug, description string) (bool, error) {
	content, err := os.ReadFile(parentReadmePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	desc := description
	if desc == "" {
		desc = "TODO: Add description."
	}

	newRow := fmt.Sprintf("| [%s](%s/README.md) | %s |", childSlug, childSlug, desc)

	// Find ## Contents section
	contentsIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Contents" {
			contentsIdx = i
			break
		}
	}

	if contentsIdx >= 0 {
		// Find the end of the table (next ## or end of section)
		insertIdx := contentsIdx + 1
		for insertIdx < len(lines) {
			trimmed := strings.TrimSpace(lines[insertIdx])
			if strings.HasPrefix(trimmed, "## ") && trimmed != "## Contents" {
				break
			}
			insertIdx++
		}
		// Back up past trailing blank lines
		for insertIdx > contentsIdx+1 && strings.TrimSpace(lines[insertIdx-1]) == "" {
			insertIdx--
		}
		// Insert the new row
		lines = append(lines[:insertIdx+1], lines[insertIdx:]...)
		lines[insertIdx] = newRow
	} else {
		// Create ## Contents section after ## Summary
		summaryIdx := -1
		for i, line := range lines {
			if strings.TrimSpace(line) == "## Summary" {
				summaryIdx = i
				break
			}
		}

		insertAfter := 0
		if summaryIdx >= 0 {
			// Find end of Summary section
			insertAfter = summaryIdx + 1
			for insertAfter < len(lines) {
				trimmed := strings.TrimSpace(lines[insertAfter])
				if strings.HasPrefix(trimmed, "## ") {
					break
				}
				insertAfter++
			}
		}

		contentsBlock := []string{
			"## Contents",
			"",
			"| Child | Description |",
			"|---|---|",
			newRow,
			"",
		}

		newLines := make([]string, 0, len(lines)+len(contentsBlock))
		newLines = append(newLines, lines[:insertAfter]...)
		newLines = append(newLines, contentsBlock...)
		newLines = append(newLines, lines[insertAfter:]...)
		lines = newLines
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(parentReadmePath, []byte(result), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// updateFeatureIndex adds a new row to the feature index at spec/features/README.md.
// Returns true if the file was modified.
func updateFeatureIndex(indexPath, featureID, description string) (bool, error) {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No index file — skip silently
		}
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	desc := description
	if desc == "" {
		desc = "TODO: Add description."
	}

	newRow := fmt.Sprintf("| [%s](%s/README.md) | %s |", featureID, featureID, desc)

	// Find the Contents table (look for a markdown table with links)
	tableEnd := -1
	inTable := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "|") {
			inTable = true
			tableEnd = i
		} else if inTable && trimmed == "" {
			break
		} else if inTable && strings.HasPrefix(trimmed, "## ") {
			break
		}
	}

	if tableEnd >= 0 {
		// Insert after the last table row
		insertIdx := tableEnd + 1
		lines = append(lines[:insertIdx+1], lines[insertIdx:]...)
		lines[insertIdx] = newRow
	} else {
		// Append to end
		lines = append(lines, "", newRow)
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(indexPath, []byte(result), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
