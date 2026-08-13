# pkg/state/gitstore

Git-backed implementation of `state.Store`. Maps every interface method to file operations, markdown table rendering, and atomic commit-and-push in a Synchestra state repository.

This is the default state store backend — it requires no external infrastructure beyond a git remote.

## Status

`Task()`/`Chat()`/`Project()` are stub implementations — all methods return `errNotImplemented`; the full implementation is tracked separately. `Agent()` (`agent.go`) is implemented: it wires `pkg/state/agentstore`'s backend-neutral domain logic to a real Git-backed `replication.Journal` via `dalgo2ingitdb`/DALgo (state-store/topology, Task 1), with `state-store/journal-batching`'s documented default group-commit batching (100 items/1000ms) as of task-3.

### Agent() caching and Close (task-3)

`GitStateStore.Agent()` constructs its `*agentstore.Store` (and the `replication.Journal` beneath it) once per `GitStateStore` instance and caches it (`agentOnce`/`agentCore`/`agentErr`) — every subsequent `Agent()` call on the SAME `GitStateStore` returns the SAME instance. This matters for two things batching needs: concurrent writers sharing one `GitStateStore` actually share a pending batch (rather than each opening its own single-item "batch"), and `GitStateStore.Close` closes the journal that was actually written to rather than a freshly-constructed empty one. A distinct `GitStateStore` (a restarted CLI process, or a test's own `newStore()`-per-call helper) still gets its own fresh journal, exactly as before this change.

`GitStateStore.Close(ctx)` flushes any pending group-commit batch and must be called before a one-shot CLI process exits for a lone pending write to return promptly rather than waiting out the batching window — see `pkg/state/agentstore/README.md`'s "Close" section and Open Questions, and `pkg/state/closeafter.go`'s `CloseAfter` for the safe (single-`Append`-only) way to race it against an in-flight write.

## Spec

See `spec/features/state-store/backends/git/` for the method-to-git-operation mapping, and `spec/features/state-store/topology/` plus `spec/features/agent-coordination/` for `Agent()`.

## Open Questions

- `Task()`/`Chat()`/`Project()` remain unimplemented stubs; when they land, confirm whether they should share `Agent()`'s per-instance construction-caching pattern or have their own lifecycle needs.
