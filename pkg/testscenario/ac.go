package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ACResolver resolves AC files from a spec repository root.
type ACResolver struct {
	specRoot string
}

// NewACResolver creates a new ACResolver rooted at specRoot.
func NewACResolver(specRoot string) *ACResolver {
	return &ACResolver{specRoot: specRoot}
}

// Resolve resolves ACs for the given featurePath and selector.
// selector may be "*" for all ACs or a comma-separated list of slugs
// (optionally in markdown link syntax).
func (r *ACResolver) Resolve(featurePath, selector string) ([]ACFile, error) {
	acsDir := filepath.Join(r.specRoot, "features", filepath.FromSlash(featurePath), "_acs")
	if selector == "*" {
		return r.resolveAll(acsDir)
	}
	return r.resolveSpecific(acsDir, selector)
}

func (r *ACResolver) resolveAll(acsDir string) ([]ACFile, error) {
	entries, err := os.ReadDir(acsDir)
	if err != nil {
		return nil, fmt.Errorf("reading acs directory %s: %w", acsDir, err)
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" {
			continue
		}
		slugs = append(slugs, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(slugs)
	var acs []ACFile
	for _, slug := range slugs {
		ac, err := r.readACFile(acsDir, slug)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (r *ACResolver) resolveSpecific(acsDir, selector string) ([]ACFile, error) {
	slugs := strings.Split(selector, ",")
	var acs []ACFile
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		// Strip markdown link syntax: [slug](path) → slug
		if idx := strings.Index(slug, "]"); idx > 0 && slug[0] == '[' {
			slug = slug[1:idx]
		}
		ac, err := r.readACFile(acsDir, slug)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (r *ACResolver) readACFile(acsDir, slug string) (ACFile, error) {
	path := filepath.Join(acsDir, slug+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ACFile{}, fmt.Errorf("reading AC file %s: %w", path, err)
	}
	return ParseACFile(data, slug)
}

// ParseACFile parses an AC markdown file with the given slug.
// The expected format is:
//
//	# AC: <slug>
//
//	**Status:** <status>
//	**Feature:** [feature/path](../README.md)
//
//	## Description
//	<description text>
//
//	## Inputs
//	| Name | Required | Description |
//	|---|---|---|
//	| <name> | Yes/No | <desc> |
//
//	## Verification
//	```<language>
//	<script>
//	```
func ParseACFile(data []byte, slug string) (ACFile, error) {
	text := string(data)
	lines := strings.Split(text, "\n")

	ac := ACFile{Slug: slug}

	// Find H1 line.
	h1LineIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# AC:") || strings.HasPrefix(line, "# ") {
			h1LineIdx = i
			break
		}
	}
	if h1LineIdx < 0 {
		return ACFile{}, fmt.Errorf("missing '# AC:' heading")
	}

	// Parse metadata lines between H1 and first ## heading.
	for i := h1LineIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.HasPrefix(line, "**Status:**") {
			ac.Status = strings.TrimSpace(strings.TrimPrefix(line, "**Status:**"))
		} else if strings.HasPrefix(line, "**Feature:**") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "**Feature:**"))
			// Extract text from markdown link [text](url) → text
			linkText, _ := parseMarkdownLink(raw)
			if linkText != "" {
				ac.FeaturePath = linkText
			} else {
				ac.FeaturePath = raw
			}
		}
	}

	// Split the rest (after H1) into ## sections.
	restAfterH1 := strings.Join(lines[h1LineIdx+1:], "\n")
	sections := splitIntoSections(restAfterH1)

	// sections[0] is the header block (already parsed above).
	// Process remaining sections.
	for _, sectionText := range sections[1:] {
		sectionLines := strings.Split(sectionText, "\n")
		heading := strings.TrimSpace(sectionLines[0])
		body := strings.Join(sectionLines[1:], "\n")

		switch heading {
		case "Description":
			ac.Description = strings.TrimSpace(body)
		case "Inputs":
			inputs, err := parseACInputsTable(body)
			if err != nil {
				return ACFile{}, fmt.Errorf("inputs section: %w", err)
			}
			ac.Inputs = inputs
		case "Verification":
			code, lang, err := extractCodeBlock(body)
			if err != nil {
				return ACFile{}, fmt.Errorf("verification section: %w", err)
			}
			ac.Verification = code
			ac.Language = lang
		case "Scenarios":
			// Intentionally ignored.
		}
	}

	return ac, nil
}

// parseACInputsTable parses the Inputs markdown table body into []ACInput.
// Expected columns: Name | Required | Description
func parseACInputsTable(body string) ([]ACInput, error) {
	var inputs []ACInput
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := splitTableRow(trimmed)
		if len(cells) < 3 {
			continue
		}
		name := cells[0]
		requiredStr := cells[1]
		desc := cells[2]
		// Skip header and separator rows.
		if name == "Name" || strings.HasPrefix(name, "---") || name == "" {
			continue
		}
		inputs = append(inputs, ACInput{
			Name:        name,
			Required:    strings.EqualFold(requiredStr, "yes"),
			Description: desc,
		})
	}
	return inputs, nil
}
