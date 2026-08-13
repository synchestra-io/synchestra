package replication

// Features implemented: state-store/topology, state-store/backends/git, state-store/backends/sqlite, agent-coordination, state-store/journal-batching
// Features depended on:  state-store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo/dbschema"
	"github.com/dal-go/dalgo/ddl"
	"github.com/dal-go/record"
)

const (
	eventsCollection   = "synchestra_journal_events"
	headCollection     = "synchestra_journal_heads"
	messagesCollection = "synchestra_coordination_messages"
	outboxCollection   = "synchestra_replication_outbox"
	fallbackCollection = "synchestra_git_fallback_inbox"
)

// DALJournal stores typed coordination messages, their immutable journal
// records, and one outbox item per replica through a single DALgo read-write
// transaction. Git uses the DALgo-to-inGitDB adapter; SQLite uses
// dalgo2sqlite. No Synchestra domain handler writes a second store directly.
type DALJournal struct {
	db         dal.DB
	replicaIDs []string
	message    string
	projectID  string
	endpointID string

	// mu guards role/epoch together. Promote's PromoteToActive/FenceAsReplica
	// hold it for the whole checkpoint-then-transition sequence so a
	// concurrent Append/IngestReplica never observes the new epoch paired
	// with the old role or vice versa.
	mu    sync.RWMutex
	role  Role
	epoch int64

	batchSettings BatchSettings
	batcher       *appendBatcher
}

// DALJournalOptions configures the durable outbox. CommitMessage is passed to
// DALgo's transaction options; the Git adapter turns it into the one commit
// that contains the domain message, journal record, head, and outbox records.
// SQLite simply treats it as transaction metadata.
type DALJournalOptions struct {
	// ProjectID scopes every physical key and is required: one server database
	// may host many projects, but no journal instance may cross their boundary.
	ProjectID      string
	EndpointID     string
	Role           Role
	AuthorityEpoch int64
	ReplicaIDs     []string
	CommitMessage  string
	// MaxBatchItems and MaxBatchDelayMS configure group-commit batching on
	// the append path (state-store/journal-batching). Nil means "use the
	// documented default" (100 items / 1000ms); an explicit pointer to 0
	// disables that dimension; both explicitly 0 restores immediate
	// per-event commits, byte-identical to the pre-batching journal. See
	// BatchSettings/resolveBatchSettings and this package's README
	// "Batching" section.
	MaxBatchItems   *int
	MaxBatchDelayMS *int
}

func NewDALJournal(db dal.DB, options DALJournalOptions) (*DALJournal, error) {
	if db == nil {
		return nil, fmt.Errorf("replication: DAL database is required")
	}
	if strings.TrimSpace(options.ProjectID) == "" {
		return nil, fmt.Errorf("replication: journal project id is required")
	}
	if strings.TrimSpace(options.EndpointID) == "" {
		return nil, fmt.Errorf("replication: journal endpoint id is required")
	}
	if options.Role != RoleActive && options.Role != RoleReplica {
		return nil, fmt.Errorf("replication: journal endpoint %q has invalid role %q", options.EndpointID, options.Role)
	}
	if options.AuthorityEpoch < 1 {
		return nil, fmt.Errorf("replication: journal authority epoch must be positive")
	}
	ids := append([]string(nil), options.ReplicaIDs...)
	sort.Strings(ids)
	for i, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("replication: replica id is required")
		}
		if i > 0 && ids[i-1] == id {
			return nil, fmt.Errorf("replication: duplicate replica id %q", id)
		}
	}
	j := &DALJournal{
		db: db, projectID: options.ProjectID, endpointID: options.EndpointID, role: options.Role, epoch: options.AuthorityEpoch,
		replicaIDs: ids, message: options.CommitMessage,
		batchSettings: resolveBatchSettings(options.MaxBatchItems, options.MaxBatchDelayMS),
	}
	if !j.batchSettings.disabled() {
		j.batcher = newAppendBatcher(j.batchSettings, j.commitBatch)
	}
	return j, nil
}

