# State replication

`replication` contains the backend-neutral, ordered journal contract shared by
Git and SQLite state endpoints. It does not know how either backend stores a
record. Backends implement the small `Journal` interface; the domain sends one
immutable transition at a time and replication preserves its authority epoch,
sequence, event ID, correlation ID, and checksum chain.

The first consumer is typed agent communication: `message.sent`,
`message.acknowledged`, `worklog.checkpointed`, and
`repository.ref.updated`. These records stay append-only, so they are usable
both through the active store and through the Git fallback inbox.

## Open Questions

None at this time.
