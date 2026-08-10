---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: SQLite State Store

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends/sqlite?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends/sqlite?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends/sqlite?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends/sqlite?op=request-change) |
**Status:** Approved
**Source Ideas:** —

## Summary

DALgo-backed SQLite state storage owned by the Synchestra server for low-latency coordination and indexed activity queries.

## Problem

Git remains the mandatory portable audit and recovery medium, but it is costly
for short-lived claims, presence, mailbox reads, and repository-wide activity
queries. A Synchestra server needs one transactional, indexed local endpoint
without making SQLite a second writer or a private replacement for the Git
record.

## Behavior

### DALgo endpoint, not a Synchestra-specific database

`sqlitestore` is a DALgo-backed implementation of the backend-neutral
`state.Store` contract. It uses `dalgo2sqlite`; domain packages do not import
SQLite drivers, SQL strings, filesystem paths, or Git packages. Schema changes
are versioned DALgo migrations. Synchestra owns its domain schema and migration
history, but it does not implement a parallel persistence abstraction.

The server owns the SQLite file, opens it with a single-writer transaction
discipline, enables foreign keys and a bounded busy timeout, and takes an
operator-visible backup/checkpoint before each incompatible migration. A local
path is never shared by multiple servers. Multi-host deployment is a future
backend, not a reason to weaken SQLite's ownership model.

### Records and atomic transitions

SQLite stores domain projections for projects, efforts, runs, worktree claims,
leases, messages, activity, and state-change metadata. Each authoritative
transition is one DALgo transaction that writes:

1. the domain projection and its optimistic precondition result;
2. an immutable `(authority_epoch, sequence, event_id)` journal entry;
3. one transactional-outbox delivery for every configured replica; and
4. an auditable command result containing the cursor.

The unique keys `(project_id, authority_epoch, sequence)`, `event_id`, and
`idempotency_key` make retries safe. A task/worktree claim uses a conditional
write tied to the current authority epoch and lease fence. Two concurrent
writers cannot both receive success. The server returns conflict information,
not a retry that might overwrite the winner.

### Authority roles and replication

SQLite can be either the active store or a mirror. In the founder-MVP default,
SQLite is active and the required Git/inGitDB endpoint is a mirror. The server
acknowledges once the SQLite transaction commits; a caller that needs portable
durability requests the Git mirror barrier and receives the Git commit SHA only
after it is contiguous at that cursor.

When Git is active, SQLite is read-only with respect to domain commands: the
server imports the verified Git journal in order and exposes explicitly labelled
replica reads. It never infers mutations from the current Git tree alone. The
same workload is run in both directions to compare latency, contention,
replication lag, recovery point, and recovery time.

SQLite rejects stale epochs. Before enabling writes after restart it fetches
and reconciles Git; a higher Git promotion epoch leaves it a mirror until an
explicit fenced promotion. No server outage or replica lag authorizes a hidden
second active SQLite writer.

### Server interfaces and observability

The server exposes low-latency command, query, streaming-notification, and
health interfaces. A query response identifies `active` or `replica`, endpoint
ID, authority epoch, cursor, and staleness. Health reports migration version,
outbox depth, per-replica cursor/lag, checksum failures, last successful
checkpoint, and recovery state. Metrics exclude message bodies, prompts,
credentials, and local work-log contents.

`synchestra state status`, `verify`, `replicate`, and `wait` are the
operator-facing controls defined by the topology Feature. SQLite-specific
maintenance is never used to silently change authority.

## Acceptance Criteria

### AC: sqlite-active-uses-one-transaction

**Given** SQLite is the active store with a required Git mirror
**When** an agent claims a worktree and the process is interrupted at every
transaction boundary
**Then** recovery finds either no claim/domain/journal/outbox change or exactly
one of each at one cursor; retrying the command ID never produces a second
claim.

### AC: sqlite-mirror-is-labelled-and-caught-up

**Given** Git is active and SQLite is configured as a mirror
**When** Git transitions are reconciled
**Then** SQLite reaches an identical verified cursor, serves indexed active and
recent views labelled `replica`, and rejects all domain writes.

### AC: sqlite-restart-obeys-fencing

**Given** SQLite was active, Git was promoted during server outage, and the old
server restarts
**When** it opens its database
**Then** it discovers the higher Git epoch before serving writes, imports it as
a replica, and rejects stale-epoch commands until explicit promotion.

### AC: git-barrier-proves-portable-durability

**Given** a terminal run update on SQLite active
**When** the caller asks to wait for the required Git mirror
**Then** success contains the original SQLite cursor and the Git commit SHA
that recorded it; timeout reports committed active state and unsatisfied mirror
durability without ambiguity.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