// EnsureDALJournalSchema provisions exactly the collections required by this
// vertical slice through DALgo DDL. It is deliberately usable with either
// physical adapter; no SQL or filesystem schema is owned by Synchestra.
func EnsureDALJournalSchema(ctx context.Context, db dal.DB) error {
	for _, name := range []string{eventsCollection, headCollection, messagesCollection, outboxCollection, fallbackCollection} {
		collection := dbschema.CollectionDef{
			Name: name,
			Fields: []dbschema.FieldDef{
				{Name: "id", Type: dbschema.String, Nullable: false},
				{Name: "event_json", Type: dbschema.String, Nullable: false},
			},
			PrimaryKey: []dal.FieldName{"id"},
		}
		if err := ddl.CreateCollection(ctx, db, collection, ddl.IfNotExists()); err != nil {
			return fmt.Errorf("replication: create %s: %w", name, err)
		}
	}
	return nil
}

// roleEpoch returns a consistent snapshot of the endpoint's local role and
// authority epoch. See the DALJournal.mu doc comment for why Append and
// IngestReplica must read both fields together.
func (j *DALJournal) roleEpoch() (Role, int64) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.role, j.epoch
}

// EndpointID and RoleEpoch expose the fields PromotableJournal needs; they
// are otherwise unexported so nothing outside promotion.go's fencing seam can
// read or race the role/epoch pair casually.
func (j *DALJournal) EndpointID() string       { return j.endpointID }
func (j *DALJournal) RoleEpoch() (Role, int64) { return j.roleEpoch() }

// BatchSettings returns this journal's effective group-commit configuration.
func (j *DALJournal) BatchSettings() BatchSettings { return j.batchSettings }

// Close flushes any pending batch (state-store/journal-batching#ac:close-flushes-pending-batch);
// see BatchedJournal's doc comment for the full contract.
func (j *DALJournal) Close(ctx context.Context) error {
	if j.batcher == nil {
		return nil
	}
	return j.batcher.Close(ctx)
}

// HeadEvent returns the full latest durable event, not just its cursor and
// hash. Promote's resumable-fence detection (promotion.go's ensureFenced)
// inspects HeadEvent's Kind/IdempotencyKey/Payload to recognize a checkpoint
// a previous, partial Promote call already left durably in place, rather
// than re-fencing or refusing outright.
func (j *DALJournal) HeadEvent(ctx context.Context) (Event, bool, error) {
	head, _, err := j.Head(ctx)
	if err != nil {
		return Event{}, false, err
	}
	if head.IsZero() {
		return Event{}, false, nil
	}
	events, err := j.After(ctx, Cursor{Epoch: head.Epoch, Sequence: head.Sequence - 1})
	if err != nil {
		return Event{}, false, err
	}
	if len(events) == 0 {
		return Event{}, false, fmt.Errorf("replication: head cursor %v has no matching event", head)
	}
	return events[len(events)-1], true, nil
}

