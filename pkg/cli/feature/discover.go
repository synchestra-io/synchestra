package feature

// Features implemented: cli/feature/list, cli/feature/tree, cli/feature/deps, cli/feature/refs
// Features depended on:  feature (spec/features/feature)

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// featuresDir returns the default features directory name.
// In the future this could be read from project config (project_dirs.specifications).
const defaultSpecDir = "spec"
const featuresSubDir = "features"

// exitError is an error that carries an exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// ExitCode returns the exit code for the error.
func (e *exitError) ExitCode() int { return e.code }

// findSpecRepoRoot walks up from startDir looking for synchestra-spec-repo.yaml.
// As a fallback, it also checks for a spec/features/ directory (for repos that
// are themselves spec repos but don't yet have the config file).
// Returns the directory that serves as the spec repo root.
func findSpecRepoRoot(startDir string) (string, error) {
	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		// Primary: look for explicit config file
		configPath := filepath.Join(current, "synchestra-spec-repo.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return current, nil
		}

		// Fallback: look for spec/features/ directory (the repo is itself a spec repo)
		featPath := filepath.Join(current, defaultSpecDir, featuresSubDir)
		if info, err := os.Stat(featPath); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", &exitError{code: 3, msg: "project not found: no synchestra-spec-repo.yaml or spec/features/ in any parent directory"}
		}
		current = parent
	}
}

// resolveFeaturesDir returns the absolute path to the features directory.
// It finds the spec repo root from CWD, then appends spec/features/.
func resolveFeaturesDir(projectFlag string) (string, error) {
	if projectFlag != "" {
		return "", &exitError{code: 2, msg: "--project with project lookup is not yet implemented; run from within a spec repo directory"}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", &exitError{code: 10, msg: fmt.Sprintf("cannot determine working directory: %v", err)}
	}

	root, err := findSpecRepoRoot(cwd)
	if err != nil {
		return "", err
	}

	featDir := filepath.Join(root, defaultSpecDir, featuresSubDir)
	info, err := os.Stat(featDir)
	if err != nil || !info.IsDir() {
		return "", &exitError{code: 3, msg: fmt.Sprintf("features directory not found: %s", featDir)}
	}

	return featDir, nil
}

// discoverFeatures walks the features directory and returns all feature IDs
// sorted alphabetically. A feature is any directory containing a README.md.
// Directories prefixed with _ are skipped (reserved for tooling).
func discoverFeatures(featuresDir string) ([]string, error) {
	var features []string

	err := filepath.WalkDir(featuresDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		// Skip the root features directory itself
		if path == featuresDir {
			return nil
		}
		// Skip reserved directories (prefixed with _)
		if strings.HasPrefix(d.Name(), "_") {
			return filepath.SkipDir
		}
		// Check if directory contains a README.md
		readmePath := filepath.Join(path, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			relPath, _ := filepath.Rel(featuresDir, path)
			featureID := filepath.ToSlash(relPath)
			features = append(features, featureID)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking features directory: %w", err)
	}

	sort.Strings(features)
	return features, nil
}

// featureNode represents a feature in a tree structure.
type featureNode struct {
	name     string
	children []*featureNode
}

// buildTree builds a tree from a sorted list of feature IDs.
func buildTree(featureIDs []string) []*featureNode {
	roots := make([]*featureNode, 0)
	nodeMap := make(map[string]*featureNode)

	for _, id := range featureIDs {
		parts := strings.Split(id, "/")
		name := parts[len(parts)-1]
		node := &featureNode{name: name}
		nodeMap[id] = node

		if len(parts) == 1 {
			roots = append(roots, node)
		} else {
			parentID := strings.Join(parts[:len(parts)-1], "/")
			if parent, ok := nodeMap[parentID]; ok {
				parent.children = append(parent.children, node)
			} else {
				// Parent not a feature — treat as root
				roots = append(roots, node)
			}
		}
	}

	return roots
}

// printTree writes the tree to w with tab indentation.
func printTree(w *strings.Builder, nodes []*featureNode, depth int) {
	for _, node := range nodes {
		for i := 0; i < depth; i++ {
			w.WriteByte('\t')
		}
		w.WriteString(node.name)
		w.WriteByte('\n')
		printTree(w, node.children, depth+1)
	}
}

// parseDependencies reads a feature's README.md and extracts the ## Dependencies section.
// Returns a sorted list of feature IDs listed as bullet items.
// Supports two formats:
//   - bare ID:      "- claim-and-push"
//   - markdown link: "- [Name](../path/README.md) — optional description"
//     where the feature ID is extracted from the relative link path.
func parseDependencies(readmePath string) ([]string, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var deps []string
	inDeps := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "## Dependencies" {
			inDeps = true
			continue
		}
		if inDeps && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inDeps && strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if item == "" {
				continue
			}
			dep := extractFeatureID(item)
			if dep != "" {
				deps = append(deps, dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Strings(deps)
	return deps, nil
}

// extractFeatureID extracts a feature ID from a dependency list item.
// Handles:
//   - bare ID: "claim-and-push" → "claim-and-push"
//   - markdown link: "[Name](../path/README.md)" → feature ID derived from path
//   - markdown link with description: "[Name](../path/README.md) — description" → same
func extractFeatureID(item string) string {
	// Check for markdown link: [text](url)
	if strings.HasPrefix(item, "[") {
		closeBracket := strings.Index(item, "](")
		if closeBracket == -1 {
			return item // not a valid link, treat as bare ID
		}
		rest := item[closeBracket+2:]
		closeParen := strings.Index(rest, ")")
		if closeParen == -1 {
			return item
		}
		linkPath := rest[:closeParen]
		return featureIDFromRelativePath(linkPath)
	}
	// Bare ID — strip any trailing description after " —" or " -"
	if idx := strings.Index(item, " —"); idx != -1 {
		item = strings.TrimSpace(item[:idx])
	} else if idx := strings.Index(item, " - "); idx != -1 {
		item = strings.TrimSpace(item[:idx])
	}
	return item
}

// featureIDFromRelativePath converts a relative path like "../cli/README.md"
// to a feature ID like "cli". It strips the leading "../" segments and
// trailing "/README.md".
func featureIDFromRelativePath(relPath string) string {
	// Normalize to forward slashes
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	// Strip trailing /README.md
	relPath = strings.TrimSuffix(relPath, "/README.md")
	relPath = strings.TrimSuffix(relPath, "/readme.md")
	// Strip leading ../ segments
	parts := strings.Split(relPath, "/")
	var clean []string
	for _, p := range parts {
		if p == ".." || p == "." || p == "" {
			continue
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, "/")
}

// featureExists checks if a feature ID corresponds to a valid feature directory with README.md.
func featureExists(featuresDir, featureID string) bool {
	readmePath := filepath.Join(featuresDir, filepath.FromSlash(featureID), "README.md")
	_, err := os.Stat(readmePath)
	return err == nil
}

// featureReadmePath returns the absolute path to a feature's README.md.
func featureReadmePath(featuresDir, featureID string) string {
	return filepath.Join(featuresDir, filepath.FromSlash(featureID), "README.md")
}
