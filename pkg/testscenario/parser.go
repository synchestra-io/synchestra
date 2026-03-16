package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"strings"
)

// ParseScenario parses a markdown scenario file into a Scenario struct.
func ParseScenario(data []byte) (*Scenario, error) {
	text := string(data)
	lines := strings.Split(text, "\n")
	s := &Scenario{}
	stepNames := make(map[string]bool)

	// Phase 1: Extract H1 title ("# Scenario: <title>")
	h1LineIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# Scenario:") {
			s.Title = strings.TrimSpace(strings.TrimPrefix(line, "# Scenario:"))
			h1LineIdx = i
			break
		}
	}
	if h1LineIdx < 0 {
		return nil, fmt.Errorf("missing '# Scenario:' heading")
	}

	// Phase 2: Extract header metadata from lines between H1 and first ## heading.
	// We'll work on the full text using section splitting.
	// First, find the portion after the H1 line.
	restAfterH1 := strings.Join(lines[h1LineIdx+1:], "\n")

	// Split into sections by "## " headings.
	// The first element is the header block (description/tags).
	// Subsequent elements are section content with heading as first line.
	parts := splitIntoSections(restAfterH1)

	// Parse header metadata from the first part (before any ## heading).
	headerBlock := parts[0]
	for _, line := range strings.Split(headerBlock, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**Description:**") {
			s.Description = strings.TrimSpace(strings.TrimPrefix(line, "**Description:**"))
		} else if strings.HasPrefix(line, "**Tags:**") {
			tagsStr := strings.TrimSpace(strings.TrimPrefix(line, "**Tags:**"))
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					s.Tags = append(s.Tags, tag)
				}
			}
		}
	}

	// Phase 3 & 4: Process each ## section.
	for _, sectionText := range parts[1:] {
		// First line of section is the heading text (after "## " was stripped by split).
		sectionLines := strings.Split(sectionText, "\n")
		heading := strings.TrimSpace(sectionLines[0])
		body := strings.Join(sectionLines[1:], "\n")

		switch heading {
		case "Setup":
			code, lang, err := extractCodeBlock(body)
			if err != nil {
				return nil, fmt.Errorf("setup section: %w", err)
			}
			s.Setup = code
			s.SetupLanguage = lang
		case "Teardown":
			code, lang, err := extractCodeBlock(body)
			if err != nil {
				return nil, fmt.Errorf("teardown section: %w", err)
			}
			s.Teardown = code
			s.TeardownLanguage = lang
		default:
			step, err := parseStep(heading, body)
			if err != nil {
				return nil, fmt.Errorf("step %q: %w", heading, err)
			}
			if stepNames[step.Name] {
				return nil, fmt.Errorf("duplicate step name: %q", step.Name)
			}
			stepNames[step.Name] = true
			s.Steps = append(s.Steps, step)
		}
	}

	// Phase 5: Validate.
	// Validate each step has exactly one of code or include.
	// Validate depends-on references.
	orderedNames := make([]string, 0, len(s.Steps))
	for i, step := range s.Steps {
		// Validate code vs include.
		hasCode := step.Code != "" || step.Language != ""
		hasInclude := step.Include != ""
		if hasCode && hasInclude {
			return nil, fmt.Errorf("step %q has both code and include", step.Name)
		}
		if !hasCode && !hasInclude {
			return nil, fmt.Errorf("step %q has neither code nor include", step.Name)
		}

		// Validate depends-on references point to earlier steps.
		for _, dep := range step.DependsOn {
			if dep == "(none)" {
				continue
			}
			found := false
			for _, prevName := range orderedNames {
				if prevName == dep {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("step %q depends on %q which does not exist or is not an earlier step", step.Name, dep)
			}
		}

		orderedNames = append(orderedNames, s.Steps[i].Name)
	}

	return s, nil
}