// Append and IngestReplica hold j.mu for the role/epoch precondition check
// AND the step that makes the event durably observable to a concurrent
// fence, not just the check. With batching disabled that step is IngestReplica's
// (always) or Append's (when j.batcher is nil) entire underlying transaction,
// exactly as before batching existed. With batching enabled, Append instead
// holds j.mu only for the brief enqueue-into-pending step -- see batch.go's
// appendBatcher.enqueue and its "Promotion interplay (flush-then-fence)" doc
// comment for why that is still enough to close the same race: once
// FenceAsReplica (or PromoteToActive) requests the exclusive lock, Go's
// sync.RWMutex blocks any new Append/IngestReplica from starting until the
// fence/promotion completes, and any Append that already holds the read lock
// finishes enqueueing (or, unbatched, its whole transaction) before the fence
// can proceed, and FenceAsReplica flushes that pending batch itself before
// ever building its checkpoint. Every write therefore resolves to exactly one
// of "completed (or durably queued to complete) before the role/epoch
// transition" or "observed the new role/epoch and failed here" -- never a raw
// checksum-chain/sequence-gap error that hides a role transition as its real
// cause.
func (j *DALJournal) Append(ctx context.Context, event Event) error {
	j.mu.RLock()
	if j.role != RoleActive || event.Cursor.Epoch != j.epoch {
		j.mu.RUnlock()
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	if j.batcher == nil {
		defer j.mu.RUnlock()
		return j.append(ctx, event)
	}
	done, flushNow, err := j.batcher.enqueue(event)
	j.mu.RUnlock()
	if err != nil {
		return err
	}
	if flushNow {
		j.batcher.flush(ctx)
	}
	return j.batcher.wait(ctx, done)
}

func (j *DALJournal) IngestReplica(ctx context.Context, event Event) error {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if j.role != RoleReplica {
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	return j.append(ctx, event)
}

// append is the unbatched, single-event transaction: used directly by
// IngestReplica (replica ingest is never batched -- see this package's
// README "Batching" section) and by Append when batching is disabled. It
// shares its per-event validate-and-write logic with commitBatch via
// commitOneLocked so both paths behave identically -- see AC
// zero-disables-each-dimension's byte-identical requirement.
func (j *DALJournal) append(ctx context.Context, event Event) error {
	if err := event.Verify(); err != nil {
		return err
	}
	if event.ProjectID != j.projectID {
		return fmt.Errorf("replication: event project %q does not match journal project %q", event.ProjectID, j.projectID)
	}
	options := []dal.TransactionOption{}
	if j.message != "" {
		options = append(options, dal.TxWithMessage(fmt.Sprintf("%s: %s", j.message, event.EventID)))
	}
	return j.db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		head, hash, err := loadHead(ctx, tx, j.projectID)
		if err != nil {
			return err
		}
		encoded, err := j.commitOneLocked(ctx, tx, &head, &hash, event)
		if err != nil {
			return err
		}
		if encoded == nil {
			// Idempotent no-op: nothing new was written, so there is no new
			// head to record -- matches the pre-batching behavior exactly.
			return nil
		}
		return setJSON(ctx, tx, headCollection, headKey(j.projectID), encoded)
	}, options...)
}

// infraCommitError marks a commitOneLocked failure as an infrastructure
// failure -- a storage read/write itself returning an error -- rather than a
// per-event validation failure (content, sequence/epoch fencing, checksum
// chain, or idempotency collision). commitBatch's loop uses this
// distinction to decide whether to skip just this event (a validation
// failure: the rest of the batch, sharing the running head, still commits --
// AC batched-events-preserve-fencing-and-order) or abort the WHOLE
// transaction (an infrastructure failure: the transaction's outcome for
// every other event in this batch is now unknown, so nothing in it may be
// treated as durable -- AC append-acknowledges-only-durable-commits). append
// (the unbatched, single-event path) does not need this distinction --
// either kind of error already aborts its one-event transaction the same
// way -- so it is unwrapped there via Unwrap(), preserving every existing
// errors.Is/errors.As check and the failure's original message.
type infraCommitError struct{ err error }

func (e *infraCommitError) Error() string { return e.err.Error() }
func (e *infraCommitError) Unwrap() error { return e.err }

