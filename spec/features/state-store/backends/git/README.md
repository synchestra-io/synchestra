# Backend: Git State Store

**Parent:** [Backends](../)

**Status:** Conceptual

## Summary

The git backend (`gitstore`) is the default `state.Store` implementation. It maps every interface method to file operations, markdown table rendering, and atomic commit-and-push in the [state repository](../../../../architecture/repository-types.md#state-repository).

This backend requires no external infrastructure — only a git remote. It is the reference implementation against which all other backends are validated.

## Implementation Location

`pkg/state/gitstore/`

## Method Mapping

### TaskStore

| Interface Method | Git Operation |
|---|---|
| `Task().Create()` | Create `tasks/{slug}/README.md` with metadata, update board table, commit |
| `Task().Get()` | Parse `tasks/{slug}/README.md` |
| `Task().List()` | Scan `tasks/` directories, parse each README, apply filter |
| `Task().Enqueue()` | Update status field in `tasks/{slug}/README.md`, commit, push |
| `Task().Claim()` | Update status → claimed + claim metadata, commit, push (fail on conflict = another agent won) |
| `Task().Start()` | Update status → in_progress, commit, push |
| `Task().Complete()` | Update status → completed + summary, move board row to recently-finished, commit, push |
| `Task().Fail()` | Update status → failed + reason, move board row to recently-finished, commit, push |
| `Task().Block()` | Update status → blocked + reason, commit, push |
| `Task().Unblock()` | Update status → in_progress, commit, push |
| `Task().Release()` | Update status → queued, clear claim metadata, commit, push |
| `Task().RequestAbort()` | Set `abort_requested: true` in task README, commit, push |
| `Task().ConfirmAbort()` | Update status → aborted, move board row to recently-finished, commit, push |

### Board

| Interface Method | Git Operation |
|---|---|
| `Task().Board().Rebuild()` | Regenerate `tasks/README.md` markdown table from all task READMEs |
| `Task().Board().Get()` | Parse `tasks/README.md` markdown table into `BoardView` |

### ArtifactStore (Task-Scoped)

| Interface Method | Git Operation |
|---|---|
| `Task().Artifact(ctx, slug).Put()` | Write file to `tasks/{slug}/artifacts/{name}`, commit |
| `Task().Artifact(ctx, slug).Get()` | Read file from `tasks/{slug}/artifacts/{name}` |
| `Task().Artifact(ctx, slug).List()` | List files in `tasks/{slug}/artifacts/` |

### ChatStore

| Interface Method | Git Operation |
|---|---|
| `Chat().Create()` | Create `chats/{id}/README.md` with metadata, commit |
| `Chat().Get()` | Parse `chats/{id}/README.md` |
| `Chat().List()` | Scan `chats/` directories, parse each README, apply filter |
| `Chat().Finalize()` | Update status → finalized, flush messages to `history.jsonl`, commit, push |
| `Chat().Abandon()` | Update status → abandoned, flush messages, commit, push |
| `Chat().AppendMessages()` | Append to server-side buffer (flushed to git on finalize/checkpoint) |
| `Chat().Messages()` | Read from `chats/{id}/history.jsonl` |

### ArtifactStore (Chat-Scoped)

| Interface Method | Git Operation |
|---|---|
| `Chat().Artifact(ctx, id).Put()` | Write file to `chats/{id}/artifacts/{name}`, commit |
| `Chat().Artifact(ctx, id).Get()` | Read file from `chats/{id}/artifacts/{name}` |
| `Chat().Artifact(ctx, id).List()` | List files in `chats/{id}/artifacts/` |

### ProjectStore

| Interface Method | Git Operation |
|---|---|
| `Project().Config()` | Read and parse `synchestra-state-repo.yaml` |
| `Project().UpdateConfig()` | Write `synchestra-state-repo.yaml`, commit |
| `Project().RebuildREADME()` | Regenerate root `README.md` from project state |

## Sync Mode

The git backend supports configurable sync behavior via `GitStoreOptions.SyncMode`:

| Mode | Pull before read | Push after write | Use case |
|---|---|---|---|
| `sync` | Yes | Yes | Multi-host: multiple agents on different machines. Every operation hits the remote to ensure consistency. This is the default. |
| `local` | No | No | Single-host: one machine runs all agents against a local clone. No network I/O per operation. The caller is responsible for periodic `git pull`/`git push` (e.g., via cron or a background goroutine). |

```go
type SyncMode string

const (
    SyncModeSync  SyncMode = "sync"  // pull before read, push after write (default)
    SyncModeLocal SyncMode = "local" // local-only, no remote I/O per operation
)

type GitStoreOptions struct {
    StateRepoPath string
    SpecRepoPaths []string
    SyncMode      SyncMode // defaults to SyncModeSync
}
```

In `local` mode:
- **Reads** operate directly on the working tree — no `git pull` first.
- **Writes** commit locally but do not push. The working tree is always up-to-date with the latest local commit.
- **Claim atomicity** is still guaranteed within the host (file-level locking or similar), but not across hosts. Running multiple hosts in `local` mode against the same remote will cause conflicts.
- **Periodic sync** is the caller's responsibility. A typical setup pushes every N seconds or on a signal.

In `sync` mode:
- Every mutating operation follows the full pull → validate → commit → push protocol.
- Read operations pull first to ensure freshness.
- This is the safe default for distributed agent setups.

## Atomicity

The git backend relies on git's push-or-fail semantics for atomicity in `sync` mode. The protocol for mutating operations:

1. Pull latest state from remote
2. Validate preconditions (task exists, correct status, etc.)
3. Update local files
4. Commit
5. Push
6. On push conflict: pull, re-verify preconditions, retry or fail

In `local` mode, atomicity is scoped to the local host (file-level locking prevents concurrent local writes from corrupting state).

See [Task Status Board: Claiming a Task](../../../task-status-board/README.md#claiming-a-task-optimistic-locking) for the detailed claiming protocol and conflict resolution.

## Performance Characteristics

| Operation | Cost (`sync` mode) | Cost (`local` mode) | Notes |
|---|---|---|---|
| Read (Get, List) | Pull + file I/O | File I/O only | `local` avoids network round-trip |
| Write (Create, status transitions) | File I/O + commit + push | File I/O + commit | `local` defers push |
| Claim (contended) | File I/O + commit + push + retry | File I/O + commit + local lock | `local` has no remote contention |
| Board Rebuild | Scan all task directories | Same | O(n) in number of tasks regardless of mode |

For single-host setups, `local` mode eliminates network overhead entirely — mutations become local file + commit operations. Periodic push keeps the remote in sync for visibility (dashboards, other tools) without blocking the hot path.

For projects with hundreds of concurrent agents or thousands of tasks, database backends may offer better performance. See the [backend matrix](../) for alternatives.

## Outstanding Questions

- Should the git backend batch multiple mutations into a single commit when they occur within a short window (e.g., creating a task and immediately enqueuing it)?
- How should the git backend handle partial failures (e.g., commit succeeds but push fails due to network error)?
- Should `AppendMessages` write to a local buffer file or hold messages in memory until finalize?