// splitIntoSections splits markdown text into sections by "## " headings.
// The first element is any content before the first ## heading.
// Each subsequent element starts with the heading text (without "## " prefix) on the first line.
func splitIntoSections(text string) []string {
	var result []string
	lines := strings.Split(text, "\n")
	var current []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if inSection || len(current) > 0 {
				result = append(result, strings.Join(current, "\n"))
			}
			// Start new section with heading text as first line.
			headingText := strings.TrimPrefix(line, "## ")
			current = []string{headingText}
			inSection = true
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 || inSection {
		result = append(result, strings.Join(current, "\n"))
	}

	// Ensure at least one element (the header block).
	if len(result) == 0 {
		result = []string{""}
	}

	return result
}

// parseStep parses a step section given its heading and body text.
func parseStep(heading, body string) (Step, error) {
	step := Step{Name: heading}
	bodyLines := strings.Split(body, "\n")

	// We'll parse the body line by line tracking current table context.
	type tableState int
	const (
		tableNone tableState = iota
		tableOutputs
		tableACs
	)

	state := tableNone
	var tableLines []string

	for lineIdx, line := range bodyLines {
		trimmed := strings.TrimSpace(line)

		// Detect inline metadata fields.
		if strings.HasPrefix(trimmed, "**Depends on:**") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "**Depends on:**"))
			if val != "" && val != "(none)" {
				for _, dep := range strings.Split(val, ",") {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						step.DependsOn = append(step.DependsOn, dep)
					}
				}
			}
			state = tableNone
			continue
		}
		if strings.HasPrefix(trimmed, "**Parallel:**") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "**Parallel:**"))
			step.Parallel = strings.EqualFold(val, "true")
			state = tableNone
			continue
		}
		if strings.HasPrefix(trimmed, "**Outputs:**") {
			// Flush any pending table.
			if state == tableACs && len(tableLines) > 0 {
				acs, err := parseACsTable(tableLines)
				if err != nil {
					return step, err
				}
				step.ACs = append(step.ACs, acs...)
			}
			state = tableOutputs
			tableLines = nil
			continue
		}
		if strings.HasPrefix(trimmed, "**ACs:**") {
			// Flush any pending outputs table.
			if state == tableOutputs && len(tableLines) > 0 {
				outputs, err := parseOutputsTable(tableLines)
				if err != nil {
					return step, err
				}
				step.Outputs = append(step.Outputs, outputs...)
			}
			state = tableACs
			tableLines = nil
			continue
		}
		if strings.HasPrefix(trimmed, "**Include:**") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "**Include:**"))
			_, url := parseMarkdownLink(val)
			if url == "" {
				url = val
			}
			step.Include = url
			state = tableNone
			continue
		}

		// Code fence detection.
		if strings.HasPrefix(trimmed, "```") {
			// Flush pending table.
			if state == tableOutputs && len(tableLines) > 0 {
				outputs, err := parseOutputsTable(tableLines)
				if err != nil {
					return step, err
				}
				step.Outputs = append(step.Outputs, outputs...)
				tableLines = nil
			} else if state == tableACs && len(tableLines) > 0 {
				acs, err := parseACsTable(tableLines)
				if err != nil {
					return step, err
				}
				step.ACs = append(step.ACs, acs...)
				tableLines = nil
			}
			state = tableNone

			// Find the rest of the code block in bodyLines.
			code, lang, err := extractCodeBlockFromLines(bodyLines, lineIdx)
			if err != nil {
				return step, err
			}
			step.Code = code
			step.Language = lang
			break
		}

		// Table rows.
		if state == tableOutputs || state == tableACs {
			if strings.HasPrefix(trimmed, "|") {
				tableLines = append(tableLines, trimmed)
			}
			continue
		}

		_ = lineIdx
	}

	// Flush remaining tables.
	if state == tableOutputs && len(tableLines) > 0 {
		outputs, err := parseOutputsTable(tableLines)
		if err != nil {
			return step, err
		}
		step.Outputs = append(step.Outputs, outputs...)
	} else if state == tableACs && len(tableLines) > 0 {
		acs, err := parseACsTable(tableLines)
		if err != nil {
			return step, err
		}
		step.ACs = append(step.ACs, acs...)
	}

	return step, nil
}