// commitOneLocked validates and (if valid) writes event's rows using tx,
// treating (*head, *hash) as the running cursor/checksum state validateNext
// checks against -- advanced in place on success, left untouched on failure.
// It is the one place that decides "does this event commit," shared by
// append (single-event, used by IngestReplica always and by Append when
// batching is disabled) and commitBatch (group-commit, multi-event): both
// paths validate and write an event identically. Returns (nil, nil) for an
// idempotent replay (event already durable with a matching checksum) --
// nothing new is written and the caller must not advance any head record for
// it.
func (j *DALJournal) commitOneLocked(ctx context.Context, tx dal.ReadwriteTransaction, head *Cursor, hash *string, event Event) ([]byte, error) {
	if err := event.Verify(); err != nil {
		return nil, err
	}
	if event.ProjectID != j.projectID {
		return nil, fmt.Errorf("replication: event project %q does not match journal project %q", event.ProjectID, j.projectID)
	}
	existing, found, err := loadEvent(ctx, tx, j.projectID, event.EventID)
	if err != nil {
		return nil, &infraCommitError{err}
	}
	if found {
		if existing.Checksum != event.Checksum {
			return nil, fmt.Errorf("replication: event id %q reused with different checksum", event.EventID)
		}
		return nil, nil
	}
	if err := validateNext(*head, *hash, event); err != nil {
		return nil, err
	}
	if existingID, found, err := loadIdempotencyKey(ctx, tx, j.projectID, event.IdempotencyKey); err != nil {
		return nil, &infraCommitError{err}
	} else if found {
		return nil, fmt.Errorf("replication: idempotency key %q already belongs to event %q", event.IdempotencyKey, existingID)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return nil, &infraCommitError{fmt.Errorf("replication: encode event: %w", err)}
	}
	if err := setJSON(ctx, tx, eventsCollection, eventKey(j.projectID, event.EventID), encoded); err != nil {
		return nil, &infraCommitError{err}
	}
	if err := setJSON(ctx, tx, messagesCollection, eventKey(j.projectID, event.EventID), encoded); err != nil {
		return nil, &infraCommitError{err}
	}
	for _, replicaID := range j.replicaIDs {
		if err := setJSON(ctx, tx, outboxCollection, outboxKey(j.projectID, replicaID, event.EventID), encoded); err != nil {
			return nil, &infraCommitError{err}
		}
	}
	*head, *hash = event.Cursor, event.Checksum
	return encoded, nil
}

// commitBatch is DALJournal's batchCommitFunc (batch.go): it durably commits
// every event in events, in order, inside ONE storage transaction -- events,
// their per-replica outbox rows, and a single head advance -- exactly as REQ
// group-commit-not-buffering and AC batch-flushes-at-item-threshold require
// ("one Git commit on the inGitDB backend"). An individual event that fails
// validation (content, sequence, checksum chain, or idempotency-key
// collision -- see commitOneLocked) does not abort the transaction: its slot
// in the returned []error carries that failure and the running head/hash
// used for the remaining events' validation is left exactly as it was, so
// the rest of the batch, sharing that same running head, still commits in
// cursor order (AC batched-events-preserve-fencing-and-order). Only a
// genuine infrastructure failure (commitOneLocked's *infraCommitError,
// meaning the transaction itself errored -- e.g. a storage write failure)
// aborts the WHOLE transaction and fails every event in the batch with that
// same error -- the second return value -- since that is the only case
// where nothing in the batch is durable (AC append-acknowledges-only-durable-commits).
func (j *DALJournal) commitBatch(ctx context.Context, events []Event) ([]error, error) {
	results := make([]error, len(events))
	options := []dal.TransactionOption{}
	if j.message != "" {
		options = append(options, dal.TxWithMessage(fmt.Sprintf("%s: batch of %d event(s)", j.message, len(events))))
	}
	txErr := j.db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		head, hash, err := loadHead(ctx, tx, j.projectID)
		if err != nil {
			return err
		}
		var lastEncoded []byte
		for i, event := range events {
			encoded, commitErr := j.commitOneLocked(ctx, tx, &head, &hash, event)
			if commitErr != nil {
				var infra *infraCommitError
				if errors.As(commitErr, &infra) {
					// The transaction is now in an unknown state: abort it
					// entirely rather than silently committing whatever
					// happened to write before this failure.
					return commitErr
				}
				results[i] = commitErr
				continue
			}
			if encoded != nil {
				lastEncoded = encoded
			}
		}
		if lastEncoded == nil {
			return nil
		}
		return setJSON(ctx, tx, headCollection, headKey(j.projectID), lastEncoded)
	}, options...)
	if txErr != nil {
		for i := range results {
			results[i] = txErr
		}
		return results, txErr
	}
	return results, nil
}

