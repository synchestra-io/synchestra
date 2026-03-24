package gitstore_test

// Features depended on: state-store/backends/git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/gitstore"
)

func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bare.git")
	cmd := exec.Command("git", "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
}

func cloneRepo(t *testing.T, bare string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "clone")
	cmd := exec.Command("git", "clone", bare, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("config", "pull.rebase", "false")
	return dir
}

// seedBare pushes an initial commit to the bare repo so clones have a tracking branch.
func seedBare(t *testing.T, bare string) {
	t.Helper()
	setup := cloneRepo(t, bare)
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = setup
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(setup, "README.md"), []byte("init\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	run("push", "origin", "HEAD")
}

func newSync(t *testing.T, repoPath string) state.StateSync {
	t.Helper()
	ctx := context.Background()
	store, err := gitstore.New(ctx, gitstore.GitStoreOptions{
		StoreOptions: state.StoreOptions{StateRepoPath: repoPath},
	})
	if err != nil {
		t.Fatalf("gitstore.New: %v", err)
	}
	return store.State()
}

func TestSyncPull(t *testing.T) {
	bare := initBareRepo(t)
	seedBare(t, bare)

	// Clone 1 — this is the store under test.
	clone1 := cloneRepo(t, bare)

	// Clone 2 — push a new commit to bare.
	clone2 := cloneRepo(t, bare)
	run2 := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = clone2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(clone2, "pushed.txt"), []byte("from clone2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run2("add", ".")
	run2("commit", "-m", "add pushed.txt")
	run2("push", "origin", "HEAD")

	// Pull via store and verify the file appears.
	sync := newSync(t, clone1)
	if err := sync.Pull(context.Background()); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone1, "pushed.txt")); err != nil {
		t.Fatalf("pushed.txt not found after Pull: %v", err)
	}
}

func TestSyncPush(t *testing.T) {
	bare := initBareRepo(t)
	seedBare(t, bare)

	clone1 := cloneRepo(t, bare)

	// Write a file and commit in clone1 (without pushing).
	run1 := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = clone1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(clone1, "local.txt"), []byte("local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run1("add", ".")
	run1("commit", "-m", "add local.txt")

	// Push via store.
	sync := newSync(t, clone1)
	if err := sync.Push(context.Background()); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Verify via a fresh clone from bare.
	verify := cloneRepo(t, bare)
	if _, err := os.Stat(filepath.Join(verify, "local.txt")); err != nil {
		t.Fatalf("local.txt not found in bare after Push: %v", err)
	}
}

func TestSyncRoundTrip(t *testing.T) {
	bare := initBareRepo(t)
	seedBare(t, bare)

	clone1 := cloneRepo(t, bare)

	// Push a remote commit via clone2.
	clone2 := cloneRepo(t, bare)
	run2 := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = clone2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(clone2, "remote.txt"), []byte("remote\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run2("add", ".")
	run2("commit", "-m", "add remote.txt")
	run2("push", "origin", "HEAD")

	// Also create a local commit in clone1 to push.
	run1 := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = clone1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(clone1, "local2.txt"), []byte("local2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run1("add", ".")
	run1("commit", "-m", "add local2.txt")

	// Sync (Pull then Push).
	sync := newSync(t, clone1)
	if err := sync.Sync(context.Background()); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// clone1 should have remote.txt (from Pull).
	if _, err := os.Stat(filepath.Join(clone1, "remote.txt")); err != nil {
		t.Fatalf("remote.txt not found in clone1 after Sync: %v", err)
	}

	// bare should have local2.txt (from Push) — verify via fresh clone.
	verify := cloneRepo(t, bare)
	if _, err := os.Stat(filepath.Join(verify, "local2.txt")); err != nil {
		t.Fatalf("local2.txt not found in bare after Sync: %v", err)
	}
}
