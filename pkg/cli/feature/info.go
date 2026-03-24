package feature

// Features implemented: cli/feature/info
// Features depended on:  cli/feature

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/synchestra/pkg/cli/exitcode"
	"gopkg.in/yaml.v3"
)

func infoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <feature_id>",
		Short: "Show feature metadata, section TOC, and children",
		Long: `Returns structured metadata and a section table-of-contents with line
ranges for a feature's README.md, enabling agents to surgically read only
the sections they need. Default output is YAML; use --format for JSON or text.`,
		Args: cobra.ExactArgs(1),
		RunE: runInfo,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json, text")
	return cmd
}

// featureInfo is the top-level output structure for feature info.
type featureInfo struct {
	Path     string        `yaml:"path" json:"path"`
	Status   string        `yaml:"status" json:"status"`
	Deps     []string      `yaml:"deps" json:"deps"`
	Refs     []string      `yaml:"refs" json:"refs"`
	Children []childInfo   `yaml:"children,omitempty" json:"children,omitempty"`
	Plans    []string      `yaml:"plans,omitempty" json:"plans,omitempty"`
	Sections []sectionInfo `yaml:"sections" json:"sections"`
}

// childInfo represents a child sub-feature.
type childInfo struct {
	Path     string `yaml:"path" json:"path"`
	InReadme bool   `yaml:"in_readme" json:"in_readme"`
}

// sectionInfo represents a heading section in the README.
type sectionInfo struct {
	Title    string        `yaml:"title" json:"title"`
	Lines    string        `yaml:"lines" json:"lines"`
	Items    int           `yaml:"items,omitempty" json:"items,omitempty"`
	Children []sectionInfo `yaml:"children,omitempty" json:"children,omitempty"`
}

func runInfo(cmd *cobra.Command, args []string) error {
	featureID := args[0]
	projectFlag, _ := cmd.Flags().GetString("project")
	formatFlag, _ := cmd.Flags().GetString("format")

	if formatFlag != "yaml" && formatFlag != "json" && formatFlag != "text" {
		return exitcode.InvalidArgsErrorf("invalid format: %s (supported: yaml, json, text)", formatFlag)
	}

	featuresDir, err := resolveFeaturesDir(projectFlag)
	if err != nil {
		return err
	}

	if !featureExists(featuresDir, featureID) {
		return exitcode.NotFoundErrorf("feature not found: %s", featureID)
	}

	readmePath := featureReadmePath(featuresDir, featureID)

	// Extract metadata
	status, err := parseFeatureStatus(readmePath)
	if err != nil {
		return exitcode.UnexpectedErrorf("reading feature status: %v", err)
	}

	deps, err := parseDependencies(readmePath)
	if err != nil {
		return exitcode.UnexpectedErrorf("reading dependencies: %v", err)
	}

	refs, err := findFeatureRefs(featuresDir, featureID)
	if err != nil {
		return exitcode.UnexpectedErrorf("finding references: %v", err)
	}

	// Discover children
	children, err := discoverChildFeatures(featuresDir, featureID, readmePath)
	if err != nil {
		return exitcode.UnexpectedErrorf("discovering children: %v", err)
	}

	// Find linked plans
	specRoot := filepath.Dir(featuresDir) // spec/features/ -> spec/
	plans, err := findLinkedPlans(filepath.Dir(specRoot), featureID)
	if err != nil {
		return exitcode.UnexpectedErrorf("finding linked plans: %v", err)
	}

	// Parse sections
	sections, err := parseSections(readmePath)
	if err != nil {
		return exitcode.UnexpectedErrorf("parsing sections: %v", err)
	}

	info := featureInfo{
		Path:     featureID,
		Status:   status,
		Deps:     deps,
		Refs:     refs,
		Children: children,
		Plans:    plans,
		Sections: sections,
	}

	return writeFeatureInfo(cmd.OutOrStdout(), formatFlag, info)
}

