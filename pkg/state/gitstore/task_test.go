package gitstore_test

// Features depended on: state-store, state-store/backends/git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

// setupTestRepo creates a bare repo and a clone with an initial commit
// containing a tasks/ directory and an empty board.
func setupTestRepo(t *testing.T) (bareDir, cloneDir string) {
	t.Helper()
	bareDir = t.TempDir()
	cloneDir = t.TempDir()

	cmds := [][]string{
		{"git", "init", "--bare", bareDir},
		{"git", "clone", bareDir, cloneDir},
		{"git", "-C", cloneDir, "config", "user.email", "test@test.com"},
		{"git", "-C", cloneDir, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	// Create tasks/ dir with an initial empty board.
	tasksDir := filepath.Join(cloneDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	board := "# Tasks\n\n| Task | Status | Depends on | Branch | Agent | Requester | Time |\n|---|---|---|---|---|---|---|\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "README.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}

	cmds = [][]string{
		{"git", "-C", cloneDir, "add", "."},
		{"git", "-C", cloneDir, "commit", "-m", "init"},
		{"git", "-C", cloneDir, "push", "origin", "main"},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			// Try master if main fails on push.
			if args[len(args)-1] == "main" {
				args[len(args)-1] = "master"
				out2, err2 := exec.Command(args[0], args[1:]...).CombinedOutput()
				if err2 != nil {
					t.Fatalf("%v failed: %v\n%s\nalso tried master: %v\n%s", args, err, out, err2, out2)
				}
				continue
			}
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}
	return bareDir, cloneDir
}

// newTestStore creates a GitStateStore pointing at cloneDir with on_commit sync.
func newTestStore(t *testing.T, cloneDir string) state.Store {
	t.Helper()
	s, err := gitstore.New(context.Background(), gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{
			StateRepoPath: cloneDir,
			SpecRepoPaths: []string{t.TempDir()},
			Sync: state.SyncConfig{
				Pull: state.SyncOnCommit,
				Push: state.SyncOnCommit,
			},
		},
		RunID: "test-run-1",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

// newLocalStore creates a store with manual sync (no push/pull).
func newLocalStore(t *testing.T) (state.Store, string) {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	board := "# Tasks\n\n| Task | Status | Depends on | Branch | Agent | Requester | Time |\n|---|---|---|---|---|---|---|\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "README.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}

	cmds = [][]string{
		{"git", "-C", dir, "add", "."},
		{"git", "-C", dir, "commit", "-m", "init"},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	s, err := gitstore.New(context.Background(), gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{
			StateRepoPath: dir,
			Sync: state.SyncConfig{
				Pull: state.SyncManual,
				Push: state.SyncManual,
			},
		},
		RunID: "test-run-1",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s, dir
}

func createTask(t *testing.T, store state.Store, slug, title string) state.CoordinatedTask {
	t.Helper()
	task, err := store.Task().Create(context.Background(), state.TaskCreateParams{
		Slug:      slug,
		Title:     title,
		Requester: "@test",
	})
	if err != nil {
		t.Fatalf("Create(%s) error: %v", slug, err)
	}
	return task
}

func TestCreate(t *testing.T) {
	store, dir := newLocalStore(t)

	task := createTask(t, store, "setup-db", "Setup Database")

	if task.Slug != "setup-db" {
		t.Errorf("slug = %q, want %q", task.Slug, "setup-db")
	}
	if task.Title != "Setup Database" {
		t.Errorf("title = %q, want %q", task.Title, "Setup Database")
	}
	if task.Status != state.TaskStatusPlanning {
		t.Errorf("status = %q, want %q", task.Status, state.TaskStatusPlanning)
	}

	// Verify file on disk.
	readme := filepath.Join(dir, "tasks", "setup-db", "README.md")
	if _, err := os.Stat(readme); err != nil {
		t.Errorf("task README not found: %v", err)
	}

	// Verify board updated.
	boardData, err := os.ReadFile(filepath.Join(dir, "tasks", "README.md"))
	if err != nil {
		t.Fatalf("read board: %v", err)
	}
	if got := string(boardData); !contains(got, "setup-db") {
		t.Errorf("board does not contain task slug")
	}
}

func TestGet(t *testing.T) {
	store, _ := newLocalStore(t)

	createTask(t, store, "my-task", "My Task")

	got, err := store.Task().Get(context.Background(), "my-task")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Slug != "my-task" {
		t.Errorf("slug = %q, want %q", got.Slug, "my-task")
	}
	if got.Title != "My Task" {
		t.Errorf("title = %q, want %q", got.Title, "My Task")
	}
	if got.Status != state.TaskStatusPlanning {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusPlanning)
	}
	if got.Requester != "@test" {
		t.Errorf("requester = %q, want %q", got.Requester, "@test")
	}
}

func TestGet_NotFound(t *testing.T) {
	store, _ := newLocalStore(t)

	_, err := store.Task().Get(context.Background(), "nonexistent")
	if !errors.Is(err, state.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	store, _ := newLocalStore(t)

	createTask(t, store, "task-a", "Task A")
	createTask(t, store, "task-b", "Task B")
	createTask(t, store, "task-c", "Task C")

	tasks, err := store.Task().List(context.Background(), state.TaskFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("len = %d, want 3", len(tasks))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "task-a", "Task A")
	createTask(t, store, "task-b", "Task B")

	if err := store.Task().Enqueue(ctx, "task-b"); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	queued := state.TaskStatusQueued
	tasks, err := store.Task().List(ctx, state.TaskFilter{Status: &queued})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len = %d, want 1", len(tasks))
	}
	if tasks[0].Slug != "task-b" {
		t.Errorf("slug = %q, want %q", tasks[0].Slug, "task-b")
	}
}

func TestEnqueue(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")

	if err := store.Task().Enqueue(ctx, "my-task"); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusQueued {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusQueued)
	}
}

func TestEnqueue_InvalidTransition(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})
	_ = store.Task().Start(ctx, "my-task")
	_ = store.Task().Complete(ctx, "my-task", "done")

	err := store.Task().Enqueue(ctx, "my-task")
	if !errors.Is(err, state.ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestClaim(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")

	err := store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "opus-4"})
	if err != nil {
		t.Fatalf("Claim error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusClaimed {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusClaimed)
	}
	if got.Run != "agent/test-run-1" {
		t.Errorf("run = %q, want %q", got.Run, "agent/test-run-1")
	}
	if got.Model != "opus-4" {
		t.Errorf("model = %q, want %q", got.Model, "opus-4")
	}
}

func TestStart(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})

	if err := store.Task().Start(ctx, "my-task"); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusInProgress {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusInProgress)
	}
}