// extractCodeBlock extracts the first code block from a text body.
func extractCodeBlock(text string) (code, language string, err error) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			return extractCodeBlockFromLines(lines, i)
		}
	}
	return "", "", fmt.Errorf("no code block found")
}

// extractCodeBlockFromLines extracts a code block starting at the given line index.
func extractCodeBlockFromLines(lines []string, startIdx int) (code, language string, err error) {
	openLine := strings.TrimSpace(lines[startIdx])
	// openLine is "```" or "```bash" etc.
	lang := strings.TrimPrefix(openLine, "```")
	lang = strings.TrimSpace(lang)

	if lang == "" {
		// Find actual line number by counting from the start.
		// We use startIdx+1 as a 1-based approximation.
		return "", "", fmt.Errorf("code block at line %d is missing language annotation", startIdx+1)
	}

	// Validate language.
	switch lang {
	case "bash", "python", "starlark":
		// OK
	default:
		return "", "", fmt.Errorf("unsupported language annotation %q (supported: bash, python, starlark)", lang)
	}

	// Collect lines until closing ```.
	var codeLines []string
	for i := startIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "```" {
			return strings.Join(codeLines, "\n"), lang, nil
		}
		codeLines = append(codeLines, lines[i])
	}
	return "", "", fmt.Errorf("unclosed code block starting at line %d", startIdx+1)
}

// parseOutputsTable parses the Outputs markdown table rows into Output structs.
// Expected columns: Name | Store | Extract
func parseOutputsTable(rows []string) ([]Output, error) {
	var outputs []Output
	for _, row := range rows {
		cells := splitTableRow(row)
		if len(cells) < 3 {
			continue
		}
		// Skip header and separator rows.
		name := cells[0]
		storeStr := cells[1]
		extract := cells[2]
		if name == "Name" || strings.HasPrefix(name, "---") || name == "" {
			continue
		}
		// Strip surrounding backticks from extract.
		extract = strings.TrimPrefix(extract, "`")
		extract = strings.TrimSuffix(extract, "`")
		var store OutputStore
		switch strings.ToLower(storeStr) {
		case "context":
			store = StoreContext
		case "step":
			store = StoreStep
		case "both":
			store = StoreBoth
		default:
			return nil, fmt.Errorf("unknown output store %q", storeStr)
		}
		outputs = append(outputs, Output{
			Name:    name,
			Store:   store,
			Extract: extract,
		})
	}
	return outputs, nil
}

// parseACsTable parses the ACs markdown table rows into ACRef structs.
// Expected columns: Feature | ACs
func parseACsTable(rows []string) ([]ACRef, error) {
	var acs []ACRef
	for _, row := range rows {
		cells := splitTableRow(row)
		if len(cells) < 2 {
			continue
		}
		featureCell := cells[0]
		acsCell := cells[1]
		if featureCell == "Feature" || strings.HasPrefix(featureCell, "---") || featureCell == "" {
			continue
		}
		// Feature cell may be a markdown link: [text](url)
		linkText, linkURL := parseMarkdownLink(featureCell)
		featurePath := featureCell
		featureLink := ""
		if linkText != "" {
			featurePath = linkText
			featureLink = linkURL
		}
		acs = append(acs, ACRef{
			FeaturePath: featurePath,
			FeatureLink: featureLink,
			ACs:         strings.TrimSpace(acsCell),
		})
	}
	return acs, nil
}

// splitTableRow splits a markdown table row by "|" and trims whitespace from each cell.
// Leading and trailing "|" are ignored.
func splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}

// parseMarkdownLink extracts text and URL from a "[text](url)" pattern.
// Returns empty strings if no link is found.
func parseMarkdownLink(s string) (text, url string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return "", ""
	}
	closeBracket := strings.Index(s, "]")
	if closeBracket < 0 {
		return "", ""
	}
	text = s[1:closeBracket]
	rest := s[closeBracket+1:]
	if !strings.HasPrefix(rest, "(") {
		return text, ""
	}
	closeParen := strings.Index(rest, ")")
	if closeParen < 0 {
		return text, ""
	}
	url = rest[1:closeParen]
	return text, url
}
