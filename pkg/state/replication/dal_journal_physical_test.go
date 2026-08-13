package replication

// Features implemented: state-store/topology, state-store/backends/git, state-store/backends/sqlite, agent-coordination, state-store/journal-batching

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo2sql"
	"github.com/dal-go/dalgo2sqlite"
	dalrecord "github.com/dal-go/record"
	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
)

// TestOutboxKeyPrefixDoesNotCollideAcrossReplicas proves finding #7 with the
// exact colliding pair the review construction produced: replica ID "abc"
// (keyPart "YWJj") and a second replica ID ridB whose base64url-encoded
// keyPart is literally "YWJj--event-" (i.e. ridB is
// base64url-decode("YWJj--event-")). With the old "--replica-"/"--event-"
// text separators, ridA's OWN outboxKeyPrefix ("...--replica-YWJj--event-")
// is a byte-for-byte PREFIX of every one of ridB's outboxKey rows
// ("...--replica-YWJj--event---event-<eventKeyPart>"), because '-' is part
// of base64.RawURLEncoding's own alphabet and ridB's encoded form happens to
// contain the "--event-" delimiter text mid-string. So scanning for ridA's
// rows (loadOutboxEvents(project, "abc")) would incorrectly also match every
// row actually queued for ridB. The current '.'-based keySeparator (outside
// that alphabet) makes the construction structurally impossible: this test
// reproduces the old collision against the pre-fix scheme inline and proves
// the current key functions no longer exhibit it.
func TestOutboxKeyPrefixDoesNotCollideAcrossReplicas(t *testing.T) {
	const ridA = "abc"
	decoded, err := base64.RawURLEncoding.DecodeString("YWJj--event-")
	if err != nil {
		t.Fatal(err)
	}
	ridB := string(decoded)
	if ridA == ridB {
		t.Fatal("test fixture invalid: the two replica IDs must differ")
	}

	// Reproduce the OLD (pre-fix) key scheme inline to prove this exact pair
	// really did collide under it -- otherwise this test would not be
	// demonstrating anything about the fix.
	oldKeyPart := func(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }
	oldPrefix := func(project, replica string) string {
		return "project-" + oldKeyPart(project) + "--replica-" + oldKeyPart(replica) + "--event-"
	}
	oldKey := func(project, replica, event string) string {
		return oldPrefix(project, replica) + oldKeyPart(event)
	}
	oldPrefixA := oldPrefix("proj", ridA)
	oldKeyB := oldKey("proj", ridB, "some-event")
	if !strings.HasPrefix(oldKeyB, oldPrefixA) {
		t.Fatalf("test fixture invalid: expected the OLD key scheme to collide on this pair (prefix=%q key=%q)", oldPrefixA, oldKeyB)
	}

	// The current key functions must not exhibit the same collision.
	prefixA := outboxKeyPrefix("proj", ridA)
	keyB := outboxKey("proj", ridB, "some-event")
	if strings.HasPrefix(keyB, prefixA) {
		t.Fatalf("replica %q's outbox prefix %q matches replica %q's event key %q — collision reproduced", ridA, prefixA, ridB, keyB)
	}
	if prefixA == outboxKeyPrefix("proj", ridB) {
		t.Fatalf("distinct replica IDs produced the same outbox prefix: %q", prefixA)
	}
}

