package gitstore

// Features implemented: state-store/backends/git, cli/task/claim, cli/task/update
// Features depended on:  state-store, state-store/task-store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/synchestra-io/specscore/pkg/task"
	"github.com/synchestra-io/synchestra/pkg/cli/gitops"
	"github.com/synchestra-io/synchestra/pkg/state"
)

// gitTaskStore implements state.TaskStore backed by git-managed markdown files.
type gitTaskStore struct{ store *GitStateStore }

// Compile-time interface checks.
var (
	_ state.TaskStore = (*gitTaskStore)(nil)
	_ state.Board     = (*gitBoard)(nil)
)

// gitBoard implements state.Board backed by tasks/README.md.
type gitBoard struct{ store *GitStateStore }

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

func (t *gitTaskStore) tasksDir() string           { return filepath.Join(t.store.stateRepoPath, "tasks") }
func (t *gitTaskStore) taskDir(slug string) string { return filepath.Join(t.tasksDir(), slug) }
func (t *gitTaskStore) boardPath() string          { return filepath.Join(t.tasksDir(), "README.md") }
func (t *gitTaskStore) shouldPull() bool           { return t.store.sync.Pull == state.SyncOnCommit }
func (t *gitTaskStore) shouldPush() bool           { return t.store.sync.Push == state.SyncOnCommit }

func (t *gitTaskStore) maybePull() error {
	if t.shouldPull() {
		return gitops.Pull(t.store.stateRepoPath)
	}
	return nil
}

func (t *gitTaskStore) maybePush() error {
	if t.shouldPush() {
		return gitops.Push(t.store.stateRepoPath)
	}
	return nil
}

// ---------------------------------------------------------------------------
// File I/O helpers
// ---------------------------------------------------------------------------

func (t *gitTaskStore) readBoard() (boardData, error) {
	data, err := os.ReadFile(t.boardPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return boardData{}, nil
		}
		return boardData{}, err
	}
	return parseBoard(data)
}

func (t *gitTaskStore) writeBoard(bd boardData) error {
	return os.WriteFile(t.boardPath(), renderBoard(bd), 0o644)
}

func (t *gitTaskStore) readTaskFile(slug string) (task.TaskFileData, error) {
	p := filepath.Join(t.taskDir(slug), "README.md")
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.TaskFileData{}, state.ErrNotFound
		}
		return task.TaskFileData{}, err
	}
	return parseTaskFile(data)
}

func (t *gitTaskStore) writeTaskFile(slug string, d task.TaskFileData) error {
	dir := t.taskDir(slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "README.md"), renderTaskFile(d), 0o644)
}

// findRow locates a board row by slug.  Returns -1, nil when not found.
func (t *gitTaskStore) findRow(bd *boardData, slug string) (int, *boardRow) {
	for i := range bd.Rows {
		if bd.Rows[i].Slug == slug {
			return i, &bd.Rows[i]
		}
	}
	return -1, nil
}

// assembleTask builds a state.CoordinatedTask from a boardRow and task.TaskFileData.
func assembleTask(slug string, row boardRow, tf task.TaskFileData) state.CoordinatedTask {
	return state.CoordinatedTask{
		Task: task.Task{
			Slug:      slug,
			Title:     tf.Title,
			Status:    row.Status,
			DependsOn: tf.DependsOn,
			Requester: row.Requester,
			Summary:   tf.Summary,
		},
		Run:   row.Branch,
		Model: row.Agent,
	}
}

// commitFiles commits the listed repo-relative paths.
func (t *gitTaskStore) commitFiles(files []string, message string) error {
	return gitops.Commit(t.store.stateRepoPath, files, message)
}

// ---------------------------------------------------------------------------
// TaskStore methods
// ---------------------------------------------------------------------------

