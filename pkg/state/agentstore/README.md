# pkg/state/agentstore

Backend-neutral implementation of `state.AgentStore` — the `Effort()/Run()/Worktree()/Message()/Activity()/Journal()/Cursor()/Lease()/Health()` sub-store composed on `state.Store` alongside `Task()/Chat()/Project()/State()`.

## Why this package exists

`pkg/state/replication` already abstracts Git/inGitDB and SQLite/DALgo behind one `Journal` interface (`Append`/`After`/`Head`) with role/epoch fencing built in (`RoleFenceError`, `ErrEpochFenced`). Since that abstraction is backend-neutral by design, the effort/run/worktree-claim/message/activity/lease domain logic above it only needs to be written once — here — rather than once per backend (`gitstore`, a future `sqlitestore`, ...). Each backend's `Store.Agent()` method constructs a `replication.Journal` for its own storage and hands it to `agentstore.New`, returning the resulting `*agentstore.Store` directly. See `pkg/state/gitstore/agent.go` for the Git wiring.

## How it works

Every write (`Effort().Create`, `Run().Start`, `Worktree().Claim`, ...) builds a small JSON payload and appends one `replication.Event` through the journal. Every read replays `Journal.After(Cursor{})` and folds the relevant event kinds into an in-memory projection — there is no cached or incrementally-maintained state. This trades read throughput for determinism and simplicity, the same tradeoff `pkg/state/gitstore/board.go` documents for `Task().Board().Rebuild()` scanning every task directory.

Exclusivity (`agent-coordination#ac:one-writer-claim-is-fenced`) and fencing (`state-store/topology#ac:promotion-fences-former-active`) both funnel through `LeaseStore` (`lease.go`): `Acquire` re-derives its "is this resource already held" projection on every retry of a lost sequence race, so two concurrent callers targeting the same resource key can never both succeed — and any Store instance whose configured `AuthorityEpoch` has been fenced out by a promotion recorded through a different Store instance gets `state.ErrLeaseFenced` (wrapping `replication.RoleFenceError`/`ErrEpochFenced`) on its very next write. `WorktreeStore.Claim/Renew/Release` (`worktree.go`) delegate directly to `LeaseStore` rather than re-implementing that race handling.

## Schema and migration ownership

This package's only persistence surface is the journal, so it has no DALgo table schema to migrate (that belongs to each backend, e.g. `replication.EnsureDALJournalSchema` today, a future `sqlitestore`'s DALgo migrations later). Its schema is the set of event `Kind` strings and JSON payload shapes declared in `schema.go`, each pinned to an explicit version (`.../v1`) the same way `replication.EventSchemaV1` and `replication.fallbackKinds` are pinned. A payload change adds a new versioned `Kind`; it never mutates an existing one in place. `EventKinds()` enumerates the full owned vocabulary. `message.sent`/`message.acknowledged` deliberately reuse the exact strings `replication.fallbackKinds` already allow-lists, so a future transport-switch reconciliation (task-3) has one shared vocabulary rather than two.

## Status

Implements Task 1 of `spec/plans/synchestra-coordination-foundation.md`: the contract surface, Git wiring, and claim/lease fencing. It deliberately does **not** implement task-3's effort/run lifecycle state machine (`planning`/`active`/`handoff_pending`/.../`archived`), CLI verbs, scope-overlap detection, or the topology `Promote()` administrative workflow.

## Spec

See `spec/features/state-store/topology/` and `spec/features/agent-coordination/`.

## Open Questions

- `Claim` acquires a lease and then appends a `worktree.claimed` follow-up event as two separate journal writes; if the process crashes between them, the lease is held with no matching claim record visible via `Worktree().Get`. `Journal` has no multi-event transaction primitive to close this gap; task-3's recovery/handoff work should decide whether an orphaned-lease reconciliation pass is worth adding here or is subsumed by its own recovery records.
- Message/activity/effort/run writes reuse a fresh event ID as both `EventID` and `IdempotencyKey` on every call, so a client-side retry of the identical logical command is not deduplicated (only the benign "someone else advanced the journal tail" race is retried transparently). Command-level idempotent retry, if needed, is a follow-up.
- `HealthStore.Report` always labels this Store's own perspective `"active"`; it does not yet distinguish a Store wired to a replica-role `Journal`. Task 5/6 (replica workers, promotion) should decide whether that belongs here or in a thin wrapper closer to the topology config.
