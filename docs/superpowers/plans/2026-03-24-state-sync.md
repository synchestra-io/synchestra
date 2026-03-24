# State Sync Commands & Sync Policy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `synchestra state pull/push/sync` CLI commands and a configurable sync policy (`SyncConfig`) to the state store interface and git backend.

**Architecture:** Extends the existing `state.Store` interface with a `State() StateSync` sub-interface following the hierarchical composition pattern. Adds `SyncConfig` and `SyncPolicy` types to `pkg/state/types.go`. Updates the git backend to replace `SyncMode` with `SyncConfig` and adds a `RunID` field for agent branching. Creates a new `pkg/cli/state/` package with three cobra subcommands.

**Tech Stack:** Go 1.26, cobra, existing `pkg/cli/gitops/` for git operations, `gopkg.in/yaml.v3` for config parsing.

**Spec:** `spec/features/cli/state/README.md`, `spec/features/state-store/README.md`, `spec/features/state-store/backends/git/README.md`

**Design doc:** `docs/superpowers/specs/2026-03-24-state-sync-commands-design.md`

**AGENTS.md rules to follow:**
- Every `.go` file must have `// Features implemented:` / `// Features depended on:` comments after the `package` declaration
- After any change to `.go` files: `gofmt -w`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`, `go vet ./...`
- Every directory MUST have a `README.md` with an "Outstanding Questions" section

---

## Scope

**In scope:** SyncPolicy/SyncConfig types, StateSync interface, git backend update (replace SyncMode), CLI `state pull/push/sync` commands (stubs), SyncConfig YAML parsing.

**Deferred to subsequent plans:**
- Environment-level sync policy overrides (`~/.synchestra.yaml`, `synchestra-server.yaml`) and strictness comparison logic (`IsStricter()`)
- Contended operations override wiring (`task claim` forcing immediate round-trip)
- Full `Pull`/`Push`/`Sync` git operation implementations (plan delivers stubs returning `errNotImplemented`)
- Spec file updates already completed in commit `c81e422` (CLI README, command-environments, task claim, state-store, git backend, state-repo config)

---

## File Structure

```
pkg/
  state/
    store.go          — Modify: add State() StateSync accessor to Store interface
    types.go          — Modify: add SyncPolicy, SyncConfig, StoreOptions.Sync
    types_test.go     — Create: tests for SyncPolicy and SyncConfig types
    sync.go           — Create: StateSync interface definition
    store_test.go     — Create: compile-time interface checks for Store and StateSync
    syncconfig.go     — Create: ParseSyncPolicy function
    syncconfig_test.go — Create: tests for ParseSyncPolicy
    gitstore/
      gitstore.go     — Modify: replace SyncMode/Options with SyncConfig/GitStoreOptions, add State() accessor
      sync.go         — Create: gitStateSync struct implementing state.StateSync
  cli/
    main.go           — Modify: register state.Command()
    state/
      README.md       — Create: package documentation
      state.go        — Create: Command() returning the "state" cobra command group
      pull.go         — Create: pullCommand() and runPull()
      push.go         — Create: pushCommand() and runPush()
      sync.go         — Create: syncCommand() and runSync()
      resolve.go      — Create: resolveStateRepoPath() shared project resolution logic
      errors.go       — Create: exitError type (per-package convention)
      pull_test.go    — Create: tests for pull command
      push_test.go    — Create: tests for push command
      sync_test.go    — Create: tests for sync command
```

---

## Task 1: Add SyncPolicy and SyncConfig types to state package

**Files:**
- Modify: `pkg/state/types.go`

- [ ] **Step 1: Write test for SyncPolicy string constants**

```go
// pkg/state/types_test.go
package state

import (
	"testing"
	"time"
)

func TestSyncPolicyValues(t *testing.T) {
	tests := []struct {
		policy SyncPolicy
		want   string
	}{
		{SyncOnCommit, "on_commit"},
		{SyncOnInterval, "on_interval"},
		{SyncOnSessionEnd, "on_session_end"},
		{SyncManual, "manual"},
	}
	for _, tt := range tests {
		if string(tt.policy) != tt.want {
			t.Errorf("SyncPolicy %q != %q", tt.policy, tt.want)
		}
	}
}