func TestDALJournal_PhysicalGitActiveReplicatesToSQLiteAndRestarts(t *testing.T) {
	ctx := context.Background()
	gitRoot, gitJournal := newGitJournal(t, []string{"sqlite-mirror"})
	sqlitePath, sqliteJournal := newSQLiteJournalWithRole(t, []string{"git-mirror"}, RoleReplica, "sqlite-mirror")

	events := relayEvents(t)
	appendAll(t, gitJournal, events)
	if got := gitCommand(t, gitRoot, "rev-list", "--count", "HEAD"); got != "5" {
		t.Fatalf("Git active commit count = %s, want schema base plus one durable commit per event", got)
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
	_, reopened := newSQLiteJournalAt(t, sqlitePath, []string{"git-mirror"}, RoleReplica, "sqlite-mirror")
	assertPhysicalParity(t, gitJournal, reopened)
}

func TestDALJournal_PhysicalSQLiteActiveReplicatesToGitAndDoesNotDualWrite(t *testing.T) {
	ctx := context.Background()
	gitRoot, gitMirror := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	_, sqliteActive := newSQLiteJournal(t, []string{"git-mirror"})
	events := relayEvents(t)
	appendAll(t, sqliteActive, events)

	// A failed mirror delivery never rolls back or ambiguously changes the
	// SQLite active transaction. The durable outbox remains pending evidence;
	// drain/ack recovery is Planned and is not simulated by this test.
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
	if got := gitCommand(t, gitRoot, "rev-list", "--count", "HEAD"); got != "5" {
		t.Fatalf("Git mirror commit count = %s, want schema base plus 4", got)
	}
}

// TestDALJournal_OutboxDrainDeliversAndAcksAcrossBothPhysicalDirections
// exercises PendingOutbox/AckOutbox/DrainOutbox against real SQLite and Git
// DALgo adapters in both supported topologies, proving the durable outbox
// this test's sibling physical tests only assert gets written is also
// correctly drained and acknowledged on real backends, not just the in-memory
// harness.
func TestDALJournal_OutboxDrainDeliversAndAcksAcrossBothPhysicalDirections(t *testing.T) {
	ctx := context.Background()
	t.Run("git_active_to_sqlite_mirror", func(t *testing.T) {
		_, gitJournal := newGitJournal(t, []string{"sqlite-mirror"})
		_, sqliteJournal := newSQLiteJournalWithRole(t, nil, RoleReplica, "sqlite-mirror")
		events := relayEvents(t)
		appendAll(t, gitJournal, events)

		health, err := DrainOutbox(ctx, gitJournal, sqliteJournal, "sqlite-mirror")
		if err != nil {
			t.Fatalf("drain Git -> SQLite: %v", err)
		}
		if health.EventLag != 0 || health.Cursor != events[len(events)-1].Cursor {
			t.Fatalf("drain health = %+v", health)
		}
		assertPhysicalParity(t, gitJournal, sqliteJournal)
		pending, err := gitJournal.PendingOutbox(ctx, "sqlite-mirror")
		if err != nil || len(pending) != 0 {
			t.Fatalf("Git active outbox after drain = %#v, %v; want empty", pending, err)
		}
	})
	t.Run("sqlite_active_to_git_mirror", func(t *testing.T) {
		gitRoot, gitMirror := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
		_, sqliteActive := newSQLiteJournal(t, []string{"git-mirror"})
		events := relayEvents(t)
		appendAll(t, sqliteActive, events)

		health, err := DrainOutbox(ctx, sqliteActive, gitMirror, "git-mirror")
		if err != nil {
			t.Fatalf("drain SQLite -> Git: %v", err)
		}
		if health.EventLag != 0 {
			t.Fatalf("drain health = %+v, want no lag", health)
		}
		assertPhysicalParity(t, sqliteActive, gitMirror)
		if got := gitCommand(t, gitRoot, "rev-list", "--count", "HEAD"); got != "5" {
			t.Fatalf("Git mirror commit count = %s, want schema base plus 4 (one commit per drained event)", got)
		}
		pending, err := sqliteActive.PendingOutbox(ctx, "git-mirror")
		if err != nil || len(pending) != 0 {
			t.Fatalf("SQLite active outbox after drain = %#v, %v; want empty", pending, err)
		}

		// AckOutbox is idempotent on both physical adapters: acking an
		// already-drained row again must not error.
		if err := sqliteActive.AckOutbox(ctx, "git-mirror", events[0].EventID); err != nil {
			t.Fatalf("re-ack of an already-acked physical outbox row: %v", err)
		}
	})
}

func TestGitPushJournal_ReplicaIngestIsRemoteDurableAndAuthorityAppendIsFenced(t *testing.T) {
	ctx := context.Background()
	fixture := newGitDurabilityFixtureWithRole(t, RoleReplica, "git-mirror", []string{"git-mirror"})
	_, sqliteActive := newSQLiteJournal(t, []string{"git-mirror"})
	events := relayEvents(t)
	appendAll(t, sqliteActive, events)

	health, err := Replicate(ctx, sqliteActive, fixture.pushed, "git-mirror")
	if err != nil || health.EventLag != 0 || health.Cursor != events[len(events)-1].Cursor {
		t.Fatalf("SQLite active -> durable Git replica = %+v, %v", health, err)
	}
	remoteBeforeFence := gitCommand(t, fixture.root, "ls-remote", "origin", "refs/heads/main")
	err = fixture.pushed.Append(ctx, events[0])
	var fence *RoleFenceError
	if !errors.Is(err, ErrRoleFenced) || !errors.As(err, &fence) {
		t.Fatalf("Git replica public append error = %v, want role fence evidence", err)
	}
	if fence.EndpointID != "git-mirror" || fence.Role != RoleReplica || fence.AuthorityEpoch != 1 || fence.EventEpoch != 1 {
		t.Fatalf("Git wrapper role fence evidence = %+v", fence)
	}
	if remoteAfterFence := gitCommand(t, fixture.root, "ls-remote", "origin", "refs/heads/main"); remoteAfterFence != remoteBeforeFence {
		t.Fatalf("role-fenced append changed remote: before=%s after=%s", remoteBeforeFence, remoteAfterFence)
	}

	fresh := filepath.Join(t.TempDir(), "fresh-replica")
	if out, err := exec.Command("git", "clone", fixture.bare, fresh).CombinedOutput(); err != nil {
		t.Fatalf("fresh replica clone: %v: %s", err, out)
	}
	freshDB, err := dalgo2ingitdb.NewDatabase(fresh, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	freshJournal, err := NewDALJournal(freshDB, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "git-mirror", Role: RoleReplica, AuthorityEpoch: 1, ReplicaIDs: []string{"git-mirror"}, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	assertPhysicalParity(t, sqliteActive, freshJournal)
	for _, collection := range []string{eventsCollection, messagesCollection, outboxCollection, headCollection} {
		assertEncodedCollectionParity(t, sqliteActive.db, freshDB, collection)
	}
}

func TestDALFallbackInbox_GitStoresOnlyAllowedCommunicationEnvelopes(t *testing.T) {
	ctx := context.Background()
	_, gitJournal := newGitJournal(t, nil)
	inbox, err := NewDALFallbackInbox(gitJournal.db, "github.com/fair-split/relay", "synchestra fallback")
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := NewFallbackEnvelope(FallbackEnvelope{ProjectID: "github.com/fair-split/relay", EnvelopeID: "offline-message", OccurredAt: time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "send", IdempotencyKey: "offline-message", Kind: "message.sent", Payload: json.RawMessage(`{"body":"server is down"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := inbox.AppendFallback(ctx, allowed); err != nil {
		t.Fatal(err)
	}
	if got, err := inbox.FallbackEnvelopes(ctx); err != nil || len(got) != 1 || got[0].EnvelopeID != allowed.EnvelopeID {
		t.Fatalf("fallback inbox = %#v, %v", got, err)
	}
	claim, err := NewFallbackEnvelope(FallbackEnvelope{ProjectID: "github.com/fair-split/relay", EnvelopeID: "claim", OccurredAt: time.Date(2026, 8, 10, 13, 1, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "claim", IdempotencyKey: "claim", Kind: "claim.acquired", Payload: json.RawMessage(`{"claim_id":"worktree-1"}`)})
	if err == nil {
		t.Fatalf("claim fallback envelope unexpectedly accepted: %#v", claim)
	}
}

func TestDALJournal_SQLiteRollsBackDomainJournalAndOutboxTogether(t *testing.T) {
	ctx := context.Background()
	_, sqliteJournal := newSQLiteJournal(t, []string{"git-mirror"})
	failing := failAfterSet{DB: sqliteJournal.db, failOn: 2}
	journal, err := NewDALJournal(&failing, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "sqlite-active", Role: RoleActive, AuthorityEpoch: 1, ReplicaIDs: []string{"git-mirror"}, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
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

func TestDALJournal_RoleAndEpochFenceDomainWritesButAllowReplicaIngest(t *testing.T) {
	ctx := context.Background()
	_, mirror := newSQLiteJournalWithRole(t, nil, RoleReplica, "sqlite-mirror")
	event := relayEvents(t)[0]
	err := mirror.Append(ctx, event)
	var fence *RoleFenceError
	if !errors.Is(err, ErrRoleFenced) || !errors.As(err, &fence) {
		t.Fatalf("mirror direct append error = %v, want role fence evidence", err)
	}
	if fence.EndpointID != "sqlite-mirror" || fence.Role != RoleReplica || fence.AuthorityEpoch != 1 || fence.EventEpoch != 1 {
		t.Fatalf("role fence evidence = %+v", fence)
	}
	if err := mirror.IngestReplica(ctx, event); err != nil {
		t.Fatalf("explicit replica ingest: %v", err)
	}
	if head, _, err := mirror.Head(ctx); err != nil || head != event.Cursor {
		t.Fatalf("replica head after ingest = %+v, %v", head, err)
	}

	_, active := newSQLiteJournal(t, nil)
	if err := active.IngestReplica(ctx, event); !errors.Is(err, ErrRoleFenced) {
		t.Fatalf("active endpoint replica ingest error = %v, want role fence", err)
	}
	future := event
	future.Cursor.Epoch = 2
	future.Checksum = ""
	future, err = NewEvent(future)
	if err != nil {
		t.Fatal(err)
	}
	if err := active.Append(ctx, future); !errors.Is(err, ErrRoleFenced) {
		t.Fatalf("wrong authority epoch append error = %v, want role fence", err)
	}
}

func TestGitPushJournal_BareOriginReceiptAndFreshCloneParity(t *testing.T) {
	ctx := context.Background()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitCommand(t, t.TempDir(), "init", "--bare", "--initial-branch=main", bare)
	root := filepath.Join(t.TempDir(), "writer")
	if out, err := exec.Command("git", "clone", bare, root).CombinedOutput(); err != nil {
		t.Fatalf("clone writer: %v: %s", err, out)
	}
	gitCommand(t, root, "config", "user.email", "test@example.com")
	gitCommand(t, root, "config", "user.name", "Test")
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDALJournalSchema(ctx, database); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, root, "add", "-A")
	gitCommand(t, root, "commit", "-m", "journal schema")
	gitCommand(t, root, "push", "origin", "HEAD:refs/heads/main")
	local, err := NewDALJournal(database, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "git-active", Role: RoleActive, AuthorityEpoch: 1, CommitMessage: "synchestra state", MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	pushed, err := NewGitPushJournal(local, root, "origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	event := relayEvents(t)[0]
	receipt, err := pushed.AppendAndPush(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.AwaitingPush || receipt.CommitSHA == "" || receipt.RemoteRef != "refs/heads/main" {
		t.Fatalf("receipt = %+v, want remote durability evidence", receipt)
	}
	if got := gitCommand(t, root, "ls-remote", "origin", "refs/heads/main"); !strings.HasPrefix(got, receipt.CommitSHA) {
		t.Fatalf("remote ref = %s, want %s", got, receipt.CommitSHA)
	}
	inbox, err := NewDALFallbackInbox(database, "github.com/fair-split/relay", "synchestra fallback")
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := NewGitFallbackInboxPush(inbox, pushed.remote)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := NewFallbackEnvelope(FallbackEnvelope{ProjectID: "github.com/fair-split/relay", EnvelopeID: "offline-message", OccurredAt: time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "send", IdempotencyKey: "offline-message", Kind: "message.sent", Payload: json.RawMessage(`{"body":"server unavailable"}`)})
	if err != nil {
		t.Fatal(err)
	}
	var remoteInbox FallbackInbox = fallback
	if err := remoteInbox.AppendFallback(ctx, envelope); err != nil {
		t.Fatalf("interface-typed fallback remote append: %v", err)
	}
	// A transport failure after receipt finalization keeps the serialized event
	// and exact commit/base/ref durable. A new wrapper reconstructs and resumes
	// it without appending a second journal commit.
	missing := filepath.Join(t.TempDir(), "missing.git")
	pushed.remote.testFault = func(stage string) error {
		if stage == "receipt-finalized" {
			gitCommand(t, root, "remote", "set-url", "origin", missing)
		}
		return nil
	}
	pending, err := pushed.AppendAndPush(ctx, relayEvents(t)[1])
	if err == nil || !pending.AwaitingPush {
		t.Fatalf("failed transport receipt = %+v, %v", pending, err)
	}
	commitCount := gitCommand(t, root, "rev-list", "--count", "HEAD")
	gitCommand(t, root, "remote", "set-url", "origin", bare)
	restarted, err := NewGitPushJournal(local, root, "origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := restarted.ResumeOperation(ctx, pending.OperationID)
	if err != nil || resumed.AwaitingPush {
		t.Fatalf("resume = %+v, %v", resumed, err)
	}
	if got := gitCommand(t, root, "rev-list", "--count", "HEAD"); got != commitCount {
		t.Fatalf("resume appended a second commit: got %s, want %s", got, commitCount)
	}

	// A deletion after expected-base capture remains a CAS failure rather than
	// permission to recreate the ref. The same serialized operation resumes once
	// the expected base is restored.
	restarted.remote.testFault = func(stage string) error {
		if stage == "receipt-finalized" {
			gitCommand(t, bare, "update-ref", "-d", "refs/heads/main")
		}
		return nil
	}
	pending, err = restarted.AppendAndPush(ctx, relayEvents(t)[2])
	if err == nil || !pending.AwaitingPush {
		t.Fatalf("deleted-ref CAS receipt = %+v, %v", pending, err)
	}
	gitCommand(t, bare, "update-ref", "refs/heads/main", pending.ExpectedBase)
	restarted.remote.testFault = nil
	if _, err := restarted.ResumeOperation(ctx, pending.OperationID); err != nil {
		t.Fatalf("resume after restoring expected base: %v", err)
	}
	fresh := filepath.Join(t.TempDir(), "fresh")
	if out, err := exec.Command("git", "clone", bare, fresh).CombinedOutput(); err != nil {
		t.Fatalf("fresh clone: %v: %s", err, out)
	}
	freshDB, err := dalgo2ingitdb.NewDatabase(fresh, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	freshJournal, err := NewDALJournal(freshDB, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "git-mirror", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	assertPhysicalParity(t, local, freshJournal)
	freshInbox, err := NewDALFallbackInbox(freshDB, "github.com/fair-split/relay", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := freshInbox.FallbackEnvelopes(ctx); err != nil || len(got) != 1 || got[0].Checksum != envelope.Checksum {
		t.Fatalf("fresh clone fallback inbox = %#v, %v", got, err)
	}
}

type journalFailer struct {
	Journal
	err error
}

func (j journalFailer) Append(context.Context, Event) error        { return j.err }
func (j journalFailer) IngestReplica(context.Context, Event) error { return j.err }

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
	return newGitJournalWithRole(t, replicaIDs, RoleActive, "git-active")
}

func newGitJournalWithRole(t *testing.T, replicaIDs []string, role Role, endpointID string) (string, *DALJournal) {
	t.Helper()
	return newGitJournalWithOptions(t, DALJournalOptions{EndpointID: endpointID, Role: role, ReplicaIDs: replicaIDs, CommitMessage: "synchestra state", MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
}

// newGitJournalWithOptions is the batching-tests' entry point: opts is used
// almost verbatim (only ProjectID/AuthorityEpoch are defaulted when zero),
// so a caller can set MaxBatchItems/MaxBatchDelayMS explicitly to exercise
// group-commit batching against a real Git/inGitDB backend
// (state-store/journal-batching) rather than going through
// newGitJournalWithRole's fixed zeroBatch default.
func newGitJournalWithOptions(t *testing.T, opts DALJournalOptions) (string, *DALJournal) {
	t.Helper()
	root := t.TempDir()
	gitInit(t, root)
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDALJournalSchema(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, root, "add", "-A")
	gitCommand(t, root, "commit", "-m", "journal schema")
	if opts.ProjectID == "" {
		opts.ProjectID = "github.com/fair-split/relay"
	}
	if opts.EndpointID == "" {
		opts.EndpointID = "git-active"
	}
	if opts.Role == "" {
		opts.Role = RoleActive
	}
	if opts.AuthorityEpoch == 0 {
		opts.AuthorityEpoch = 1
	}
	if opts.CommitMessage == "" {
		// dalgo2ingitdb only creates a Git commit for a transaction carrying
		// dal.TxWithMessage (see DALJournal.append/commitBatch, which add it
		// only when j.message != ""); without this default every write here
		// would silently stage without ever committing.
		opts.CommitMessage = "synchestra state"
	}
	journal, err := NewDALJournal(database, opts)
	if err != nil {
		t.Fatal(err)
	}
	return root, journal
}

func newSQLiteJournal(t *testing.T, replicaIDs []string) (string, *DALJournal) {
	t.Helper()
	return newSQLiteJournalWithRole(t, replicaIDs, RoleActive, "sqlite-active")
}

func newSQLiteJournalWithRole(t *testing.T, replicaIDs []string, role Role, endpointID string) (string, *DALJournal) {
	t.Helper()
	return newSQLiteJournalAt(t, filepath.Join(t.TempDir(), "synchestra.db"), replicaIDs, role, endpointID)
}

func newSQLiteJournalAt(t *testing.T, path string, replicaIDs []string, role Role, endpointID string) (string, *DALJournal) {
	t.Helper()
	return newSQLiteJournalAtWithOptions(t, path, DALJournalOptions{EndpointID: endpointID, Role: role, ReplicaIDs: replicaIDs, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
}

// newSQLiteJournalWithOptions is the batching-tests' entry point over a real
// SQLite/DALgo backend: opts is used almost verbatim (only
// ProjectID/EndpointID/Role/AuthorityEpoch are defaulted when zero), so a
// caller can set MaxBatchItems/MaxBatchDelayMS explicitly
// (state-store/journal-batching).
func newSQLiteJournalWithOptions(t *testing.T, opts DALJournalOptions) (string, *DALJournal) {
	t.Helper()
	return newSQLiteJournalAtWithOptions(t, filepath.Join(t.TempDir(), "synchestra.db"), opts)
}

func newSQLiteJournalAtWithOptions(t *testing.T, path string, opts DALJournalOptions) (string, *DALJournal) {
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
	if opts.ProjectID == "" {
		opts.ProjectID = "github.com/fair-split/relay"
	}
	if opts.EndpointID == "" {
		opts.EndpointID = "sqlite-active"
	}
	if opts.Role == "" {
		opts.Role = RoleActive
	}
	if opts.AuthorityEpoch == 0 {
		opts.AuthorityEpoch = 1
	}
	journal, err := NewDALJournal(database, opts)
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

func assertEncodedCollectionParity(t *testing.T, source, replica queryExecutor, collection string) {
	t.Helper()
	ctx := context.Background()
	want, err := loadEncodedCollection(ctx, source, collection)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadEncodedCollection(ctx, replica, collection)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", want) {
		t.Fatalf("physical %s mismatch\n got: %#v\nwant: %#v", collection, got, want)
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
	return strings.TrimSpace(string(out))
}
