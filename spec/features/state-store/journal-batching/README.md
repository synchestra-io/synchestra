---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Journal Group-Commit Batching

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/journal-batching?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/journal-batching?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/journal-batching?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/journal-batching?op=request-change) |
**Status:** Approved
**Source Ideas:** —
**Parent:** [State Store](../)

## Summary

The journal append path supports group commit: concurrent appends accumulate
into one batch that is committed in a single storage transaction when either a
configurable item count or a configurable time window is reached — whichever
comes first. On the Git-backed store this collapses what would be one commit
per event into one commit per batch; on SQLite it collapses per-event
transactions the same way. Batching is a write-path optimization only: it
changes commit granularity, never durability semantics, ordering, validation,
or the atomic outbox fan-out contract.

Configuration is two knobs, each with `0` meaning disabled:

- `max_items` — flush when the pending batch reaches this many events
  (default **100**).
- `max_delay_ms` — flush when this much time has elapsed since the first
  event entered the pending batch (default **1000**).

When both are `0`, every append commits immediately in its own transaction
(the pre-batching behavior). When either knob is non-zero, the first threshold
reached triggers the flush. Closing the journal — including normal process
exit — flushes the pending batch before returning, so short-lived CLI
invocations are not delayed by the time window.

## Problem

Every journal append currently commits its own storage transaction. On the
inGitDB backend that is one Git commit per event; a burst of N events (bulk
imports, high-frequency activity streams, future audited messaging) produces N
commits plus N fan-out writes, dominating write latency and bloating history.
The storage layer already supports rollback-safe multi-record transactions
(`ingitdb/dalgo2ingitdb v0.3.1`), so the only missing piece is an append path
that accumulates and commits in groups without weakening the journal's crash
guarantees.

## Semantics

### REQ: group-commit-not-buffering

Batching is group commit, not fire-and-forget buffering: an `Append` call does
not return success until the transaction containing its event has durably
committed. Callers block for at most the remaining time window. There is no
window in which an acknowledged event exists only in memory, and no window in
which an event is durable without its outbox fan-out rows — the entire batch
(events, head advance, all per-replica outbox rows) commits in one
transaction.

### REQ: per-event-contracts-survive-batching

Sequence validation, role/epoch fencing, and content checks apply per event
exactly as in unbatched mode. Events within a batch commit in cursor order. A
single rejected event fails only its own `Append`; the remaining events in the
batch still commit.

## Acceptance Criteria

### AC: batch-flushes-at-item-threshold

**Given** a journal configured with `max_items: N` (N > 0)
**When** the Nth event enters the pending batch before the time window elapses
**Then** the batch commits immediately in one storage transaction containing
all N events, their per-replica outbox rows, and a single head advance — one
Git commit on the inGitDB backend.

### AC: batch-flushes-at-time-threshold

**Given** a journal configured with `max_delay_ms: K` (K > 0)
**When** fewer than `max_items` events have accumulated and K milliseconds
have elapsed since the first pending event entered the batch
**Then** the batch commits at that point; no event waits longer than K beyond
its append call.

### AC: zero-disables-each-dimension

**Given** `max_items: 0` and/or `max_delay_ms: 0`
**When** events are appended
**Then** a zeroed knob contributes no flush trigger, and with both knobs zero
every event commits immediately in its own transaction, byte-identical in
observable behavior to the pre-batching journal.

### AC: defaults-are-100-items-1000ms

**Given** a journal constructed without explicit batching configuration
**When** its effective configuration is inspected
**Then** `max_items` is 100 and `max_delay_ms` is 1000.

### AC: append-acknowledges-only-durable-commits

**Given** any batching configuration
**When** a process crashes before a pending batch has committed
**Then** no crashed `Append` call had already returned success, the store
contains only fully committed batches, and recovery finds no event without its
outbox fan-out rows or head advance.

### AC: close-flushes-pending-batch

**Given** a journal with a non-empty pending batch
**When** the journal is closed, including via normal process exit
**Then** the pending batch commits before close returns, and a one-shot append
in a short-lived process is not delayed by the time window.

### AC: batched-events-preserve-fencing-and-order

**Given** a batch containing a mix of valid events and one event rejected by
sequence validation or role/epoch fencing
**When** the batch flushes
**Then** the rejected event's `Append` alone returns the fencing error, the
valid events commit in cursor order, and replicas draining the outbox observe
the same order and fan-out as unbatched appends would have produced.

## Open Questions

- Should a lone appender flush early when no other append is in flight,
  instead of waiting out the time window? Today the answer is no — the exit
  flush covers one-shot CLI processes, and long-running servers amortize the
  window — but interactive latency evidence may reopen this.