func TestSyncConfigDefaults(t *testing.T) {
	var cfg SyncConfig
	if cfg.Pull != "" {
		t.Error("zero-value Pull should be empty string")
	}
	// Verify the type compiles with all fields
	cfg = SyncConfig{
		Pull:         SyncOnCommit,
		PullInterval: 0,
		Push:         SyncOnInterval,
		PushInterval: 5 * time.Minute,
	}
	if cfg.Pull != SyncOnCommit {
		t.Error("unexpected Pull value")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/state/ -run TestSyncPolicy -v`
Expected: FAIL — `SyncPolicy` type not yet defined

- [ ] **Step 3: Add types to types.go**

Add to end of `pkg/state/types.go`:

```go
import "time"

// SyncPolicy controls when the store automatically syncs with the remote.
type SyncPolicy string

const (
	// SyncOnCommit syncs after every merge to local main. Default.
	SyncOnCommit SyncPolicy = "on_commit"

	// SyncOnInterval syncs on a timer.
	SyncOnInterval SyncPolicy = "on_interval"

	// SyncOnSessionEnd syncs when the agent session ends.
	SyncOnSessionEnd SyncPolicy = "on_session_end"

	// SyncManual syncs only via explicit Pull/Push/Sync calls.
	SyncManual SyncPolicy = "manual"
)

// SyncConfig holds the sync policy for automatic pull/push behaviour.
// Pull and push policies are independent. Both default to SyncOnCommit.
type SyncConfig struct {
	Pull         SyncPolicy
	PullInterval time.Duration // used when Pull is SyncOnInterval
	Push         SyncPolicy
	PushInterval time.Duration // used when Push is SyncOnInterval
}
```

Also add `Sync SyncConfig` field to the existing `StoreOptions` struct.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/state/ -run TestSyncPolicy -v`
Expected: PASS

- [ ] **Step 5: Run full validation**

Run: `gofmt -w pkg/state/types.go pkg/state/types_test.go && golangci-lint run ./pkg/state/... && go test ./pkg/state/... && go build ./... && go vet ./...`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add pkg/state/types.go pkg/state/types_test.go
git commit -m "feat(state): add SyncPolicy and SyncConfig types"
```

---

## Task 2: Add StateSync interface and update git backend (atomic — keeps build green)

**Files:**
- Create: `pkg/state/sync.go`
- Modify: `pkg/state/store.go`
- Modify: `pkg/state/gitstore/gitstore.go`
- Create: `pkg/state/gitstore/sync.go`

**Why one task:** Adding `State()` to `Store` and updating gitstore must happen atomically — otherwise the build breaks between commits.

- [ ] **Step 1: Write test verifying Store interface includes State() accessor**

```go
// pkg/state/store_test.go
package state_test

import (
	"context"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// Compile-time check that Store requires State() accessor.
type mockStore struct{}

func (m *mockStore) Task() state.TaskStore       { return nil }
func (m *mockStore) Chat() state.ChatStore       { return nil }
func (m *mockStore) Project() state.ProjectStore { return nil }
func (m *mockStore) State() state.StateSync      { return nil }

var _ state.Store = (*mockStore)(nil)

func TestStateSyncInterface(t *testing.T) {
	// Verify StateSync interface has all three methods via compile-time check.
	var _ state.StateSync = (*mockStateSync)(nil)
}

type mockStateSync struct{}

func (m *mockStateSync) Pull(_ context.Context) error { return nil }
func (m *mockStateSync) Push(_ context.Context) error { return nil }
func (m *mockStateSync) Sync(_ context.Context) error { return nil }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/state/ -run TestStateSync -v`
Expected: FAIL — `StateSync` interface and `State()` method not defined

- [ ] **Step 3: Create pkg/state/sync.go**

```go
package state

// Features implemented: state-store

import "context"

// StateSync provides manual synchronization controls for the state store.
// Backends without a remote concept (e.g., SQLite) return a no-op implementation.
type StateSync interface {
	// Pull fetches the latest state from the remote.
	Pull(ctx context.Context) error

	// Push sends local state to the remote.
	Push(ctx context.Context) error

	// Sync performs a full round-trip: pull then push.
	Sync(ctx context.Context) error
}
```

- [ ] **Step 4: Add State() accessor to Store interface in store.go**

In `pkg/state/store.go`, add to the `Store` interface:

```go
	// State returns the sync sub-interface for manual sync controls.
	State() StateSync
```

- [ ] **Step 5: Create gitstore/sync.go with stub StateSync implementation**

Note: `errNotImplemented` is already defined in `pkg/state/gitstore/gitstore.go` — reuse it, do not duplicate.

```go
package gitstore

// Features implemented: state-store/backends/git
// Features depended on:  state-store

import (
	"context"

	"github.com/synchestra-io/synchestra/pkg/state"
)

type gitStateSync struct {
	store *GitStateStore
}

func (s *gitStateSync) Pull(_ context.Context) error { return errNotImplemented }
func (s *gitStateSync) Push(_ context.Context) error { return errNotImplemented }
func (s *gitStateSync) Sync(_ context.Context) error { return errNotImplemented }

// Compile-time interface check.
var _ state.StateSync = (*gitStateSync)(nil)
```

- [ ] **Step 6: Update gitstore.go — replace SyncMode with SyncConfig, add State() accessor**

Replace the `SyncMode`, `Options`, and `GitStateStore` types:

```go
// GitStoreOptions holds git-backend-specific configuration.
type GitStoreOptions struct {
	state.StoreOptions          // embeds shared options including SyncConfig
	RunID              string   // agent branch: agent/<run-id>
}

// GitStateStore is the git-backed implementation of state.Store.
type GitStateStore struct {
	stateRepoPath string
	specRepoPaths []string
	sync          state.SyncConfig
	runID         string
}
```

Update `New()`:

```go
func New(_ context.Context, opts GitStoreOptions) (state.Store, error) {
	sync := opts.Sync
	if sync.Pull == "" {
		sync.Pull = state.SyncOnCommit
	}
	if sync.Push == "" {
		sync.Push = state.SyncOnCommit
	}
	return &GitStateStore{
		stateRepoPath: opts.StateRepoPath,
		specRepoPaths: opts.SpecRepoPaths,
		sync:          sync,
		runID:         opts.RunID,
	}, nil
}
```

Add `State()` accessor:

```go
func (s *GitStateStore) State() state.StateSync { return &gitStateSync{store: s} }
```

Remove the old `SyncMode` type, `SyncModeSync`/`SyncModeLocal` constants, and the old `Options` struct entirely. The old `Options.StateRepoPath` and `Options.SpecRepoPaths` fields are now accessed through the embedded `state.StoreOptions` (field paths stay the same due to embedding promotion, e.g., `opts.StateRepoPath` still works). The `opts.Sync` field replaces the old `opts.SyncMode`. Update any callers of `gitstore.New()` (currently only `pkg/cli/project/` and tests) to pass `GitStoreOptions` instead of `Options`.

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./pkg/state/... -v`
Expected: PASS

- [ ] **Step 8: Run full validation**

Run: `gofmt -w pkg/state/sync.go pkg/state/store.go pkg/state/store_test.go pkg/state/gitstore/gitstore.go pkg/state/gitstore/sync.go && golangci-lint run ./... && go test ./... && go build ./... && go vet ./...`
Expected: All pass

- [ ] **Step 9: Commit**

```bash
git add pkg/state/sync.go pkg/state/store.go pkg/state/store_test.go pkg/state/gitstore/gitstore.go pkg/state/gitstore/sync.go
git commit -m "feat(state): add StateSync interface, update gitstore to use SyncConfig"
```

---

## Task 3: Create CLI state command group with pull/push/sync (TDD)

**Files:**
- Create: `pkg/cli/state/README.md`
- Create: `pkg/cli/state/state.go`
- Create: `pkg/cli/state/pull.go`, `pkg/cli/state/push.go`, `pkg/cli/state/sync.go`
- Create: `pkg/cli/state/pull_test.go`, `pkg/cli/state/push_test.go`, `pkg/cli/state/sync_test.go`
- Create: `pkg/cli/state/resolve.go`
- Create: `pkg/cli/state/errors.go`
- Modify: `pkg/cli/main.go`

- [ ] **Step 1: Create pkg/cli/state/README.md**

```markdown
# Package: cli/state

CLI commands for manual state repository synchronization.

## Commands

- `synchestra state pull` — pull latest state from origin
- `synchestra state push` — push local state to origin
- `synchestra state sync` — full bidirectional sync

## Outstanding Questions

None at this time.
```

- [ ] **Step 2: Create pkg/cli/state/errors.go**

```go
package state

// Features implemented: cli/state

// exitError carries an exit code for CLI error handling.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }
func (e *exitError) ExitCode() int { return e.code }
```

- [ ] **Step 3: Create pkg/cli/state/resolve.go**

```go
package state

// Features implemented: cli/state
// Features depended on:  project-definition

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// specRepoConfig is the minimal structure of synchestra-spec-repo.yaml
// needed to resolve the state repo path.
type specRepoConfig struct {
	StateRepo string `yaml:"state_repo"`
}

// stateRepoConfig is the minimal structure of synchestra-state-repo.yaml
// for direct detection when running from within the state repo.
type stateRepoConfig struct {
	Title string `yaml:"title"`
}

// resolveStateRepoPath finds the state repo path for the current project.
// It walks up from startDir looking for synchestra-spec-repo.yaml (reads
// state_repo field) or synchestra-state-repo.yaml (direct detection).
func resolveStateRepoPath(startDir string) (string, error) {
	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		// Check for spec repo config (spec repo → state repo via state_repo field)
		specPath := filepath.Join(current, "synchestra-spec-repo.yaml")
		if _, err := os.Stat(specPath); err == nil {
			data, err := os.ReadFile(specPath)
			if err != nil {
				return "", fmt.Errorf("reading %s: %w", specPath, err)
			}
			var cfg specRepoConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return "", fmt.Errorf("parsing %s: %w", specPath, err)
			}
			if cfg.StateRepo == "" {
				return "", &exitError{code: 3, msg: fmt.Sprintf("no state_repo field in %s", specPath)}
			}
			// TODO: Resolve state_repo URL to local path using repos_dir convention
			return cfg.StateRepo, nil
		}

		// Check for state repo config (direct detection)
		statePath := filepath.Join(current, "synchestra-state-repo.yaml")
		if _, err := os.Stat(statePath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", &exitError{code: 3, msg: "project not found: no synchestra-spec-repo.yaml or synchestra-state-repo.yaml in any parent directory"}
		}
		current = parent
	}
}
```

- [ ] **Step 4: Create pkg/cli/state/state.go**

```go
package state

// Features implemented: cli/state

import "github.com/spf13/cobra"

// Command returns the "state" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "State repository synchronization — pull, push, sync",
	}
	cmd.AddCommand(
		pullCommand(),
		pushCommand(),
		syncCommand(),
	)
	return cmd
}
```

- [ ] **Step 5: Write tests for pull/push/sync commands**

Create `pkg/cli/state/pull_test.go`:

```go
package state

import (
	"bytes"
	"testing"
)

func TestPullCommand_Help(t *testing.T) {
	cmd := pullCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected help output, got empty string")
	}
}

func TestPullCommand_AcceptsProjectFlag(t *testing.T) {
	cmd := pullCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--project", "test-project"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from stub implementation")
	}
	ec, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if ec.ExitCode() != 10 {
		t.Errorf("expected exit code 10, got %d", ec.ExitCode())
	}
}

func TestPullCommand_RejectsExtraArgs(t *testing.T) {
	cmd := pullCommand()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"extra-arg"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for extra args")
	}
}
```

Create `pkg/cli/state/push_test.go` and `pkg/cli/state/sync_test.go` with the same pattern, using `pushCommand()` and `syncCommand()` respectively.

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./pkg/cli/state/ -v`
Expected: FAIL — `pullCommand`, `pushCommand`, `syncCommand` not defined

- [ ] **Step 7: Create pkg/cli/state/pull.go**

```go
package state

// Features implemented: cli/state/pull
// Features depended on:  state-store, state-store/backends/git

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest state from origin to local main",
		Args:  cobra.NoArgs,
		RunE:  runPull,
	}
	cmd.Flags().String("project", "", "project identifier (autodetected from current directory if omitted)")
	return cmd
}

func runPull(cmd *cobra.Command, _ []string) error {
	// TODO: Resolve project, construct store, call store.State().Pull(ctx)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "state pull: not implemented yet")
	return &exitError{code: 10, msg: "synchestra state pull is not yet implemented"}
}
```

- [ ] **Step 8: Create pkg/cli/state/push.go and sync.go (same pattern)**

Same structure as pull.go. `push.go` implements `pushCommand()`/`runPush()`, `sync.go` implements `syncCommand()`/`runSync()`. Each has its own `// Features implemented:` comment (`cli/state/push` and `cli/state/sync` respectively).

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./pkg/cli/state/ -v`
Expected: PASS

- [ ] **Step 10: Register state command in pkg/cli/main.go**

Add import: `statecmd "github.com/synchestra-io/synchestra/pkg/cli/state"`

Add to `rootCmd.AddCommand(...)`: `statecmd.Command(),`

- [ ] **Step 11: Run full validation**

Run: `gofmt -w pkg/cli/state/*.go pkg/cli/main.go && golangci-lint run ./... && go test ./... && go build ./... && go vet ./...`
Expected: All pass

- [ ] **Step 12: Verify commands are registered**

Run: `go run . state --help`
Expected: Shows pull, push, sync subcommands

- [ ] **Step 13: Commit**

```bash
git add pkg/cli/state/ pkg/cli/main.go
git commit -m "feat(cli): add synchestra state pull/push/sync command stubs with tests"
```

---

## Task 4: Add SyncConfig parsing from YAML

**Files:**
- Create: `pkg/state/syncconfig.go`
- Create: `pkg/state/syncconfig_test.go`

- [ ] **Step 1: Write test for parsing sync config from YAML values**

```go
// pkg/state/syncconfig_test.go
package state

import (
	"testing"
	"time"
)

func TestParseSyncPolicy(t *testing.T) {
	tests := []struct {
		input    string
		policy   SyncPolicy
		interval time.Duration
		wantErr  bool
	}{
		{"on_commit", SyncOnCommit, 0, false},
		{"on_interval=5m", SyncOnInterval, 5 * time.Minute, false},
		{"on_interval=30s", SyncOnInterval, 30 * time.Second, false},
		{"on_session_end", SyncOnSessionEnd, 0, false},
		{"manual", SyncManual, 0, false},
		{"on_interval", "", 0, true},  // missing duration
		{"unknown", "", 0, true},
		{"", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			policy, interval, err := ParseSyncPolicy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if policy != tt.policy {
				t.Errorf("policy = %q, want %q", policy, tt.policy)
			}
			if interval != tt.interval {
				t.Errorf("interval = %v, want %v", interval, tt.interval)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/state/ -run TestParseSyncPolicy -v`
Expected: FAIL — `ParseSyncPolicy` not defined

- [ ] **Step 3: Implement ParseSyncPolicy in pkg/state/syncconfig.go**

```go
package state

// Features implemented: state-store
// Features depended on:  project-definition

import (
	"fmt"
	"strings"
	"time"
)

// ParseSyncPolicy parses a sync policy string value from configuration.
// Handles plain values ("on_commit", "manual") and parameterized values
// ("on_interval=5m").
func ParseSyncPolicy(s string) (SyncPolicy, time.Duration, error) {
	switch {
	case s == string(SyncOnCommit):
		return SyncOnCommit, 0, nil
	case s == string(SyncOnSessionEnd):
		return SyncOnSessionEnd, 0, nil
	case s == string(SyncManual):
		return SyncManual, 0, nil
	case strings.HasPrefix(s, string(SyncOnInterval)+"="):
		durStr := strings.TrimPrefix(s, string(SyncOnInterval)+"=")
		d, err := time.ParseDuration(durStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid interval duration %q: %w", durStr, err)
		}
		return SyncOnInterval, d, nil
	case s == string(SyncOnInterval):
		return "", 0, fmt.Errorf("on_interval requires a duration (e.g., on_interval=5m)")
	default:
		return "", 0, fmt.Errorf("unknown sync policy %q", s)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/state/ -run TestParseSyncPolicy -v`
Expected: PASS

- [ ] **Step 5: Run full validation**

Run: `gofmt -w pkg/state/syncconfig.go pkg/state/syncconfig_test.go && golangci-lint run ./... && go test ./... && go build ./... && go vet ./...`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add pkg/state/syncconfig.go pkg/state/syncconfig_test.go
git commit -m "feat(state): add ParseSyncPolicy for YAML config parsing"
```
