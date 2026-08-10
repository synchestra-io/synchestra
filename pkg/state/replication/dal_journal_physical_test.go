package replication

// Features implemented: state-store/topology, state-store/backends/git, state-store/backends/sqlite, agent-coordination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo2sql"
	"github.com/dal-go/dalgo2sqlite"
	dalrecord "github.com/dal-go/record"
	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
)

func TestDALJournal_PhysicalGitActiveReplicatesToSQLiteAndRestarts(t *testing.T) {
	ctx := context.Background()
	gitRoot, gitJournal := newGitJournal(t, []string{"sqlite-mirror"})
	sqlitePath, sqliteJournal := newSQLiteJournal(t, []string{"git-mirror"})

	events := relayEvents(t)
	appendAll(t, gitJournal, events)
	if got := gitCommand(t, gitRoot, "rev-list", "--count", "HEAD"); got != "4" {
		t.Fatalf("Git active commit count = %s, want one durable commit per event", got)
	}
	if head, _, err := sqliteJournal.Head(ctx); err != nil || !head.IsZero() {
		t.Fatalf("SQLite mirror changed before replicate: head=%+v err=%v", head, err)
	}
	if outbox, err := loadCollection(ctx, gitJournal.db, outboxCollection); err != nil || len(outbox) != len(events) {
		t.Fatalf("Git active outbox = %d, %v; want %d entries", len(outbox), err, len(events))
	}

	health, err := Replicate(ctx, gitJournal, sqliteJournal, "sqlite-mirror")
	if err != nil {
		t.Fatalf("Git -> SQLite replicate: %v", err)
	}
	if health.EventLag != 0 || health.Cursor != (Cursor{Epoch: 1, Sequence: 4}) {
		t.Fatalf("Git -> SQLite health = %+v", health)
	}
	assertPhysicalParity(t, gitJournal, sqliteJournal)

	// Reopen the SQLite file through the DALgo adapter. The journal head and
	// ordered audit history must survive process restart, not just memory.
	if database, ok := sqliteJournal.db.(*dalgo2sqlite.Database); ok {
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
	}
	_, reopened := newSQLiteJournalAt(t, sqlitePath, []string{"git-mirror"})
	assertPhysicalParity(t, gitJournal, reopened)
}

func TestDALJournal_PhysicalSQLiteActiveReplicatesToGitAndDoesNotDualWrite(t *testing.T) {
	ctx := context.Background()
	gitRoot, gitMirror := newGitJournal(t, nil)
	_, sqliteActive := newSQLiteJournal(t, []string{"git-mirror"})
	events := relayEvents(t)
	appendAll(t, sqliteActive, events)

	// A failed mirror delivery never rolls back or ambiguously changes the
	// SQLite active transaction. Its durable outbox remains the recovery path.
	broken := journalFailer{Journal: gitMirror, err: errors.New("Git unavailable")}
	failureHealth, err := Replicate(ctx, sqliteActive, broken, "git-mirror")
	if err == nil {
		t.Fatal("replication to unavailable Git mirror unexpectedly succeeded")
	}
	if failureHealth.EventLag != 4 || failureHealth.LastError == "" {
		t.Fatalf("failed mirror health = %+v; want visible lag and error", failureHealth)
	}
	if head, _, err := sqliteActive.Head(ctx); err != nil || head != (Cursor{Epoch: 1, Sequence: 4}) {
		t.Fatalf("SQLite active lost committed head after mirror failure: %+v, %v", head, err)
	}
	if head, _, err := gitMirror.Head(ctx); err != nil || !head.IsZero() {
		t.Fatalf("failed Git mirror should stay empty: %+v, %v", head, err)
	}
	if outbox, err := loadCollection(ctx, sqliteActive.db, outboxCollection); err != nil || len(outbox) != len(events) {
		t.Fatalf("SQLite durable outbox = %d, %v; want %d", len(outbox), err, len(events))
	}

	health, err := Replicate(ctx, sqliteActive, gitMirror, "git-mirror")
	if err != nil {
		t.Fatalf("SQLite -> Git replicate: %v", err)
	}
	if health.EventLag != 0 {
		t.Fatalf("SQLite -> Git lag = %d, want 0", health.EventLag)
	}
	assertPhysicalParity(t, sqliteActive, gitMirror)
	if got := gitCommand(t, gitRoot, "rev-list", "--count", "HEAD"); got != "4" {
		t.Fatalf("Git mirror commit count = %s, want 4", got)
	}
}

func TestDALJournal_GitFallbackAppendsCommunicationThenReconciles(t *testing.T) {
	ctx := context.Background()
	_, gitFallback := newGitJournal(t, []string{"sqlite-mirror"})
	_, sqliteMirror := newSQLiteJournal(t, nil)
	events := relayEvents(t)
	for _, event := range events {
		if err := AppendGitFallback(ctx, gitFallback, event); err != nil {
			t.Fatalf("append fallback communication %s: %v", event.EventID, err)
		}
	}
	claimPayload := json.RawMessage(`{"claim_id":"worktree-1"}`)
	claim, err := NewEvent(Event{ProjectID: "github.com/fair-split/relay", EventID: "claim", Cursor: Cursor{Epoch: 1, Sequence: 5},
		OccurredAt: time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "claim", IdempotencyKey: "claim", Kind: "claim.acquired", CorrelationID: "fair-split-thread", Payload: claimPayload, PreviousHash: events[len(events)-1].Checksum})
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendGitFallback(ctx, gitFallback, claim); !errors.Is(err, ErrFallbackWrite) {
		t.Fatalf("fallback claim append error = %v, want %v", err, ErrFallbackWrite)
	}

	health, err := Replicate(ctx, gitFallback, sqliteMirror, "sqlite-mirror")
	if err != nil {
		t.Fatalf("reconcile Git fallback to SQLite: %v", err)
	}
	if health.EventLag != 0 {
		t.Fatalf("fallback reconcile lag = %d, want 0", health.EventLag)
	}
	assertPhysicalParity(t, gitFallback, sqliteMirror)
}

