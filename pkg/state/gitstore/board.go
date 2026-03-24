package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// boardData holds the parsed contents of a tasks/README.md board.
type boardData struct {
	Rows []boardRow
}

// boardRow is a single parsed row from the board table.
type boardRow struct {
	Slug      string           // extracted from the link text, e.g. "setup-db"
	Status    state.TaskStatus // e.g. TaskStatusCompleted
	DependsOn []string         // parsed from comma-separated list, empty slice if "—"
	Branch    string           // e.g. "agent/run-1", empty if "—"
	Agent     string           // e.g. "Opus 4", empty if "—"
	Requester string           // e.g. "@alex", empty if "—"
	Time      string           // raw time string, "—" if empty
}

// statusEmojis maps each task status to its emoji prefix.
var statusEmojis = map[state.TaskStatus]string{
	state.TaskStatusPlanning:   "📋",
	state.TaskStatusQueued:     "⏳",
	state.TaskStatusClaimed:    "🔒",
	state.TaskStatusInProgress: "🔵",
	state.TaskStatusCompleted:  "✅",
	state.TaskStatusFailed:     "❌",
	state.TaskStatusBlocked:    "🟡",
	state.TaskStatusAborted:    "⛔",
}

// statusEmoji returns the emoji prefix for a task status.
func statusEmoji(s state.TaskStatus) string {
	if e, ok := statusEmojis[s]; ok {
		return e
	}
	return "❓"
}

// parseStatusCell parses a status cell like "✅ `completed`" into TaskStatus.
func parseStatusCell(cell string) (state.TaskStatus, error) {
	cell = strings.TrimSpace(cell)
	// Extract the status name from between backticks.
	start := strings.IndexByte(cell, '`')
	if start < 0 {
		return "", fmt.Errorf("invalid status cell: %q", cell)
	}
	end := strings.IndexByte(cell[start+1:], '`')
	if end < 0 {
		return "", fmt.Errorf("invalid status cell: %q", cell)
	}
	name := cell[start+1 : start+1+end]
	status := state.TaskStatus(name)
	if _, ok := statusEmojis[status]; !ok {
		return "", fmt.Errorf("unknown status: %q", name)
	}
	return status, nil
}

// parseBoard parses a board markdown file into structured data.
func parseBoard(data []byte) (boardData, error) {
	lines := strings.Split(string(data), "\n")
	var bd boardData

	// Find the separator line (|---|...).
	sepIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "---") {
			sepIdx = i
			break
		}
	}
	if sepIdx < 0 {
		return bd, fmt.Errorf("no table separator found")
	}

	// Parse data rows after the separator.
	for _, line := range lines[sepIdx+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		row, err := parseBoardRow(trimmed)
		if err != nil {
			return bd, err
		}
		bd.Rows = append(bd.Rows, row)
	}
	return bd, nil
}

// parseBoardRow parses a single |-delimited table row.
func parseBoardRow(line string) (boardRow, error) {
	// Split by | and trim; leading/trailing empty cells from outer pipes.
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	// Remove leading and trailing empty strings from outer pipes.
	if len(cells) > 0 && cells[0] == "" {
		cells = cells[1:]
	}
	if len(cells) > 0 && cells[len(cells)-1] == "" {
		cells = cells[:len(cells)-1]
	}
	if len(cells) != 7 {
		return boardRow{}, fmt.Errorf("expected 7 columns, got %d", len(cells))
	}

	slug := extractSlug(cells[0])

	status, err := parseStatusCell(cells[1])
	if err != nil {
		return boardRow{}, err
	}

	return boardRow{
		Slug:      slug,
		Status:    status,
		DependsOn: parseDeps(cells[2]),
		Branch:    parseDash(cells[3]),
		Agent:     parseDash(cells[4]),
		Requester: parseDash(cells[5]),
		Time:      parseDashKeep(cells[6]),
	}, nil
}

// extractSlug extracts the link text from "[slug](slug/)".
func extractSlug(cell string) string {
	cell = strings.TrimSpace(cell)
	if start := strings.IndexByte(cell, '['); start >= 0 {
		if end := strings.IndexByte(cell[start:], ']'); end > 0 {
			return cell[start+1 : start+end]
		}
	}
	return cell
}

// parseDeps parses a comma-separated dependency list, returning nil for "—".
func parseDeps(cell string) []string {
	cell = strings.TrimSpace(cell)
	if cell == "—" || cell == "" {
		return nil
	}
	parts := strings.Split(cell, ",")
	deps := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			deps = append(deps, p)
		}
	}
	return deps
}

// parseDash returns empty string for "—", otherwise the trimmed cell.
func parseDash(cell string) string {
	cell = strings.TrimSpace(cell)
	// Strip surrounding backticks if present (e.g. `agent/run-1`).
	if len(cell) >= 2 && cell[0] == '`' && cell[len(cell)-1] == '`' {
		cell = cell[1 : len(cell)-1]
	}
	if cell == "—" {
		return ""
	}
	return cell
}

// parseDashKeep returns "—" as empty, otherwise keeps raw value.
func parseDashKeep(cell string) string {
	cell = strings.TrimSpace(cell)
	if cell == "—" {
		return ""
	}
	return cell
}

// renderBoard renders a boardData to markdown bytes.
func renderBoard(bd boardData) []byte {
	var buf bytes.Buffer

	_, _ = buf.WriteString("# Tasks\n\n")
	_, _ = buf.WriteString("| Task | Status | Depends on | Branch | Agent | Requester | Time |\n")
	_, _ = buf.WriteString("|---|---|---|---|---|---|---|\n")

	for _, r := range bd.Rows {
		task := fmt.Sprintf("[%s](%s/)", r.Slug, r.Slug)
		status := fmt.Sprintf("%s `%s`", statusEmoji(r.Status), string(r.Status))
		deps := renderDash(strings.Join(r.DependsOn, ", "))
		branch := renderDashBacktick(r.Branch)
		agent := renderDash(r.Agent)
		requester := renderDash(r.Requester)
		tm := renderDash(r.Time)

		_, _ = fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s | %s | %s |\n",
			task, status, deps, branch, agent, requester, tm)
	}

	return buf.Bytes()
}

// renderDash returns "—" for empty strings.
func renderDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// renderDashBacktick wraps non-empty values in backticks, empty becomes "—".
func renderDashBacktick(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + s + "`"
}
