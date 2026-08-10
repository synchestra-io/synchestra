package replication

// Features implemented: state-store/topology, state-store/topology/offline-fallback, agent-coordination

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrEpochFenced   = errors.New("replication: authority epoch is fenced")
	ErrSequenceGap   = errors.New("replication: sequence gap")
	ErrChecksumChain = errors.New("replication: checksum chain mismatch")
	ErrFallbackWrite = errors.New("replication: Git fallback accepts communication events only")
	ErrReplicaAhead  = errors.New("replication: replica is ahead of source")
	ErrDiverged      = errors.New("replication: source and replica diverged")
)

// Journal is the one persistence seam shared by Git/inGitDB and
// SQLite/DALgo. Append is idempotent by event ID and may only accept a
// contiguous event. The adapter owns durable commit/push evidence; the domain
// never writes a second backend synchronously.
type Journal interface {
	Append(context.Context, Event) error
	After(context.Context, Cursor) ([]Event, error)
	Head(context.Context) (Cursor, string, error)
}

// ReplicaHealth makes lag or a failed replication visible rather than silently
// presenting a mirror as fresh.
type ReplicaHealth struct {
	EndpointID string
	Cursor     Cursor
	EventLag   int64
	LastOK     time.Time
	LastError  string
}

// Replicate copies all contiguous events after the replica cursor. The source
// stays authoritative; partial delivery remains observable on the replica and
// can resume safely because Append is idempotent.
func Replicate(ctx context.Context, source, replica Journal, endpointID string) (ReplicaHealth, error) {
	head, sourceHash, err := source.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, LastError: err.Error()}, err
	}
	cursor, replicaHash, err := replica.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, LastError: err.Error()}, err
	}
	if compareCursor(cursor, head) > 0 {
		err := fmt.Errorf("%w: replica %v source %v", ErrReplicaAhead, cursor, head)
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, LastError: err.Error()}, err
	}
	if cursor == head && !cursor.IsZero() && replicaHash != sourceHash {
		err := fmt.Errorf("%w at cursor %v", ErrDiverged, cursor)
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, LastError: err.Error()}, err
	}
	events, err := source.After(ctx, cursor)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, LastError: err.Error()}, err
	}
	pending := int64(len(events))
	for _, event := range events {
		if err := replica.Append(ctx, event); err != nil {
			return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, EventLag: pending, LastError: err.Error()}, err
		}
		cursor = event.Cursor
		pending--
	}
	return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, EventLag: pending, LastOK: time.Now().UTC()}, nil
}

func compareCursor(a, b Cursor) int {
	if a.Epoch != b.Epoch {
		if a.Epoch < b.Epoch {
			return -1
		}
		return 1
	}
	if a.Sequence < b.Sequence {
		return -1
	}
	if a.Sequence > b.Sequence {
		return 1
	}
	return 0
}

// MemoryJournal is a strict conformance harness implementation. It is not a
// production backend: production Git and SQLite adapters must satisfy the
// Journal contract using inGitDB/DALgo respectively.
type MemoryJournal struct {
	mu       sync.Mutex
	events   []Event
	byID     map[string]Event
	byKey    map[string]string
	head     Cursor
	headHash string
}

func NewMemoryJournal() *MemoryJournal {
	return &MemoryJournal{byID: make(map[string]Event), byKey: make(map[string]string)}
}

func (j *MemoryJournal) Append(_ context.Context, event Event) error {
	if err := event.Verify(); err != nil {
		return err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if existing, exists := j.byID[event.EventID]; exists {
		if existing.Checksum != event.Checksum {
			return fmt.Errorf("replication: event id %q reused with different checksum", event.EventID)
		}
		return nil
	}
	if existingID, exists := j.byKey[event.IdempotencyKey]; exists {
		return fmt.Errorf("replication: idempotency key %q already belongs to event %q", event.IdempotencyKey, existingID)
	}
	if j.head.IsZero() {
		if event.Cursor.Epoch != 1 || event.Cursor.Sequence != 1 || event.PreviousHash != "" {
			return fmt.Errorf("%w: first event must be 1/1 with no previous hash", ErrSequenceGap)
		}
	} else {
		if event.Cursor.Epoch < j.head.Epoch {
			return ErrEpochFenced
		}
		if event.Cursor.Epoch == j.head.Epoch && event.Cursor.Sequence != j.head.Sequence+1 {
			return fmt.Errorf("%w: got %d, want %d", ErrSequenceGap, event.Cursor.Sequence, j.head.Sequence+1)
		}
		if event.Cursor.Epoch > j.head.Epoch && (event.Cursor.Epoch != j.head.Epoch+1 || event.Cursor.Sequence != 1) {
			return fmt.Errorf("%w: new epoch %d must start at sequence 1", ErrSequenceGap, event.Cursor.Epoch)
		}
		if event.PreviousHash != j.headHash {
			return ErrChecksumChain
		}
	}
	j.events = append(j.events, event)
	j.byID[event.EventID] = event
	j.byKey[event.IdempotencyKey] = event.EventID
	j.head, j.headHash = event.Cursor, event.Checksum
	return nil
}

func (j *MemoryJournal) After(_ context.Context, cursor Cursor) ([]Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	result := make([]Event, 0, len(j.events))
	for _, event := range j.events {
		if event.Cursor.Epoch > cursor.Epoch || (event.Cursor.Epoch == cursor.Epoch && event.Cursor.Sequence > cursor.Sequence) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (j *MemoryJournal) Head(_ context.Context) (Cursor, string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.head, j.headHash, nil
}