func TestDALJournal_SQLiteRollsBackDomainJournalAndOutboxTogether(t *testing.T) {
	ctx := context.Background()
	_, sqliteJournal := newSQLiteJournal(t, []string{"git-mirror"})
	failing := failAfterSet{DB: sqliteJournal.db, failOn: 2}
	journal, err := NewDALJournal(&failing, DALJournalOptions{ReplicaIDs: []string{"git-mirror"}})
	if err != nil {
		t.Fatal(err)
	}
	event := relayEvents(t)[0]
	if err := journal.Append(ctx, event); err == nil {
		t.Fatal("injected SQLite write failure unexpectedly succeeded")
	}
	for _, collection := range []string{eventsCollection, messagesCollection, outboxCollection, headCollection} {
		records, err := loadCollection(ctx, sqliteJournal.db, collection)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 0 {
			t.Fatalf("%s has %d records after rollback, want 0", collection, len(records))
		}
	}
}

type journalFailer struct {
	Journal
	err error
}

func (j journalFailer) Append(context.Context, Event) error { return j.err }

// failAfterSet injects failure after the domain message record is written but
// before journal/head/outbox finish. The real SQLite DALgo transaction must
// roll its first write back when the wrapper returns this error.
type failAfterSet struct {
	dal.DB
	failOn int
}

func (d *failAfterSet) RunReadwriteTransaction(ctx context.Context, worker dal.RWTxWorker, options ...dal.TransactionOption) error {
	return d.DB.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		return worker(ctx, &failAfterSetTx{ReadwriteTransaction: tx, failOn: d.failOn})
	}, options...)
}

type failAfterSetTx struct {
	dal.ReadwriteTransaction
	failOn int
	sets   int
}

func (tx *failAfterSetTx) Set(ctx context.Context, record dalrecord.Record) error {
	tx.sets++
	if tx.sets == tx.failOn {
		return errors.New("inject DAL write failure")
	}
	return tx.ReadwriteTransaction.Set(ctx, record)
}

func newGitJournal(t *testing.T, replicaIDs []string) (string, *DALJournal) {
	t.Helper()
	root := t.TempDir()
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	gitInit(t, root)
	journal, err := NewDALJournal(database, DALJournalOptions{ReplicaIDs: replicaIDs, CommitMessage: "synchestra state"})
	if err != nil {
		t.Fatal(err)
	}
	return root, journal
}

func newSQLiteJournal(t *testing.T, replicaIDs []string) (string, *DALJournal) {
	t.Helper()
	return newSQLiteJournalAt(t, filepath.Join(t.TempDir(), "synchestra.db"), replicaIDs)
}

func newSQLiteJournalAt(t *testing.T, path string, replicaIDs []string) (string, *DALJournal) {
	t.Helper()
	recordsets := make(map[string]*dalgo2sql.Recordset, 4)
	for _, collection := range []string{eventsCollection, headCollection, messagesCollection, outboxCollection} {
		recordsets[collection] = dalgo2sql.NewRecordset(collection, dalgo2sql.Table, []dal.FieldRef{dal.Field("id")})
	}
	database, err := dalgo2sqlite.NewDatabaseWithOptions(path, dal.NewSchema(nil, nil), dalgo2sql.DbOptions{Recordsets: recordsets})
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	journal, err := NewDALJournal(database, DALJournalOptions{ReplicaIDs: replicaIDs})
	if err != nil {
		t.Fatal(err)
	}
	return path, journal
}

func relayEvents(t *testing.T) []Event {
	t.Helper()
	inputs := []struct{ id, kind string }{
		{"request", "message.requested"},
		{"proposal", "message.proposed"},
		{"counterexample", "message.counterexample"},
		{"decision", "decision.accepted"},
	}
	previous := ""
	events := make([]Event, 0, len(inputs))
	for index, input := range inputs {
		payload, err := json.Marshal(map[string]any{"thread_id": "fair-split", "body": input.kind, "result": "EUR 3.34/3.33/3.33"})
		if err != nil {
			t.Fatal(err)
		}
		event, err := NewEvent(Event{ProjectID: "github.com/fair-split/relay", EventID: input.id,
			Cursor: Cursor{Epoch: 1, Sequence: int64(index + 1)}, OccurredAt: time.Date(2026, 8, 10, 12, index, 0, 0, time.UTC),
			ActorID: "agent:codex", CommandID: "fair-split", IdempotencyKey: input.id, Kind: input.kind,
			CorrelationID: "fair-split-thread", Payload: payload, PreviousHash: previous})
		if err != nil {
			t.Fatal(err)
		}
		previous = event.Checksum
		events = append(events, event)
	}
	return events
}

func appendAll(t *testing.T, journal Journal, events []Event) {
	t.Helper()
	for _, event := range events {
		if err := journal.Append(context.Background(), event); err != nil {
			t.Fatalf("append %s: %v", event.EventID, err)
		}
	}
}

func assertPhysicalParity(t *testing.T, source, replica Journal) {
	t.Helper()
	ctx := context.Background()
	want, err := source.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := replica.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", want) {
		t.Fatalf("physical journal mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}, {"config", "commit.gpgsign", "false"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func gitCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out[:len(out)-1])
}