func (t *gitTaskStore) Create(_ context.Context, params state.TaskCreateParams) (state.CoordinatedTask, error) {
	// Write task file.
	tf := task.TaskFileData{
		Title:     params.Title,
		DependsOn: params.DependsOn,
	}
	if err := t.writeTaskFile(params.Slug, tf); err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("write task file: %w", err)
	}

	// Update board.
	bd, err := t.readBoard()
	if err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("read board: %w", err)
	}
	row := boardRow{
		Slug:      params.Slug,
		Status:    state.TaskStatusPlanning,
		DependsOn: params.DependsOn,
		Requester: params.Requester,
	}
	bd.Rows = append(bd.Rows, row)
	if err := t.writeBoard(bd); err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("write board: %w", err)
	}

	// Commit.
	files := []string{
		filepath.Join("tasks", params.Slug, "README.md"),
		filepath.Join("tasks", "README.md"),
	}
	if err := t.commitFiles(files, fmt.Sprintf("task: create %s", params.Slug)); err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("commit: %w", err)
	}
	if err := t.maybePush(); err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("push: %w", err)
	}

	return assembleTask(params.Slug, row, tf), nil
}

func (t *gitTaskStore) Get(_ context.Context, slug string) (state.CoordinatedTask, error) {
	if err := t.maybePull(); err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("pull: %w", err)
	}

	tf, err := t.readTaskFile(slug)
	if err != nil {
		return state.CoordinatedTask{}, err
	}

	bd, err := t.readBoard()
	if err != nil {
		return state.CoordinatedTask{}, fmt.Errorf("read board: %w", err)
	}
	_, row := t.findRow(&bd, slug)
	if row == nil {
		return state.CoordinatedTask{}, state.ErrNotFound
	}

	return assembleTask(slug, *row, tf), nil
}

func (t *gitTaskStore) List(_ context.Context, filter state.TaskFilter) ([]state.CoordinatedTask, error) {
	if err := t.maybePull(); err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}

	bd, err := t.readBoard()
	if err != nil {
		return nil, fmt.Errorf("read board: %w", err)
	}

	var tasks []state.CoordinatedTask
	for _, row := range bd.Rows {
		if filter.Status != nil && row.Status != *filter.Status {
			continue
		}
		tf, err := t.readTaskFile(row.Slug)
		if err != nil {
			return nil, fmt.Errorf("read task %s: %w", row.Slug, err)
		}
		tasks = append(tasks, assembleTask(row.Slug, row, tf))
	}
	return tasks, nil
}

// transitionTask is a generic helper for simple status transitions.
// It validates the current status, applies the new status, optionally
// updates the task file (e.g. for summary/reason), commits and pushes.
func (t *gitTaskStore) transitionTask(
	slug string,
	allowed []state.TaskStatus,
	newStatus state.TaskStatus,
	updateTaskFile func(row *boardRow, tf *task.TaskFileData),
	commitMsg string,
) error {
	if err := t.maybePull(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	bd, err := t.readBoard()
	if err != nil {
		return fmt.Errorf("read board: %w", err)
	}
	_, row := t.findRow(&bd, slug)
	if row == nil {
		return state.ErrNotFound
	}

	if !statusIn(row.Status, allowed) {
		return state.ErrInvalidTransition
	}

	row.Status = newStatus

	var files []string
	files = append(files, filepath.Join("tasks", "README.md"))

	if updateTaskFile != nil {
		tf, err := t.readTaskFile(slug)
		if err != nil {
			return fmt.Errorf("read task file: %w", err)
		}
		updateTaskFile(row, &tf)
		if err := t.writeTaskFile(slug, tf); err != nil {
			return fmt.Errorf("write task file: %w", err)
		}
		files = append(files, filepath.Join("tasks", slug, "README.md"))
	}

	if err := t.writeBoard(bd); err != nil {
		return fmt.Errorf("write board: %w", err)
	}
	if err := t.commitFiles(files, commitMsg); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	if err := t.maybePush(); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

func statusIn(s state.TaskStatus, allowed []state.TaskStatus) bool {
	for _, a := range allowed {
		if s == a {
			return true
		}
	}
	return false
}

func (t *gitTaskStore) Enqueue(_ context.Context, slug string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusPlanning},
		state.TaskStatusQueued,
		nil,
		fmt.Sprintf("task: enqueue %s", slug),
	)
}

