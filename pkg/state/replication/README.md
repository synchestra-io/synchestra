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
every successfully-applied row in one batched `AckOutbox` call per drain
attempt (covering whatever was applied, whether the drain fully emptied the
outbox or stopped early on an ingest failure) instead of one transaction per
event. Both halves are independently safe to repeat — `Journal.Append`/
`IngestReplica` dedupe by event ID, and deleting an already-gone outbox row is
a documented no-op on every DAL adapter this package targets — so a crash
during the batched ack, or two drain workers racing over the same rows, never
double-applies or loses a delivery; an ingest failure partway through
(including one caused by a concurrent promotion fencing the replica) still
acks whatever was already applied and reports the exact resumable cursor/lag
for the rest. `DrainOutbox` never decides convergence itself: `Replicate`'s
head/cursor/checksum comparison remains the source of truth for replica
health, and the two compose (an operator typically drains, then verifies,
before treating a replica as caught up). `*DALJournal` implements
`OutboxSource` against the real `outboxCollection`; `MemoryJournal`
implements it in-memory for fast conformance and concurrency tests. Both
`DrainOutbox` and `Replicate` refuse a literal `nil` replica/source interface
value up front; neither attempts to detect a typed-nil concrete pointer via
reflection, since every constructor in this package returns a non-nil pointer
on success.

Every physical journal is configured with an endpoint role and authority
epoch. Only the active endpoint accepts domain `Append`; replicas accept the
same immutable event solely through the explicit replication-ingest seam.
`Append`/`IngestReplica` hold the endpoint's role/epoch lock for the entire
underlying append transaction, not just the precondition check — this is what
lets a concurrent write and a concurrent promotion never interleave (see
`RoleFenceError`'s doc comment). Git delivery serializes append/push
operations in the worktree's private Git directory and persists a
checksummed intent before append, allowing authority events and fallback
envelopes to resume after any commit/receipt/push crash.

## Promotion

`Promote` (`promotion.go`) implements the explicit administrative promotion
workflow from `state-store/topology`'s "Promotion and Recovery" section, using
a **fence-first** sequence — the former active is fenced *before* the
candidate is ever touched, never after:

1. **Fence the active first.** `FenceAsReplica` durably records the one
   promotion checkpoint event as the active's own next event — chaining it
   onto the active's *own current head*, re-derived fresh inside
   `FenceAsReplica`'s locked/transactional scope, never from a snapshot read
   earlier — then downgrades the active to a replica. After this step the
   former active refuses further domain writes: the system fails toward
   **zero** writers on any later failure in this call, never two.
2. **Catch the candidate up.** Promote re-reads the candidate's head and, if
   it lags the fenced active's new head, delivers the remaining events —
   including the checkpoint itself — via `Replicate`, sourcing from the
   now-fenced (but still readable) former active.
3. **Promote the candidate** using the *same* checkpoint, now durably present
   on it via ordinary replication. `PromoteToActive` only verifies the
   checkpoint is already the candidate's head and flips its role — it never
   appends the checkpoint a second time.
4. **Fan out** the checkpoint to any other reachable replicas. A delivery
   failure here is non-fatal: the swap in steps 1–3 already durably
   succeeded, so `Promote` still reports success and records the per-replica
   outcome in `PromotionResult.ReplicaOutcomes` instead of erroring out a
   promotion that worked because one lagging backup was unreachable.

This order closes four confirmed split-brain defects of the old
fence-*after*-promote design: a concurrent domain `Append` during the
promotion window can no longer permanently break the fence (`Append` and
`FenceAsReplica` hold the same role/epoch lock for their entire duration, so
they can never interleave); two concurrent `Promote` calls for different
candidates now serialize on step 1's role/epoch guard before either candidate
is touched, so at most one can ever win; a transient failure in step 1 leaves
the active completely untouched (nothing durable to roll back); and step 1's
own re-derived head is the single, non-bypassable re-validation point,
closing the old TOCTOU where the only check ran before the irreversible
candidate flip.

**Resumability.** A `Promote` call that fails after step 1 leaves the former
active durably fenced with zero endpoints currently active — by design, per
the fail-toward-zero-writers invariant above — and this state is resumable:
re-invoking `Promote` with the same active, candidate, and (explicit or
defaulted) `IdempotencyKey` detects the durable checkpoint `FenceAsReplica`
already left on the active and continues from step 2 rather than re-fencing,
refusing, or double-charging the epoch (`PromotionResult.Resumed` reports
which happened). `PromoteToActive` and the candidate side of catch-up are
themselves idempotent the same way, so a resumed call safely re-delivers
already-applied events and re-confirms an already-flipped role.

`PromotionRequest.ReplicaIDs` names the replica set the newly-active candidate
should fan future outbox rows out to — typically including the just-fenced
former active — since a promoted candidate otherwise keeps whatever (likely
stale or empty) replica set it was originally constructed with.

`Promote` operates over `PromotableJournal`, not a concrete backend type, so
it composes with `*GitPushJournal`: a Git-backed active or candidate fences
and promotes through the same durable CAS-push-plus-receipt machinery
`Append`/`IngestReplica` use, so a Git endpoint's checkpoint is never
committed locally without either being pushed or leaving a resumable pending
receipt behind (calling the raw `*DALJournal` fencing seam directly, bypassing
the push wrapper, was the original bug this composition exists to close).
`*MemoryJournal` also implements `PromotableJournal`, giving fast, `-race`-
clean concurrency tests a promotable harness without a real DB or Git
backend.

**Empty journals.** Promoting between an active and a candidate that both
have zero events is refused early with the typed `ErrPromotionEmptyJournals`
rather than failing deep inside the checkpoint append with a confusing
sequence-gap error. There is no meaningful authority to hand off between two
empty journals; establish the initial active through construction instead
(`NewDALJournal`/`NewMemoryJournal` with `Role: RoleActive,
AuthorityEpoch: 1`), never through `Promote`.

Typed sentinel errors (`ErrFenceSourceNotActive`, `ErrPromoteTargetNotReplica`,
`ErrPromoteTargetWrongEpoch`, `ErrPromotionSourceNotActive`,
`ErrPromotionCandidateNotReplica`, `ErrPromotionNotConverged`,
`ErrPromotionEmptyJournals`) let a caller distinguish a benign lost race (e.g.
someone else already promoted this candidate) from a dangerous failure with
`errors.Is`, rather than parsing a bare `fmt.Errorf` string.

## Open Questions

None at this time.
