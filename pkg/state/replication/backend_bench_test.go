package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/topology, state-store/backends/git, state-store/backends/sqlite, state-store/journal-batching

// This file is the reproducible `go test -bench` seam behind
// state-store/topology#ac:backend-comparison-is-equivalent's evidence
// requirement ("the benchmark reports latency, throughput, ... and recovery
// time for both runs"). It exercises real backends -- inGitDB/DALgo over an
// actual bare+clone Git repository (including durable CAS push where the
// architecture calls for one) and SQLite/DALgo over a real on-disk database
// -- never mocks, so the numbers reflect actual subprocess/transaction cost.
// Results are not committed here: see spec/research/state-store-backend-comparison/
// for the recorded numbers, machine context, and how to reproduce them
// (`go test -bench . -benchtime=5x ./pkg/state/replication/...`).
//
// Event counts are deliberately small (benchEventCount): each Git operation
// spawns several real `git` subprocesses, so a benchmark whose per-iteration
// cost already exceeds Go's default ~1s benchtime naturally settles at b.N=1
// -- this keeps `-bench` runs bounded to tens of seconds rather than minutes
// while still measuring the real per-event cost, which is what the
// comparison record actually reports (cost per event / per batch, not a
// throughput number that depends on an arbitrarily large N).

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo2sql"
	"github.com/dal-go/dalgo2sqlite"
	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
)

const (
	benchProjectID  = "github.com/fair-split/relay"
	benchEventCount = 20
)

func benchGit(tb testing.TB, dir string, args ...string) {
	tb.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		tb.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// benchEvents builds n deterministic, chained events starting at cursor
// epoch 1 sequence 1 -- the same shape relayEvents(t) uses in the ordinary
// test suite, generalized to an arbitrary count for benchmark sizing.
func benchEvents(tb testing.TB, n int) []Event {
	tb.Helper()
	events := make([]Event, 0, n)
	previous := ""
	for i := 0; i < n; i++ {
		payload, err := json.Marshal(map[string]any{"seq": i, "body": "benchmark payload"})
		if err != nil {
			tb.Fatal(err)
		}
		event, err := NewEvent(Event{
			ProjectID: benchProjectID, EventID: fmt.Sprintf("bench-%d", i), Cursor: Cursor{Epoch: 1, Sequence: int64(i + 1)},
			OccurredAt: time.Now().UTC(), ActorID: "agent:bench", CommandID: "bench", IdempotencyKey: fmt.Sprintf("bench-%d", i),
			Kind: "bench.event", Payload: payload, PreviousHash: previous,
		})
		if err != nil {
			tb.Fatal(err)
		}
		previous = event.Checksum
		events = append(events, event)
	}
	return events
}

// benchGitJournal opens a Git/inGitDB-backed *DALJournal directly (no
// GitPushJournal push wrapper): this is the shape pkg/state/gitstore's
// Agent() wiring itself uses in production, and the one Git shape group-
// commit batching can actually compose with (*GitPushJournal structurally
// refuses to wrap a batching-enabled journal -- see git_push_journal.go).
func benchGitJournal(tb testing.TB, role Role, replicaIDs []string, maxItems, maxDelayMS *int) *DALJournal {
	tb.Helper()
	root := tb.TempDir()
	benchGit(tb, root, "init", "--initial-branch=main")
	benchGit(tb, root, "config", "user.email", "bench@example.com")
	benchGit(tb, root, "config", "user.name", "Bench")
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		tb.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		tb.Fatal(err)
	}
	benchGit(tb, root, "add", "-A")
	benchGit(tb, root, "commit", "-m", "journal schema")
	journal, err := NewDALJournal(database, DALJournalOptions{
		ProjectID: benchProjectID, EndpointID: "git-" + string(role), Role: role, AuthorityEpoch: 1, ReplicaIDs: replicaIDs,
		CommitMessage: "bench", MaxBatchItems: maxItems, MaxBatchDelayMS: maxDelayMS,
	})
	if err != nil {
		tb.Fatal(err)
	}
	return journal
}

// benchGitPushJournal builds a real bare-origin + clone pair and returns a
// *GitPushJournal over it -- the durable-CAS-push shape the architecture
// actually requires for a Git ACTIVE endpoint ("Git active, SQLite mirror":
// "commits/pushes the Git active repository") and for proving a Git MIRROR's
// remote durability. It never composes with batching (see NewGitPushJournal).
func benchGitPushJournal(tb testing.TB, role Role, endpointID string, replicaIDs []string) (bare, root string, pushed *GitPushJournal) {
	tb.Helper()
	bare = filepath.Join(tb.TempDir(), "origin.git")
	benchGit(tb, tb.TempDir(), "init", "--bare", "--initial-branch=main", bare)
	root = filepath.Join(tb.TempDir(), "writer")
	benchGit(tb, "", "clone", bare, root)
	benchGit(tb, root, "config", "user.email", "bench@example.com")
	benchGit(tb, root, "config", "user.name", "Bench")
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		tb.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		tb.Fatal(err)
	}
	benchGit(tb, root, "add", "-A")
	benchGit(tb, root, "commit", "-m", "journal schema")
	benchGit(tb, root, "push", "origin", "HEAD:refs/heads/main")
	zero := 0
	journal, err := NewDALJournal(database, DALJournalOptions{
		ProjectID: benchProjectID, EndpointID: endpointID, Role: role, AuthorityEpoch: 1, ReplicaIDs: replicaIDs,
		CommitMessage: "bench", MaxBatchItems: &zero, MaxBatchDelayMS: &zero,
	})
	if err != nil {
		tb.Fatal(err)
	}
	pushed, err = NewGitPushJournal(journal, root, "origin", "main")
	if err != nil {
		tb.Fatal(err)
	}
	return bare, root, pushed
}