func (t *gitTaskStore) Claim(_ context.Context, slug string, claim state.ClaimParams) error {
	if err := t.maybePull(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	bd, err := t.readBoard()
	if err != nil {
		return fmt.Errorf("read board: %w", err)
	}
	_, row := t.findRow(&bd, slug)
	if row == nil {
		return state.ErrNotFound
	}
	if row.Status != state.TaskStatusQueued {
		return state.ErrInvalidTransition
	}

	row.Status = state.TaskStatusClaimed
	row.Branch = fmt.Sprintf("agent/%s", t.store.runID)
	row.Agent = claim.Model
	row.Time = time.Now().UTC().Format("2006-01-02 15:04")

	if err := t.writeBoard(bd); err != nil {
		return fmt.Errorf("write board: %w", err)
	}

	files := []string{filepath.Join("tasks", "README.md")}
	if err := t.commitFiles(files, fmt.Sprintf("task: claim %s", slug)); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if t.shouldPush() {
		if pushErr := gitops.Push(t.store.stateRepoPath); pushErr != nil {
			// Conflict resolution: pull, check if someone else claimed it.
			if pullErr := gitops.Pull(t.store.stateRepoPath); pullErr != nil {
				return fmt.Errorf("push failed and pull failed: %w", pullErr)
			}
			bd2, err := t.readBoard()
			if err != nil {
				return fmt.Errorf("re-read board after conflict: %w", err)
			}
			_, row2 := t.findRow(&bd2, slug)
			if row2 == nil {
				return state.ErrNotFound
			}
			if row2.Status != state.TaskStatusQueued {
				return state.ErrConflict
			}
			// Re-apply our claim.
			row2.Status = state.TaskStatusClaimed
			row2.Branch = fmt.Sprintf("agent/%s", t.store.runID)
			row2.Agent = claim.Model
			row2.Time = time.Now().UTC().Format("2006-01-02 15:04")
			if err := t.writeBoard(bd2); err != nil {
				return fmt.Errorf("write board retry: %w", err)
			}
			if err := t.commitFiles(files, fmt.Sprintf("task: claim %s (retry)", slug)); err != nil {
				return fmt.Errorf("commit retry: %w", err)
			}
			if err := gitops.Push(t.store.stateRepoPath); err != nil {
				return fmt.Errorf("push retry: %w", err)
			}
		}
	}
	return nil
}

func (t *gitTaskStore) Start(_ context.Context, slug string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusClaimed},
		state.TaskStatusInProgress,
		nil,
		fmt.Sprintf("task: start %s", slug),
	)
}

func (t *gitTaskStore) Complete(_ context.Context, slug string, summary string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusInProgress},
		state.TaskStatusCompleted,
		func(_ *boardRow, tf *task.TaskFileData) { tf.Summary = summary },
		fmt.Sprintf("task: complete %s", slug),
	)
}

func (t *gitTaskStore) Fail(_ context.Context, slug string, reason string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusInProgress},
		state.TaskStatusFailed,
		func(_ *boardRow, tf *task.TaskFileData) { tf.Summary = reason },
		fmt.Sprintf("task: fail %s", slug),
	)
}

func (t *gitTaskStore) Block(_ context.Context, slug string, reason string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusInProgress},
		state.TaskStatusBlocked,
		func(_ *boardRow, tf *task.TaskFileData) { tf.Summary = reason },
		fmt.Sprintf("task: block %s", slug),
	)
}