func (j *DALJournal) After(ctx context.Context, cursor Cursor) ([]Event, error) {
	events, err := loadCollection(ctx, j.db, eventsCollection)
	if err != nil {
		return nil, err
	}
	result := make([]Event, 0, len(events))
	for _, event := range events {
		if event.ProjectID == j.projectID && (event.Cursor.Epoch > cursor.Epoch || (event.Cursor.Epoch == cursor.Epoch && event.Cursor.Sequence > cursor.Sequence)) {
			result = append(result, event)
		}
	}
	SortEvents(result)
	return result, nil
}

func (j *DALJournal) Head(ctx context.Context) (Cursor, string, error) {
	return loadHead(ctx, j.db, j.projectID)
}

// PendingOutbox returns replicaID's durable outbox rows for this endpoint's
// project, ordered by cursor. Every row was written in the same transaction
// as the domain message, journal event, and head (see append above); nothing
// removes a row except AckOutbox, so this is always the exact undelivered
// tail regardless of which endpoint or process last drained it.
func (j *DALJournal) PendingOutbox(ctx context.Context, replicaID string) ([]Event, error) {
	if strings.TrimSpace(replicaID) == "" {
		return nil, fmt.Errorf("replication: outbox replica id is required")
	}
	return loadOutboxEvents(ctx, j.db, j.projectID, replicaID)
}

// AckOutbox deletes replicaID's durable outbox rows for eventIDs, all in one
// transaction (DrainOutbox batches every ack from one drain into a single
// call instead of one transaction per event). Delete is a documented no-op
// on both DAL adapters this package targets — dalgo2sql issues
// "DELETE ... WHERE id = ?" (zero rows affected is not an error) and
// dalgo2ingitdb explicitly treats a missing record as already-deleted — so
// acknowledging a row that is already gone (a second drain worker raced
// ahead, or a prior crash landed after this exact delete had already
// committed) is always safe and never surfaces an error to the caller. A
// zero-length eventIDs call is a no-op that does not open a transaction.
func (j *DALJournal) AckOutbox(ctx context.Context, replicaID string, eventIDs ...string) error {
	if strings.TrimSpace(replicaID) == "" {
		return fmt.Errorf("replication: outbox ack requires a replica id")
	}
	if len(eventIDs) == 0 {
		return nil
	}
	for _, eventID := range eventIDs {
		if strings.TrimSpace(eventID) == "" {
			return fmt.Errorf("replication: outbox ack requires a non-blank event id")
		}
	}
	options := []dal.TransactionOption{}
	if j.message != "" {
		options = append(options, dal.TxWithMessage(fmt.Sprintf("%s: ack outbox %s/%d event(s)", j.message, replicaID, len(eventIDs))))
	}
	return j.db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		for _, eventID := range eventIDs {
			key := record.NewKeyWithID(outboxCollection, outboxKey(j.projectID, replicaID, eventID))
			if err := tx.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	}, options...)
}

func validateNext(head Cursor, headHash string, event Event) error {
	if head.IsZero() {
		if event.Cursor.Epoch != 1 || event.Cursor.Sequence != 1 || event.PreviousHash != "" {
			return fmt.Errorf("%w: first event must be 1/1 with no previous hash", ErrSequenceGap)
		}
		return nil
	}
	if event.Cursor.Epoch < head.Epoch {
		return ErrEpochFenced
	}
	if event.Cursor.Epoch == head.Epoch && event.Cursor.Sequence != head.Sequence+1 {
		return fmt.Errorf("%w: got %d, want %d", ErrSequenceGap, event.Cursor.Sequence, head.Sequence+1)
	}
	if event.Cursor.Epoch > head.Epoch && (event.Cursor.Epoch != head.Epoch+1 || event.Cursor.Sequence != 1) {
		return fmt.Errorf("%w: new epoch %d must start at sequence 1", ErrSequenceGap, event.Cursor.Epoch)
	}
	if event.PreviousHash != headHash {
		return ErrChecksumChain
	}
	return nil
}

