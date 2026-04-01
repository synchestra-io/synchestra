package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"github.com/synchestra-io/specscore/pkg/task"
)

// boardData holds the parsed contents of a tasks/README.md board.
// This is a thin wrapper around task.BoardView with a Slug-oriented row type
// used internally by the gitstore layer.
type boardData struct {
	Rows []boardRow
}

// boardRow is a single parsed row from the board table.
// It wraps task.BoardRow fields plus the slug extracted from the Task link.
type boardRow struct {
	Slug      string
	Status    task.TaskStatus
	DependsOn []string
	Branch    string
	Agent     string
	Requester string
	Time      string
}

// parseStatusCell parses a status cell like "emoji `status`" into TaskStatus.
func parseStatusCell(cell string) (task.TaskStatus, error) {
	return task.ParseStatusCell(cell)
}

// parseBoard parses a board markdown file into structured data.
func parseBoard(data []byte) (boardData, error) {
	bv, err := task.ParseBoard(data)
	if err != nil {
		return boardData{}, err
	}
	var bd boardData
	for _, r := range bv.Rows {
		bd.Rows = append(bd.Rows, boardRow{
			Slug:      r.Task,
			Status:    r.Status,
			DependsOn: r.DependsOn,
			Branch:    r.Branch,
			Agent:     r.Agent,
			Requester: r.Requester,
			Time:      r.Time,
		})
	}
	return bd, nil
}

// renderBoard renders a boardData to markdown bytes.
func renderBoard(bd boardData) []byte {
	bv := &task.BoardView{}
	for _, r := range bd.Rows {
		bv.Rows = append(bv.Rows, task.BoardRow{
			Task:      r.Slug,
			Status:    r.Status,
			DependsOn: r.DependsOn,
			Branch:    r.Branch,
			Agent:     r.Agent,
			Requester: r.Requester,
			Time:      r.Time,
		})
	}
	return task.RenderBoard(bv)
}

// extractSlug extracts the link text from "[slug](slug/)".
func extractSlug(cell string) string {
	return task.ExtractSlug(cell)
}

// parseDeps parses a comma-separated dependency list, returning nil for em-dash.
func parseDeps(cell string) []string {
	return task.ParseDeps(cell)
}
