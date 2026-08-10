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

The durable outbox currently records **pending** per-replica delivery work.
Drain, acknowledgement, and recovery scheduling are **Planned**; this
foundation replicates from the authority journal and does not claim that its
write-only outbox records provide recovery.

## Open Questions

None at this time.