// writeFeatureInfo encodes info to w in the given format (yaml, json, or text).
func writeFeatureInfo(w io.Writer, formatFlag string, info featureInfo) error {
	switch formatFlag {
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(info); err != nil {
			return exitcode.UnexpectedErrorf("encoding yaml: %v", err)
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

// writeTextInfo writes the feature info in a human-readable text format.
func writeTextInfo(w io.Writer, info featureInfo) error {
	bw := bufio.NewWriter(w)

	_, _ = fmt.Fprintf(bw, "Feature: %s\n", info.Path)
	_, _ = fmt.Fprintf(bw, "Status:  %s\n", info.Status)

	if len(info.Deps) > 0 {
		_, _ = fmt.Fprintf(bw, "Deps:    %s\n", strings.Join(info.Deps, ", "))
	} else {
		_, _ = fmt.Fprintln(bw, "Deps:    (none)")
	}

	if len(info.Refs) > 0 {
		_, _ = fmt.Fprintf(bw, "Refs:    %s\n", strings.Join(info.Refs, ", "))
	} else {
		_, _ = fmt.Fprintln(bw, "Refs:    (none)")
	}

	if len(info.Children) > 0 {
		_, _ = fmt.Fprintln(bw, "\nChildren:")
		for _, c := range info.Children {
			marker := "✓"
			if !c.InReadme {
				marker = "✗"
			}
			_, _ = fmt.Fprintf(bw, "  %s %s (in_readme: %v)\n", marker, c.Path, c.InReadme)
		}
	}

	if len(info.Plans) > 0 {
		_, _ = fmt.Fprintf(bw, "\nPlans:   %s\n", strings.Join(info.Plans, ", "))
	}

	if len(info.Sections) > 0 {
		_, _ = fmt.Fprintln(bw, "\nSections:")
		printTextSections(bw, info.Sections, 1)
	}

	return bw.Flush()
}

func printTextSections(w *bufio.Writer, sections []sectionInfo, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, s := range sections {
		itemsSuffix := ""
		if s.Items > 0 {
			itemsSuffix = fmt.Sprintf(" (%d items)", s.Items)
		}
		_, _ = fmt.Fprintf(w, "%s%s [%s]%s\n", indent, s.Title, s.Lines, itemsSuffix)
		if len(s.Children) > 0 {
			printTextSections(w, s.Children, depth+1)
		}
	}
}

// parseFeatureStatus extracts the status from a feature README.
// Looks for patterns like "**Status:** Conceptual" or "Status: Implemented".
func parseFeatureStatus(readmePath string) (string, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }() // opened read-only; Close never flushes data so its error is irrelevant

	statusRe := regexp.MustCompile(`^\*?\*?Status:?\*?\*?\s*(.+)$`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match "**Status:** value" or "Status: value"
		if m := statusRe.FindStringSubmatch(line); m != nil {
			status := strings.TrimSpace(m[1])
			// Strip surrounding quotes or backticks
			status = strings.Trim(status, "`\"'")
			return status, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "Unknown", nil
}

// findFeatureRefs finds all features that reference the given featureID as a dependency.
// This is the same logic as the refs command but returns a slice.
func findFeatureRefs(featuresDir, featureID string) ([]string, error) {
	allFeatures, err := discoverFeatures(featuresDir)
	if err != nil {
		return nil, err
	}

	var refs []string
	for _, fID := range allFeatures {
		if fID == featureID {
			continue
		}
		readmePath := featureReadmePath(featuresDir, fID)
		deps, err := parseDependencies(readmePath)
		if err != nil {
			continue
		}
		for _, dep := range deps {
			if dep == featureID {
				refs = append(refs, fID)
				break
			}
		}
	}
	sort.Strings(refs)
	return refs, nil
}

// discoverChildFeatures finds immediate child sub-feature directories and checks
// whether each is listed in the parent's ## Contents table.
func discoverChildFeatures(featuresDir, featureID, readmePath string) ([]childInfo, error) {
	featureDir := filepath.Join(featuresDir, filepath.FromSlash(featureID))
	entries, err := os.ReadDir(featureDir)
	if err != nil {
		return nil, err
	}

	// Parse the Contents table from the parent README
	contentsEntries, err := parseContentsTable(readmePath)
	if err != nil {
		return nil, err
	}

	var children []childInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip reserved directories
		if strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		childReadme := filepath.Join(featureDir, entry.Name(), "README.md")
		if _, err := os.Stat(childReadme); err != nil {
			continue
		}
		childPath := featureID + "/" + entry.Name()
		inReadme := contentsEntries[entry.Name()]
		children = append(children, childInfo{
			Path:     childPath,
			InReadme: inReadme,
		})
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Path < children[j].Path
	})
	return children, nil
}

// parseContentsTable reads a README and extracts entries from the ## Contents section.
// Returns a map of directory names that appear in the Contents table.
func parseContentsTable(readmePath string) (map[string]bool, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // opened read-only; Close never flushes data so its error is irrelevant

	entries := make(map[string]bool)
	inContents := false
	scanner := bufio.NewScanner(f)

	// Match markdown links in table rows: | [Name](dir/README.md) | ... |
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "## Contents" {
			inContents = true
			continue
		}
		if inContents && strings.HasPrefix(line, "## ") {
			break
		}
		if inContents && strings.HasPrefix(line, "|") {
			matches := linkRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				linkPath := m[2]
				// Extract directory name from link path like "subdir/README.md"
				dir := strings.TrimSuffix(linkPath, "/README.md")
				dir = strings.TrimSuffix(dir, "/readme.md")
				// Handle relative paths like "./subdir/README.md"
				dir = strings.TrimPrefix(dir, "./")
				// Only take the first path segment (immediate child)
				if parts := strings.SplitN(dir, "/", 2); len(parts) > 0 && parts[0] != "" {
					entries[parts[0]] = true
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// findLinkedPlans scans spec/plans/*/README.md for plans that reference the given feature.
// It looks in the **Features:** section for markdown links to the feature path.
func findLinkedPlans(repoRoot, featureID string) ([]string, error) {
	plansDir := filepath.Join(repoRoot, "spec", "plans")
	if _, err := os.Stat(plansDir); err != nil {
		return nil, nil // no plans directory = no linked plans
	}

	var plans []string
	err := filepath.WalkDir(plansDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "README.md" {
			return nil
		}
		// Get the plan directory name
		planDir := filepath.Dir(path)
		if planDir == plansDir {
			return nil // skip the plans index README itself
		}
		planName := filepath.Base(planDir)

		if planReferencesFeature(path, featureID) {
			plans = append(plans, planName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(plans)
	return plans, nil
}

// planReferencesFeature checks if a plan README references the given feature
// in its **Features:** section.
func planReferencesFeature(planReadmePath, featureID string) bool {
	f, err := os.Open(planReadmePath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }() // opened read-only; Close never flushes data so its error is irrelevant

	inFeatures := false
	scanner := bufio.NewScanner(f)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	featureSuffix := "features/" + featureID + "/README.md"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "**Features:**") {
			inFeatures = true
			continue
		}
		// End of features section: next metadata field or blank line after entries
		if inFeatures && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, " ") && line != "" {
			break
		}
		if inFeatures {
			matches := linkRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				linkPath := m[2]
				if strings.HasSuffix(linkPath, featureSuffix) {
					return true
				}
			}
		}
	}
	return false
}

// parseSections reads a README and builds a section TOC from markdown headings.
// Supports h2 and h3 nesting. Counts list items (lines starting with "- ")
// within each section.
func parseSections(readmePath string) ([]sectionInfo, error) {
	f, err := os.Open(readmePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // opened read-only; Close never flushes data so its error is irrelevant

	type rawSection struct {
		title     string
		level     int
		startLine int
		endLine   int
		items     int
	}

	var raw []rawSection
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect headings
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			level := 2
			title := strings.TrimPrefix(trimmed, "## ")
			if strings.HasPrefix(trimmed, "### ") {
				level = 3
				title = strings.TrimPrefix(trimmed, "### ")
			}
			title = strings.TrimSpace(title)

			// Close previous section's endLine
			if len(raw) > 0 {
				raw[len(raw)-1].endLine = lineNum - 1
			}

			raw = append(raw, rawSection{
				title:     title,
				level:     level,
				startLine: lineNum,
			})
			continue
		}

		// Count list items in current section
		if len(raw) > 0 && strings.HasPrefix(trimmed, "- ") {
			raw[len(raw)-1].items++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Close last section
	if len(raw) > 0 {
		raw[len(raw)-1].endLine = lineNum
	}

	// Skip the h1 title (level 1) — we only handle h2 and h3
	// Build nested structure: h3 sections are children of the preceding h2
	var sections []sectionInfo
	for i := 0; i < len(raw); i++ {
		s := raw[i]
		if s.level == 2 {
			section := sectionInfo{
				Title: s.title,
				Lines: fmt.Sprintf("%d-%d", s.startLine, s.endLine),
				Items: s.items,
			}
			// Collect h3 children
			for j := i + 1; j < len(raw) && raw[j].level == 3; j++ {
				child := sectionInfo{
					Title: raw[j].title,
					Lines: fmt.Sprintf("%d-%d", raw[j].startLine, raw[j].endLine),
					Items: raw[j].items,
				}
				section.Children = append(section.Children, child)
			}
			sections = append(sections, section)
		}
	}

	return sections, nil
}