func (t *gitTaskStore) Unblock(_ context.Context, slug string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusBlocked},
		state.TaskStatusInProgress,
		func(_ *boardRow, tf *task.TaskFileData) { tf.Summary = "" },
		fmt.Sprintf("task: unblock %s", slug),
	)
}

func (t *gitTaskStore) Release(_ context.Context, slug string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusClaimed},
		state.TaskStatusQueued,
		func(row *boardRow, _ *task.TaskFileData) {
			row.Branch = ""
			row.Agent = ""
			row.Time = ""
		},
		fmt.Sprintf("task: release %s", slug),
	)
}

func (t *gitTaskStore) RequestAbort(_ context.Context, slug string) error {
	if err := t.maybePull(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	bd, err := t.readBoard()
	if err != nil {
		return fmt.Errorf("read board: %w", err)
	}
	_, row := t.findRow(&bd, slug)
	if row == nil {
		return state.ErrNotFound
	}
	if !statusIn(row.Status, []state.TaskStatus{state.TaskStatusClaimed, state.TaskStatusInProgress}) {
		return state.ErrInvalidTransition
	}
	// v1 no-op: abort_requested flag is not stored on disk yet.
	return nil
}

func (t *gitTaskStore) ConfirmAbort(_ context.Context, slug string) error {
	return t.transitionTask(slug,
		[]state.TaskStatus{state.TaskStatusClaimed, state.TaskStatusInProgress},
		state.TaskStatusAborted,
		nil,
		fmt.Sprintf("task: abort %s", slug),
	)
}

func (t *gitTaskStore) Board() state.Board {
	return &gitBoard{store: t.store}
}

func (t *gitTaskStore) Artifact(_ context.Context, _ string) state.ArtifactStore {
	return &gitArtifactStore{store: t.store}
}

// ---------------------------------------------------------------------------
// Board methods
// ---------------------------------------------------------------------------

func (b *gitBoard) Rebuild(_ context.Context) error {
	ts := &gitTaskStore{store: b.store}
	tasksRoot := ts.tasksDir()

	entries, err := os.ReadDir(tasksRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read tasks dir: %w", err)
	}

	var rows []boardRow
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		readme := filepath.Join(tasksRoot, slug, "README.md")
		if _, err := os.Stat(readme); err != nil {
			continue
		}
		tf, err := ts.readTaskFile(slug)
		if err != nil {
			return fmt.Errorf("parse task %s: %w", slug, err)
		}
		rows = append(rows, boardRow{
			Slug:      slug,
			Status:    state.TaskStatusPlanning,
			DependsOn: tf.DependsOn,
		})
	}

	// Try to preserve statuses from the existing board if available.
	existingBd, _ := ts.readBoard()
	existing := make(map[string]boardRow, len(existingBd.Rows))
	for _, r := range existingBd.Rows {
		existing[r.Slug] = r
	}
	for i, r := range rows {
		if old, ok := existing[r.Slug]; ok {
			rows[i] = old
		}
	}

	bd := boardData{Rows: rows}
	if err := ts.writeBoard(bd); err != nil {
		return fmt.Errorf("write board: %w", err)
	}

	files := []string{filepath.Join("tasks", "README.md")}
	if err := ts.commitFiles(files, "board: rebuild"); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (b *gitBoard) Get(_ context.Context) (state.BoardView, error) {
	ts := &gitTaskStore{store: b.store}
	if err := ts.maybePull(); err != nil {
		return state.BoardView{}, fmt.Errorf("pull: %w", err)
	}

	bd, err := ts.readBoard()
	if err != nil {
		return state.BoardView{}, fmt.Errorf("read board: %w", err)
	}

	var view state.BoardView
	for _, r := range bd.Rows {
		view.Rows = append(view.Rows, state.BoardRow{
			Task:      r.Slug,
			Status:    r.Status,
			DependsOn: r.DependsOn,
			Branch:    r.Branch,
			Agent:     r.Agent,
			Requester: r.Requester,
			Time:      r.Time,
		})
	}
	return view, nil
}