// benchSQLiteJournal opens a real on-disk SQLite/DALgo *DALJournal, mirroring
// pkg/state/replication's own physical test fixture
// (newSQLiteJournalAtWithOptions in dal_journal_physical_test.go).
func benchSQLiteJournal(tb testing.TB, role Role, endpointID string, replicaIDs []string, maxItems, maxDelayMS *int) *DALJournal {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "synchestra.db")
	recordsets := make(map[string]*dalgo2sql.Recordset, 4)
	for _, collection := range []string{eventsCollection, headCollection, messagesCollection, outboxCollection} {
		recordsets[collection] = dalgo2sql.NewRecordset(collection, dalgo2sql.Table, []dal.FieldRef{dal.Field("id")})
	}
	database, err := dalgo2sqlite.NewDatabaseWithOptions(path, dal.NewSchema(nil, nil), dalgo2sql.DbOptions{Recordsets: recordsets})
	if err != nil {
		tb.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		tb.Fatal(err)
	}
	journal, err := NewDALJournal(database, DALJournalOptions{
		ProjectID: benchProjectID, EndpointID: endpointID, Role: role, AuthorityEpoch: 1, ReplicaIDs: replicaIDs,
		CommitMessage: "bench", MaxBatchItems: maxItems, MaxBatchDelayMS: maxDelayMS,
	})
	if err != nil {
		tb.Fatal(err)
	}
	return journal
}

var zeroBenchBatch = 0

// --- Append throughput: SQLite active, batched vs unbatched -----------------
//
// This is the primary group-commit comparison: the founder-MVP default
// topology (SQLite active, Git mirror) applies every domain write through
// exactly this path (state-store/journal-batching).

func BenchmarkAppendSQLiteActiveUnbatched(b *testing.B) {
	benchmarkAppend(b, func() Journal {
		return benchSQLiteJournal(b, RoleActive, "sqlite-active", nil, &zeroBenchBatch, &zeroBenchBatch)
	})
}

func BenchmarkAppendSQLiteActiveBatchedConcurrent(b *testing.B) {
	// nil/nil resolves to the documented default (100 items / 1000ms) --
	// concurrent Appends within that window are what group-commit batching
	// exists to coalesce into one transaction; see benchmarkAppendConcurrent.
	benchmarkAppendConcurrent(b, func() Journal { return benchSQLiteJournal(b, RoleActive, "sqlite-active", nil, nil, nil) })
}

// --- Append throughput: Git active (raw DALJournal, no push), batched vs
// unbatched -------------------------------------------------------------
//
// This is the shape pkg/state/gitstore's Agent() wiring uses today
// (batching pinned off there for unrelated reasons -- see that package's
// comment); the batched variant measures what group-commit batching would
// buy Git if a future caller reached the journal directly and concurrently.

func BenchmarkAppendGitActiveUnbatched(b *testing.B) {
	benchmarkAppend(b, func() Journal { return benchGitJournal(b, RoleActive, nil, &zeroBenchBatch, &zeroBenchBatch) })
}

func BenchmarkAppendGitActiveBatchedConcurrent(b *testing.B) {
	benchmarkAppendConcurrent(b, func() Journal { return benchGitJournal(b, RoleActive, nil, nil, nil) })
}

// --- Append + durable push: Git active via GitPushJournal ------------------
//
// GitPushJournal cannot compose with batching (NewGitPushJournal refuses --
// see git_push_journal.go), so this is the one number for "Git active" the
// architecture's own "commits/pushes the Git active repository" sentence
// actually describes: per-event commit-and-push latency, not a batched
// throughput figure.