func TestComplete(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})
	_ = store.Task().Start(ctx, "my-task")

	if err := store.Task().Complete(ctx, "my-task", "All done successfully"); err != nil {
		t.Fatalf("Complete error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusCompleted {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusCompleted)
	}
	if got.Summary != "All done successfully" {
		t.Errorf("summary = %q, want %q", got.Summary, "All done successfully")
	}
}

func TestFail(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})
	_ = store.Task().Start(ctx, "my-task")

	if err := store.Task().Fail(ctx, "my-task", "dependency missing"); err != nil {
		t.Fatalf("Fail error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusFailed {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusFailed)
	}
	if got.Summary != "dependency missing" {
		t.Errorf("summary = %q, want %q", got.Summary, "dependency missing")
	}
}

func TestBlock_Unblock(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})
	_ = store.Task().Start(ctx, "my-task")

	if err := store.Task().Block(ctx, "my-task", "waiting on API key"); err != nil {
		t.Fatalf("Block error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusBlocked {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusBlocked)
	}

	if err := store.Task().Unblock(ctx, "my-task"); err != nil {
		t.Fatalf("Unblock error: %v", err)
	}

	got, err = store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusInProgress {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusInProgress)
	}
}

func TestRelease(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})

	if err := store.Task().Release(ctx, "my-task"); err != nil {
		t.Fatalf("Release error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusQueued {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusQueued)
	}
	if got.Run != "" {
		t.Errorf("run = %q, want empty", got.Run)
	}
}

func TestConfirmAbort(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "my-task", "My Task")
	_ = store.Task().Enqueue(ctx, "my-task")
	_ = store.Task().Claim(ctx, "my-task", state.ClaimParams{Model: "test"})

	if err := store.Task().ConfirmAbort(ctx, "my-task"); err != nil {
		t.Fatalf("ConfirmAbort error: %v", err)
	}

	got, err := store.Task().Get(ctx, "my-task")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != state.TaskStatusAborted {
		t.Errorf("status = %q, want %q", got.Status, state.TaskStatusAborted)
	}
}

func TestBoardGet(t *testing.T) {
	store, _ := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "task-a", "Task A")
	createTask(t, store, "task-b", "Task B")

	board, err := store.Task().Board().Get(ctx)
	if err != nil {
		t.Fatalf("Board.Get error: %v", err)
	}
	if len(board.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(board.Rows))
	}
	if board.Rows[0].Task != "task-a" {
		t.Errorf("row[0].Task = %q, want %q", board.Rows[0].Task, "task-a")
	}
}

func TestBoardRebuild(t *testing.T) {
	store, dir := newLocalStore(t)
	ctx := context.Background()

	createTask(t, store, "task-a", "Task A")
	createTask(t, store, "task-b", "Task B")

	// Corrupt the board by overwriting it.
	boardPath := filepath.Join(dir, "tasks", "README.md")
	corrupt := "# Tasks\n\n| Task | Status | Depends on | Branch | Agent | Requester | Time |\n|---|---|---|---|---|---|---|\n"
	if err := os.WriteFile(boardPath, []byte(corrupt), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := store.Task().Board().Rebuild(ctx); err != nil {
		t.Fatalf("Rebuild error: %v", err)
	}

	board, err := store.Task().Board().Get(ctx)
	if err != nil {
		t.Fatalf("Board.Get error: %v", err)
	}
	if len(board.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(board.Rows))
	}
}

func TestCreateWithRemote(t *testing.T) {
	_, cloneDir := setupTestRepo(t)
	store := newTestStore(t, cloneDir)

	task := createTask(t, store, "remote-task", "Remote Task")
	if task.Slug != "remote-task" {
		t.Errorf("slug = %q, want %q", task.Slug, "remote-task")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
