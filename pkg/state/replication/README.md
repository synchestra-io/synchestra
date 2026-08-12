# State replication

`replication` contains the backend-neutral, ordered journal contract shared by
Git and SQLite state endpoints. It does not know how either backend stores a
record. Backends implement the small `Journal` interface; the domain sends one
immutable transition at a time and replication preserves its authority epoch,
sequence, event ID, correlation ID, and checksum chain.

The first consumer is typed agent communication: `message.sent`,
`message.acknowledged`, `worklog.checkpointed`, and
`repository.ref.updated`. The Git fallback is a separate, remote-durable inbox
for a narrow v1 communication allowlist; it is not a journal authority.
Reconciliation into the active journal is **Planned** and must allocate fresh
authoritative cursors exactly once.

The durable outbox records one pending per-replica delivery row per append,
in the same transaction as the domain write, journal event, and head.
`DrainOutbox` (`outbox.go`) is the consumer: it walks an `OutboxSource`'s
pending rows for one replica in cursor order, applies each through the same
idempotent `ReplicaIngestor` seam `Replicate` uses, and acknowledges (deletes)
the row only once the replica accepts it. Both halves are independently safe
to repeat — `Journal.Append`/`IngestReplica` dedupe by event ID, and deleting
an already-gone outbox row is a documented no-op on every DAL adapter this
package targets — so a crash between apply and ack, or two drain workers
racing over the same rows, never double-applies or loses a delivery.
`DrainOutbox` never decides convergence itself: `Replicate`'s head/cursor/
checksum comparison remains the source of truth for replica health, and the
two compose (an operator typically drains, then verifies, before treating a
replica as caught up). `*DALJournal` implements `OutboxSource` against the
real `outboxCollection`; `MemoryJournal` implements it in-memory for fast
conformance and concurrency tests.

Every physical journal is configured with an endpoint role and authority
epoch. Only the active endpoint accepts domain `Append`; replicas accept the
same immutable event solely through the explicit replication-ingest seam.
Git delivery serializes append/push operations in the worktree's private Git
directory and persists a checksummed intent before append, allowing authority
events and fallback envelopes to resume after any commit/receipt/push crash.

`Promote` (`promotion.go`) implements the explicit administrative promotion
workflow from `state-store/topology`'s "Promotion and Recovery" section: it
refuses a candidate replica that is not converged with the active (lag > 0 in
either direction, or diverged at an equal cursor), then signs one promotion
checkpoint event at the next authority epoch and durably records it on both
the candidate (which becomes active) and the former active (which becomes a
fenced replica) before switching either endpoint's local role. Any other
reachable required replica receives the same checkpoint through the ordinary
`IngestReplica` seam. `Promote` does not drain or catch up the candidate
itself — callers run `DrainOutbox`/`Replicate` first — and it requires a
reachable handle on the current active, which the founder-MVP single-host
SQLite topology always has even when the server process is down.

## Open Questions

None at this time.
