package sourceref

// Features implemented: cli/code/deps
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/code/deps

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ScanResult represents the references found in a set of files.
type ScanResult struct {
	// FileRefs maps file path to list of references found in that file
	FileRefs map[string][]*Reference
}

// ScanFiles scans a list of files for Synchestra source references.
// Returns a ScanResult with all references grouped by file.
func ScanFiles(filePaths []string) (*ScanResult, error) {
	result := &ScanResult{
		FileRefs: make(map[string][]*Reference),
	}

	for _, filePath := range filePaths {
		refs, err := scanFile(filePath)
		if err != nil {
			// Skip files that can't be read (binary, permission denied, etc.)
			continue
		}
		if len(refs) > 0 {
			result.FileRefs[filePath] = refs
		}
	}

	return result, nil
}

// scanFile scans a single file for references and returns deduplicated, sorted references.
func scanFile(filePath string) ([]*Reference, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	// Map to track unique references (by resolved path + cross-repo suffix)
	seen := make(map[string]bool)
	var refs []*Reference

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		ref := ScanLine(line)
		if ref != nil {
			// Create a unique key for deduplication within a single file
			key := ref.ResolvedPath + ref.CrossRepoSuffix
			if !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Sort references alphabetically by resolved path + cross-repo suffix
	sort.Slice(refs, func(i, j int) bool {
		keyI := refs[i].ResolvedPath + refs[i].CrossRepoSuffix
		keyJ := refs[j].ResolvedPath + refs[j].CrossRepoSuffix
		return keyI < keyJ
	})

	return refs, nil
}

// ExpandGlobPattern expands a glob pattern to a list of file paths.
// Returns sorted file paths.
func ExpandGlobPattern(pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "**/*"
	}

	// Validate glob pattern first
	if _, err := filepath.Match(pattern, "test"); err != nil && pattern != "**" && pattern != "**/*" {
		// Try to validate with a test path
		_, err := filepath.Match(pattern, "")
		if err != nil {
			return nil, err
		}
	}

	// Use filepath.Walk for simple patterns, or custom logic for ** patterns
	var matches []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}
		if info.IsDir() {
			return nil
		}

		// Normalize path to use forward slashes
		normalPath := filepath.ToSlash(path)
		// Remove leading ./
		if normalPath == "."+string(filepath.Separator) {
			normalPath = normalPath[2:]
		} else if normalPath[0:2] == "./" {
			normalPath = normalPath[2:]
		}

		// Simple glob matching (handles * and ** patterns)
		ok, err := matchGlobPattern(normalPath, pattern)
		if err != nil {
			return nil
		}
		if ok {
			matches = append(matches, normalPath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(matches)
	return matches, nil
}

// matchGlobPattern matches a file path against a glob pattern.
// Supports * (matches within a path segment) and ** (matches across segments).
func matchGlobPattern(path string, pattern string) (bool, error) {
	// Handle ** patterns (match any number of directories)
	if pattern == "**/*" || pattern == "**" {
		return true, nil
	}

	// Use filepath.Match for simple patterns
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false, err
	}
	return matched, nil
}

// GetUniqueReferences extracts unique references from a ScanResult, optionally filtered by type.
// Returns references sorted by (resolved_path, cross_repo_suffix).
func GetUniqueReferences(result *ScanResult, typeFilter string) []*Reference {
	seen := make(map[string]*Reference)

	for _, refs := range result.FileRefs {
		for _, ref := range refs {
			// Apply type filter if specified
			if typeFilter != "" && ref.Type != typeFilter {
				continue
			}

			key := ref.ResolvedPath + ref.CrossRepoSuffix
			if _, exists := seen[key]; !exists {
				seen[key] = ref
			}
		}
	}

	// Convert to sorted slice
	var unique []*Reference
	for _, ref := range seen {
		unique = append(unique, ref)
	}

	sort.Slice(unique, func(i, j int) bool {
		keyI := unique[i].ResolvedPath + unique[i].CrossRepoSuffix
		keyJ := unique[j].ResolvedPath + unique[j].CrossRepoSuffix
		return keyI < keyJ
	})

	return unique
}

// FormatOutput formats the scan results for output.
// If singleFile is true, returns a flat list. Otherwise, groups by file with headers.
func FormatOutput(result *ScanResult, singleFile bool, typeFilter string) string {
	if len(result.FileRefs) == 0 {
		return ""
	}

	var output []string

	if singleFile {
		// Flat list of references (no file headers)
		refs := GetUniqueReferences(result, typeFilter)
		for _, ref := range refs {
			output = append(output, ref.ResolvedPath+ref.CrossRepoSuffix)
		}
	} else {
		// Group by file with headers
		fileNames := make([]string, 0, len(result.FileRefs))
		for fname := range result.FileRefs {
			fileNames = append(fileNames, fname)
		}
		sort.Strings(fileNames)

		for i, fname := range fileNames {
			if i > 0 {
				output = append(output, "") // Blank line between files
			}
			output = append(output, fname)

			// Get references for this file, applying type filter
			refs := result.FileRefs[fname]
			filtered := refs
			if typeFilter != "" {
				filtered = nil
				for _, ref := range refs {
					if ref.Type == typeFilter {
						filtered = append(filtered, ref)
					}
				}
			}

			for _, ref := range filtered {
				output = append(output, "  "+ref.ResolvedPath+ref.CrossRepoSuffix)
			}
		}
	}

	if len(output) == 0 {
		return ""
	}

	return fmt.Sprintf("%s\n", join(output))
}

// join helper function to join strings with newlines
func join(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += "\n"
		}
		result += s
	}
	return result
}
