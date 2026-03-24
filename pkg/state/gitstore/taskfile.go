package gitstore

// Features implemented: state-store/backends/git

import (
	"bytes"
	"fmt"
	"strings"
)

// taskFileData holds the parsed contents of a task README.md.
type taskFileData struct {
	Title       string
	Description string
	DependsOn   []string
	Summary     string
}

// parseTaskFile parses a task README.md into its structured fields.
func parseTaskFile(data []byte) (taskFileData, error) {
	content := string(data)

	// Title must be the first line and start with "# ".
	if !strings.HasPrefix(content, "# ") {
		return taskFileData{}, fmt.Errorf("missing task title: expected line starting with '# '")
	}

	firstNL := strings.Index(content, "\n")
	if firstNL == -1 {
		return taskFileData{}, fmt.Errorf("missing Dependencies section")
	}
	title := strings.TrimSpace(content[2:firstNL])
	rest := content[firstNL+1:]

	// Split the remainder on "\n## " to locate H2 sections.
	parts := strings.Split(rest, "\n## ")

	// parts[0] is everything before the first H2 — the description.
	description := strings.TrimSpace(parts[0])

	sections := make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		idx := strings.Index(p, "\n")
		if idx == -1 {
			sections[strings.TrimSpace(p)] = ""
			continue
		}
		heading := strings.TrimSpace(p[:idx])
		body := strings.TrimSpace(p[idx+1:])
		sections[heading] = body
	}

	// Dependencies section is required.
	depBody, ok := sections["Dependencies"]
	if !ok {
		return taskFileData{}, fmt.Errorf("missing Dependencies section")
	}

	var deps []string
	if depBody != "None" {
		for _, line := range strings.Split(depBody, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				deps = append(deps, strings.TrimSpace(line[2:]))
			}
		}
	}

	// Summary section is required.
	summaryBody, ok := sections["Summary"]
	if !ok {
		return taskFileData{}, fmt.Errorf("missing Summary section")
	}
	summary := summaryBody
	if summary == "None" {
		summary = ""
	}

	return taskFileData{
		Title:       title,
		Description: description,
		DependsOn:   deps,
		Summary:     summary,
	}, nil
}

// renderTaskFile renders a taskFileData to markdown bytes.
func renderTaskFile(d taskFileData) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# %s\n", d.Title)

	if d.Description != "" {
		fmt.Fprintf(&buf, "\n%s\n", d.Description)
	}

	buf.WriteString("\n## Dependencies\n\n")
	if len(d.DependsOn) == 0 {
		buf.WriteString("None\n")
	} else {
		for _, dep := range d.DependsOn {
			fmt.Fprintf(&buf, "- %s\n", dep)
		}
	}

	buf.WriteString("\n## Summary\n\n")
	if d.Summary == "" {
		buf.WriteString("None\n")
	} else {
		fmt.Fprintf(&buf, "%s\n", d.Summary)
	}

	return buf.Bytes()
}