func setJSON(ctx context.Context, tx dal.ReadwriteTransaction, collection, id string, value []byte) error {
	record := record.NewRecordWithData(record.NewKeyWithID(collection, id), map[string]any{"event_json": string(value)})
	if err := tx.Set(ctx, record); err != nil {
		return fmt.Errorf("replication: persist %s/%s: %w", collection, id, err)
	}
	return nil
}

type queryExecutor interface {
	ExecuteQueryToRecordsReader(context.Context, dal.Query) (dal.RecordsReader, error)
}

func loadHead(ctx context.Context, executor queryExecutor, projectID string) (Cursor, string, error) {
	events, err := loadCollection(ctx, executor, headCollection)
	if err != nil {
		return Cursor{}, "", err
	}
	var matching []Event
	for _, event := range events {
		if event.ProjectID == projectID {
			matching = append(matching, event)
		}
	}
	if len(matching) == 0 {
		return Cursor{}, "", nil
	}
	if len(matching) != 1 {
		return Cursor{}, "", fmt.Errorf("replication: expected one journal head for project %q, got %d", projectID, len(matching))
	}
	return matching[0].Cursor, matching[0].Checksum, nil
}

func loadEvent(ctx context.Context, executor queryExecutor, projectID, id string) (Event, bool, error) {
	events, err := loadCollection(ctx, executor, eventsCollection)
	if err != nil {
		return Event{}, false, err
	}
	for _, event := range events {
		if event.ProjectID == projectID && event.EventID == id {
			return event, true, nil
		}
	}
	return Event{}, false, nil
}

func loadIdempotencyKey(ctx context.Context, executor queryExecutor, projectID, key string) (string, bool, error) {
	events, err := loadCollection(ctx, executor, eventsCollection)
	if err != nil {
		return "", false, err
	}
	for _, event := range events {
		if event.ProjectID == projectID && event.IdempotencyKey == key {
			return event.EventID, true, nil
		}
	}
	return "", false, nil
}

// keyPart base64url-encodes an arbitrary ID segment for use inside a
// composite record key. base64.RawURLEncoding's alphabet includes '-' and
// '_', so any delimiter used to join segments MUST lie outside that alphabet
// -- see keySeparator.
func keyPart(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }

// keySeparator joins key segments. It is deliberately '.', which never
// appears in base64.RawURLEncoding output (alphabet: A-Z a-z 0-9 - _). Using
// a separator drawn from the SAME alphabet as keyPart's output (the original
// "--replica-"/"--event-" text separators both included '-', which the
// alphabet also produces) let a crafted ID collide with a delimiter and make
// one replica's outbox PREFIX a literal prefix of ANOTHER replica's rows --
// e.g. replica ID "abc" (keyPart "YWJj") vs. a replica ID X such that
// keyPart(X) == "YWJj--event-" (X = base64url-decode("YWJj--event-")): with
// the OLD text separators, outboxKeyPrefix(project, "abc") (which ends in
// "...replica-YWJj--event-") was byte-for-byte a prefix of every
// outboxKey(project, X, eventID) (which starts
// "...replica-YWJj--event---event-..."), so scanning for "abc"'s rows would
// also match every row actually queued for X. Because keySeparator can never
// occur inside a keyPart output, that construction is now structurally
// impossible: splitting any generated key on keySeparator boundaries always
// recovers the exact same segments it was built from. See
// TestOutboxKeyPrefixDoesNotCollideAcrossReplicas for the reproduction.
const keySeparator = "."

func eventKey(projectID, eventID string) string {
	return "project" + keySeparator + keyPart(projectID) + keySeparator + "event" + keySeparator + keyPart(eventID)
}

// outboxKeyPrefix identifies every outbox row queued for one replica,
// regardless of event. loadOutboxEvents filters the outbox scan against it;
// outboxKey appends the specific event segment used by Set/Delete.
func outboxKeyPrefix(projectID, replicaID string) string {
	return "project" + keySeparator + keyPart(projectID) + keySeparator + "replica" + keySeparator + keyPart(replicaID) + keySeparator + "event" + keySeparator
}
func outboxKey(projectID, replicaID, eventID string) string {
	return outboxKeyPrefix(projectID, replicaID) + keyPart(eventID)
}
func headKey(projectID string) string {
	return "project" + keySeparator + keyPart(projectID) + keySeparator + "head"
}

