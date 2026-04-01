package gitstore

// Features implemented: state-store/backends/git

import (
	"github.com/synchestra-io/specscore/pkg/task"
)

// parseTaskFile delegates to specscore's task.ParseTaskFile.
func parseTaskFile(data []byte) (task.TaskFileData, error) {
	return task.ParseTaskFile(data)
}

// renderTaskFile delegates to specscore's task.RenderTaskFile.
func renderTaskFile(d task.TaskFileData) []byte {
	return task.RenderTaskFile(d)
}
