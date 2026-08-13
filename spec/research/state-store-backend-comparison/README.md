# Research: State-Store Backend Comparison

**Status:** Recorded
**Feature:** [State Store / Topology](../../features/state-store/topology/README.md)
**Verifies:** `state-store/topology#ac:backend-comparison-is-equivalent`

## Summary

This record is the evidence `state-store/topology#ac:backend-comparison-is-equivalent`
asks for: the same conformance workload run once with **Git active, SQLite
mirror** and once with **SQLite active, Git mirror**, reporting latency,
throughput, conflicts, Git growth, mirror lag, and recovery time for both
directions, plus a statement of how domain-record/journal-checksum
equivalence is proven.

Numbers below are one recorded run on the machine described in [Machine
Context](#machine-context). They are not a performance SLA — they exist to
be reproduced and re-recorded as the implementation changes; see
[Reproducing](#reproducing).

## Correctness equivalence (checksums)

`ac:backend-comparison-is-equivalent`'s correctness half — "domain records
and journal checksums are equivalent" — is proven, not by this record, but
by pkg/state/replication's own physical test suite, which already runs an
identical event workload through both directions and asserts checksum-chain
parity:

- `TestDALJournal_PhysicalGitActiveReplicatesToSQLiteAndRestarts` (Git
  active → SQLite mirror)
- `TestDALJournal_PhysicalSQLiteActiveReplicatesToGitAndDoesNotDualWrite`
  (SQLite active → Git mirror)
- `TestDALJournal_OutboxDrainDeliversAndAcksAcrossBothPhysicalDirections`

These were delivered by task-2/task-4 of
[synchestra-coordination-foundation](../../plans/synchestra-coordination-foundation.md)
and remain green (`go test ./pkg/state/replication/...`). `VerifyConvergence`
(`pkg/state/replication/checkpoint.go`, added by task-5) is the general-purpose
primitive behind `synchestra state verify` for comparing two journals'
checksums at a shared cursor outside a test harness. This record's own
contribution is the missing **performance** half of the AC.

## Headline numbers

All benchmarks append/drain/restore **20 events** per iteration
(`benchEventCount` in `backend_bench_test.go`) against **real backends**: an
actual bare + clone Git repository (through inGitDB/DALgo, including durable
CAS push where the architecture calls for one) and a real on-disk SQLite
database (through `dalgo2sqlite`/DALgo) — no mocks. 3 iterations
(`-benchtime=3x`); reported `ns/op` is the mean per iteration.

### Append throughput (single writer, no batching)

| Backend | ns/op (20 events) | ~per event |
|---|---:|---:|
| SQLite active (unbatched) | 8,474,653 | 0.42 ms |
| Git active (unbatched, local commit only) | 1,446,630,236 | 72.3 ms |
| Git active (`GitPushJournal`: commit **+ durable push**) | 9,125,519,389 | 456.3 ms |

SQLite is ~170x faster than a bare local Git commit, and ~1,080x faster than
a Git commit that also durably proves itself pushed. This is exactly the
tradeoff `state-store/topology`'s Problem section names: "Git provides
reviewable history, portability, and recovery without a service ... SQLite
provides fast transactions ... on one server."

### Append throughput (group-commit batching, 20 concurrent callers)

| Backend | ns/op (one batch of 20) | Note |
|---|---:|---|
| SQLite active (batched, default 100 items / 1000 ms) | 1,008,465,152 | Item threshold (100) never reached at N=20; the batch waits out the full 1000 ms **time** window, not the item one. |
| Git active (batched, default 100 items / 1000 ms; no push) | 1,109,372,876 | Same time-window-dominated shape. |

**Finding:** at concurrency well below the 100-item default threshold,
group-commit batching's default *time* window (1000 ms) dominates and makes
a small concurrent burst measurably **slower** than the unbatched SQLite path
above (1.0 s vs. 8.5 ms for 20 events) — exactly the tradeoff
`state-store/journal-batching`'s README already documents in the abstract
("Append acknowledges only after its batch durably commits"). This is why
`pkg/state/gitstore`'s `Agent()` wiring pins both batching knobs to explicit
zero today (see that package's `agent.go`): nothing reaches it concurrently
yet, so there is no burst for batching to coalesce, only latency to pay. A
caller expecting bursts smaller than the item threshold should configure a
shorter `MaxBatchDelayMS` (state-store/journal-batching's two knobs are
independently tunable for exactly this reason).

### Replication lag drain (already-committed events → fresh mirror)

| Direction | ns/op (20 events) | ~per event |
|---|---:|---:|
| Git active → SQLite mirror (`DrainOutbox`) | 90,956,695 | 4.5 ms |
| SQLite active → Git mirror (`DrainOutbox`, durable push per event) | 9,202,546,194 | 460.1 ms |

Draining into SQLite is ~100x faster than draining into Git, because every
event delivered to a `*GitPushJournal`-backed mirror durably proves itself
pushed before `IngestReplica` returns (this is also exactly what makes
`Wait`'s mirror barrier able to trust a Git replica's local head as already
remote-proven in the common case — see `barrier.go`'s `RemoteReceipt` doc
comment).

### Recovery time (checkpoint + restore into a fresh endpoint)

| Direction | ns/op (20 events) | ~per event |
|---|---:|---:|
| Restore into a fresh SQLite mirror | 9,542,208 | 0.48 ms |
| Restore into a fresh, durably-pushed Git mirror | 11,020,590,750 | 551.0 ms |

**Known limitation, not fixed in this task:** `Restore` (`checkpoint.go`)
applies checkpoint events one at a time through the ordinary
`ReplicaIngestor` seam, so restoring into a `*GitPushJournal` target pays one
full commit-and-push round trip **per event**, not one push for the whole
restore. For a small checkpoint (20 events, ~11 s here) this is tolerable;
for a large one it would not be. A batch-aware Git restore (stage every
event locally, then push once) is real future work — recorded as an Open
Question below rather than solved here, since it would require a
batching-aware redesign of `GitPushJournal` explicitly called out as
out-of-scope in that type's own doc comment ("A future feature that wants
both durable per-endpoint CAS push AND batched local commits needs its own
batching-aware redesign of this type").

### Git growth

`git count-objects -v` on the bare origin, before vs. after 20 pushed
events (loose objects; no `git gc`/repack has run):

| | count | size (KiB) |
|---|---:|---:|
| Before | 7 | 28 |
| After 20 events | 147 | 588 |

~28 KiB of loose-object growth per event (~1.4 KiB average object; several
objects per commit — blob/tree/commit per inGitDB write). This is an upper
bound: Git packing (`git gc`) reclaims most of this via delta compression
against near-identical event payloads: not measured here, and worth a
follow-up once real production Git growth over a longer window is
observable.

### Conflicts / retries

Both directions in this record are single-writer, so the conflict and retry
counts are **0** for both. `state-store/topology`'s conflict/retry metrics
(`operation count, error count, conflict count, and retry count`) are
per-transaction counters already surfaced by health reporting infrastructure
(`ReplicaHealth`); a multi-writer contention benchmark that actually produces
non-zero conflicts is out of this task's scope (task-3's fenced claim work
owns concurrent-writer contention) and is recorded as an Open Question below.

## Machine context

| | |
|---|---|
| CPU | Apple M5 Max |
| OS | macOS 26.5.2 (Darwin 25.5.0, arm64) |
| Go | go1.26.4 darwin/arm64 |
| Commit | `6e68356822cceeb34a2bbb51fc95498aef2f44ad` (base of this record) |
| Recorded | 2026-08-13 |
| Command | `go test -run '^$' -bench . -benchtime=3x ./pkg/state/replication/...` |

Numbers here are single-host, unloaded-machine measurements, not a
production SLA. Re-record after any change to `DALJournal`, `GitPushJournal`,
`GitRemoteDurability`, or the batching engine that could plausibly move
these numbers.

## Reproducing

```sh
go test -run '^$' -bench . -benchtime=3x ./pkg/state/replication/...
```

Every benchmark builds its own fresh, isolated backend per iteration (a new
bare+clone Git repository, or a new on-disk SQLite file, under `t.TempDir()`)
and excludes that setup from the timed region via `b.StopTimer()`/
`b.StartTimer()` — see `backend_bench_test.go`'s package doc comment for the
full rationale, including why event counts are deliberately small (each Git
operation spawns several real `git` subprocesses, so keeping N at 20 bounds
a full `-bench .` run to a few minutes instead of many).

To reproduce a single direction, e.g. only the append-throughput
benchmarks:

```sh
go test -run '^$' -bench BenchmarkAppend -benchtime=3x ./pkg/state/replication/...
```

## Open Questions

1. A batch-aware `GitPushJournal`/`Restore` path that pushes once per
   restore instead of once per event — see "Recovery time" above. Not
   scheduled; revisit if a real restore workload's checkpoint size makes the
   current per-event cost impractical.
2. Git repository growth after realistic `git gc`/repack, rather than the
   loose-object-only number recorded here.
3. A multi-writer contention benchmark producing genuine conflict/retry
   counts (task-3's fenced-claim scope, not task-5's).

---
*This document follows repository research-record conventions (see
`spec/README.md`); it is not a SpecScore Feature/Idea/Plan artifact.*