func BenchmarkAppendGitActivePushed(b *testing.B) {
	events := benchEvents(b, benchEventCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, _, pushed := benchGitPushJournal(b, RoleActive, "git-active", nil)
		b.StartTimer()
		for _, event := range events {
			if _, err := pushed.AppendAndPush(context.Background(), event); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportMetric(float64(len(events)), "events/op")
}

// benchmarkAppend times benchEventCount sequential, unbatched Append calls
// against a freshly built journal per b.N iteration (setup excluded via
// StopTimer/StartTimer).
func benchmarkAppend(b *testing.B, build func() Journal) {
	b.Helper()
	events := benchEvents(b, benchEventCount)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		journal := build()
		b.StartTimer()
		for _, event := range events {
			if err := journal.Append(ctx, event); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportMetric(float64(len(events)), "events/op")
}

// benchmarkAppendConcurrent fires benchEventCount Appends from separate
// goroutines nearly simultaneously against one freshly built journal per
// b.N iteration -- the shape group-commit batching is actually designed
// for (see pkg/state/replication/README.md's "Batching" section): only
// concurrent callers give the batcher anything to coalesce into one
// transaction.
func benchmarkAppendConcurrent(b *testing.B, build func() Journal) {
	b.Helper()
	events := benchEvents(b, benchEventCount)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		journal := build()
		b.StartTimer()
		done := make(chan error, len(events))
		for _, event := range events {
			go func(e Event) { done <- journal.Append(ctx, e) }(event)
		}
		for range events {
			if err := <-done; err != nil {
				b.Fatal(err)
			}
		}
		if closer, ok := journal.(BatchedJournal); ok {
			b.StopTimer()
			if err := closer.Close(ctx); err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
		}
	}
	b.ReportMetric(float64(len(events)), "events/op")
}

// --- Replication lag drain: both directions ---------------------------------

// BenchmarkReplicationLagDrainGitActiveToSQLiteMirror times DrainOutbox
// delivering benchEventCount already-committed events from a Git active
// endpoint into a fresh SQLite mirror -- "Git active, SQLite mirror".
func BenchmarkReplicationLagDrainGitActiveToSQLiteMirror(b *testing.B) {
	benchmarkDrain(b,
		func() OutboxSource {
			return benchGitJournal(b, RoleActive, []string{"sqlite-mirror"}, &zeroBenchBatch, &zeroBenchBatch)
		},
		func() Journal {
			return benchSQLiteJournal(b, RoleReplica, "sqlite-mirror", nil, &zeroBenchBatch, &zeroBenchBatch)
		},
		"sqlite-mirror",
	)
}

// BenchmarkReplicationLagDrainSQLiteActiveToGitMirror times DrainOutbox
// delivering benchEventCount already-committed events from a SQLite active
// endpoint into a fresh, REQUIRED Git mirror through *GitPushJournal (so the
// number reflects real per-event durable-push cost, not merely a local
// apply) -- "SQLite active, Git mirror", the founder-MVP default.
func BenchmarkReplicationLagDrainSQLiteActiveToGitMirror(b *testing.B) {
	benchmarkDrain(b,
		func() OutboxSource {
			return benchSQLiteJournal(b, RoleActive, "sqlite-active", []string{"git-mirror"}, &zeroBenchBatch, &zeroBenchBatch)
		},
		func() Journal { _, _, pushed := benchGitPushJournal(b, RoleReplica, "git-mirror", nil); return pushed },
		"git-mirror",
	)
}

func benchmarkDrain(b *testing.B, buildSource func() OutboxSource, buildReplica func() Journal, replicaID string) {
	b.Helper()
	events := benchEvents(b, benchEventCount)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		source := buildSource()
		for _, event := range events {
			if err := source.Append(ctx, event); err != nil {
				b.Fatal(err)
			}
		}
		replica := buildReplica()
		b.StartTimer()
		if _, err := DrainOutbox(ctx, source, replica, replicaID); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(events)), "events/op")
}

// --- Restore time: both directions ------------------------------------------

// BenchmarkRestoreIntoFreshSQLiteMirror checkpoints a Git-active source at
// benchEventCount events (untimed) and times Restore into a fresh SQLite
// endpoint -- the "Git active" direction's recovery-time figure.
func BenchmarkRestoreIntoFreshSQLiteMirror(b *testing.B) {
	benchmarkRestore(b,
		func() Journal { return benchGitJournal(b, RoleActive, nil, &zeroBenchBatch, &zeroBenchBatch) },
		func() Journal {
			return benchSQLiteJournal(b, RoleReplica, "restored", nil, &zeroBenchBatch, &zeroBenchBatch)
		},
	)
}

// BenchmarkRestoreIntoFreshGitMirror checkpoints a SQLite-active source at
// benchEventCount events (untimed) and times Restore into a fresh, durably
// pushed Git endpoint -- the "SQLite active" direction's recovery-time
// figure, including the real per-event push cost Restore's ReplicaIngestor
// seam pays on a *GitPushJournal target.
func BenchmarkRestoreIntoFreshGitMirror(b *testing.B) {
	benchmarkRestore(b,
		func() Journal {
			return benchSQLiteJournal(b, RoleActive, "sqlite-active", nil, &zeroBenchBatch, &zeroBenchBatch)
		},
		func() Journal { _, _, pushed := benchGitPushJournal(b, RoleReplica, "restored", nil); return pushed },
	)
}

func benchmarkRestore(b *testing.B, buildSource func() Journal, buildTarget func() Journal) {
	b.Helper()
	events := benchEvents(b, benchEventCount)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		source := buildSource()
		for _, event := range events {
			if err := source.Append(ctx, event); err != nil {
				b.Fatal(err)
			}
		}
		checkpoint, err := NewCheckpoint(ctx, source, "source", Cursor{})
		if err != nil {
			b.Fatal(err)
		}
		target := buildTarget()
		b.StartTimer()
		if _, err := Restore(ctx, checkpoint, target, "restored"); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(events)), "events/op")
}
