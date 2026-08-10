package replication

// Features implemented: state-store/topology, state-store/backends/git, state-store/backends/sqlite, agent-coordination
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
	role       Role
	epoch      int64
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
	return &DALJournal{db: db, projectID: options.ProjectID, endpointID: options.EndpointID, role: options.Role, epoch: options.AuthorityEpoch, replicaIDs: ids, message: options.CommitMessage}, nil
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

func (j *DALJournal) Append(ctx context.Context, event Event) error {
	if j.role != RoleActive || event.Cursor.Epoch != j.epoch {
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	return j.append(ctx, event)
}

func (j *DALJournal) IngestReplica(ctx context.Context, event Event) error {
	if j.role != RoleReplica {
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	return j.append(ctx, event)
}

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
		existing, found, err := loadEvent(ctx, tx, j.projectID, event.EventID)
		if err != nil {
			return err
		}
		if found {
			if existing.Checksum != event.Checksum {
				return fmt.Errorf("replication: event id %q reused with different checksum", event.EventID)
			}
			return nil
		}
		if err := validateNext(head, hash, event); err != nil {
			return err
		}
		if existingID, found, err := loadIdempotencyKey(ctx, tx, j.projectID, event.IdempotencyKey); err != nil {
			return err
		} else if found {
			return fmt.Errorf("replication: idempotency key %q already belongs to event %q", event.IdempotencyKey, existingID)
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("replication: encode event: %w", err)
		}
		if err := setJSON(ctx, tx, eventsCollection, eventKey(j.projectID, event.EventID), encoded); err != nil {
			return err
		}
		if err := setJSON(ctx, tx, messagesCollection, eventKey(j.projectID, event.EventID), encoded); err != nil {
			return err
		}
		for _, replicaID := range j.replicaIDs {
			if err := setJSON(ctx, tx, outboxCollection, outboxKey(j.projectID, replicaID, event.EventID), encoded); err != nil {
				return err
			}
		}
		return setJSON(ctx, tx, headCollection, headKey(j.projectID), encoded)
	}, options...)
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

func keyPart(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }

func eventKey(projectID, eventID string) string {
	return "project-" + keyPart(projectID) + "--event-" + keyPart(eventID)
}
func outboxKey(projectID, replicaID, eventID string) string {
	return "project-" + keyPart(projectID) + "--replica-" + keyPart(replicaID) + "--event-" + keyPart(eventID)
}
func headKey(projectID string) string { return "project-" + keyPart(projectID) + "--head" }

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