func loadCollection(ctx context.Context, executor queryExecutor, collection string) ([]Event, error) {
	encoded, err := loadEncodedCollection(ctx, executor, collection)
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(encoded))
	for _, value := range encoded {
		var event Event
		if err := json.Unmarshal([]byte(value), &event); err != nil {
			return nil, fmt.Errorf("replication: decode %s: %w", collection, err)
		}
		if err := event.Verify(); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func loadEncodedCollection(ctx context.Context, executor queryExecutor, collection string) ([]string, error) {
	ref := dal.NewRootCollectionRef(collection, "")
	query := dal.NewQueryBuilder(dal.From(ref)).SelectIntoRecordset()
	reader, err := executor.ExecuteQueryToRecordsReader(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("replication: query %s: %w", collection, err)
	}
	defer func() { _ = reader.Close() }()
	var encodedValues []string
	for {
		record, readErr := reader.Next()
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, dal.ErrNoMoreRecords) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("replication: read %s: %w", collection, readErr)
		}
		data, ok := record.Data().(map[string]any)
		if !ok {
			return nil, fmt.Errorf("replication: %s record %v is not object data", collection, record.Key().ID)
		}
		encoded, ok := data["event_json"].(string)
		if !ok {
			return nil, fmt.Errorf("replication: %s record %v has no event_json", collection, record.Key().ID)
		}
		encodedValues = append(encodedValues, encoded)
	}
	return encodedValues, nil
}

// loadOutboxEvents scans the outbox collection for one replica's durable
// delivery rows. It cannot reuse loadEncodedCollection/loadCollection as-is:
// the stored event JSON carries no replica ID (the same event fans out to
// many replicas' keys), so filtering needs each record's key, not just its
// data.
func loadOutboxEvents(ctx context.Context, executor queryExecutor, projectID, replicaID string) ([]Event, error) {
	ref := dal.NewRootCollectionRef(outboxCollection, "")
	query := dal.NewQueryBuilder(dal.From(ref)).SelectIntoRecordset()
	reader, err := executor.ExecuteQueryToRecordsReader(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("replication: query %s: %w", outboxCollection, err)
	}
	defer func() { _ = reader.Close() }()
	prefix := outboxKeyPrefix(projectID, replicaID)
	var events []Event
	for {
		rec, readErr := reader.Next()
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, dal.ErrNoMoreRecords) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("replication: read %s: %w", outboxCollection, readErr)
		}
		data, ok := rec.Data().(map[string]any)
		if !ok {
			return nil, fmt.Errorf("replication: %s record %v is not object data", outboxCollection, rec.Key().ID)
		}
		// Workaround for dal-go/dalgo2sql query reader not populating
		// Key().ID — remove when fixed upstream:
		// https://github.com/dal-go/dalgo2sql/issues/168. dalgo2ingitdb's
		// query reader returns the record ID on Key().ID; dalgo2sql's
		// returns "" there and puts the selected id column into
		// Data()["id"] instead. Recognize whichever the adapter populated
		// rather than assuming one shape.
		id, ok := rec.Key().ID.(string)
		if !ok || id == "" {
			id, ok = data["id"].(string)
		}
		if !ok || !strings.HasPrefix(id, prefix) {
			continue
		}
		encoded, ok := data["event_json"].(string)
		if !ok {
			return nil, fmt.Errorf("replication: %s record %v has no event_json", outboxCollection, rec.Key().ID)
		}
		var event Event
		if err := json.Unmarshal([]byte(encoded), &event); err != nil {
			return nil, fmt.Errorf("replication: decode %s: %w", outboxCollection, err)
		}
		if err := event.Verify(); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	SortEvents(events)
	return events, nil
}
