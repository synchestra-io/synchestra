package replication

// Features implemented: state-store/topology, state-store/topology/offline-fallback, agent-coordination

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Backend identifies a physical implementation without deciding authority.
type Backend string

const (
	BackendGit    Backend = "git"
	BackendSQLite Backend = "sqlite"
)

// Role controls whether an endpoint accepts domain writes.
type Role string

const (
	RoleActive  Role = "active"
	RoleReplica Role = "replica"
)

// ReplicaPurpose is intentionally independent of the backend type.
type ReplicaPurpose string

const (
	PurposeMirror ReplicaPurpose = "mirror"
	PurposeBackup ReplicaPurpose = "backup"
)

// Endpoint declares a participant in one project's storage topology.
type Endpoint struct {
	ID      string
	Backend Backend
	Role    Role
	Purpose ReplicaPurpose // required for replicas, empty for active
}

// Topology has exactly one active endpoint and at least one replica. Git is
// always present, but may be active or a replica.
type Topology struct {
	ProjectID string
	Endpoints []Endpoint
}

func (t Topology) Validate() error {
	if strings.TrimSpace(t.ProjectID) == "" {
		return fmt.Errorf("replication: project id is required")
	}
	active, replicas, git := 0, 0, false
	ids := make(map[string]struct{}, len(t.Endpoints))
	for _, endpoint := range t.Endpoints {
		if strings.TrimSpace(endpoint.ID) == "" {
			return fmt.Errorf("replication: endpoint id is required")
		}
		if _, exists := ids[endpoint.ID]; exists {
			return fmt.Errorf("replication: duplicate endpoint id %q", endpoint.ID)
		}
		ids[endpoint.ID] = struct{}{}
		if endpoint.Backend != BackendGit && endpoint.Backend != BackendSQLite {
			return fmt.Errorf("replication: endpoint %q has unsupported backend %q", endpoint.ID, endpoint.Backend)
		}
		git = git || endpoint.Backend == BackendGit
		switch endpoint.Role {
		case RoleActive:
			active++
			if endpoint.Purpose != "" {
				return fmt.Errorf("replication: active endpoint %q must not declare replica purpose", endpoint.ID)
			}
		case RoleReplica:
			replicas++
			if endpoint.Purpose != PurposeMirror && endpoint.Purpose != PurposeBackup {
				return fmt.Errorf("replication: replica endpoint %q needs mirror or backup purpose", endpoint.ID)
			}
		default:
			return fmt.Errorf("replication: endpoint %q has invalid role %q", endpoint.ID, endpoint.Role)
		}
	}
	if active != 1 {
		return fmt.Errorf("replication: topology needs exactly one active endpoint, got %d", active)
	}
	if replicas == 0 {
		return fmt.Errorf("replication: topology needs at least one replica")
	}
	if !git {
		return fmt.Errorf("replication: topology needs a Git endpoint")
	}
	return nil
}

// Cursor is an ordered position. Sequences restart only after a deliberate
// authority promotion increments Epoch.
type Cursor struct {
	Epoch    int64 `json:"authority_epoch"`
	Sequence int64 `json:"sequence"`
}

func (c Cursor) IsZero() bool { return c.Epoch == 0 && c.Sequence == 0 }

// Event is the immutable, deterministic envelope replicated between stores.
// Payload is an object encoded as canonical JSON by NewEvent; callers must not
// mutate it after append.
type Event struct {
	Schema         string          `json:"schema"`
	ProjectID      string          `json:"project_id"`
	EventID        string          `json:"event_id"`
	Cursor         Cursor          `json:"cursor"`
	OccurredAt     time.Time       `json:"occurred_at"`
	ActorID        string          `json:"actor_id"`
	CommandID      string          `json:"command_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Kind           string          `json:"kind"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	PreviousHash   string          `json:"previous_hash,omitempty"`
	Checksum       string          `json:"checksum"`
}

const EventSchemaV1 = "https://synchestra.io/schemas/state-change/v1"

// NewEvent constructs and signs an event's deterministic checksum. A caller
// supplies the allocated cursor and prior checksum; allocation belongs to the
// single active endpoint, never a replica.
func NewEvent(event Event) (Event, error) {
	if event.Schema == "" {
		event.Schema = EventSchemaV1
	}
	canonical, err := canonicalPayload(event.Payload)
	if err != nil {
		return Event{}, err
	}
	event.Payload = canonical
	if err := validateEventFields(event); err != nil {
		return Event{}, err
	}
	event.OccurredAt = event.OccurredAt.UTC()
	hash, err := eventHash(event)
	if err != nil {
		return Event{}, err
	}
	event.Checksum = hash
	return event, nil
}

func (e Event) Verify() error {
	if err := validateEventFields(e); err != nil {
		return err
	}
	canonical, err := canonicalPayload(e.Payload)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, e.Payload) {
		return fmt.Errorf("replication: event %q payload is not canonical JSON", e.EventID)
	}
	if e.Checksum == "" {
		return fmt.Errorf("replication: event %q checksum is required", e.EventID)
	}
	actual, err := eventHash(e)
	if err != nil {
		return err
	}
	if actual != e.Checksum {
		return fmt.Errorf("replication: event %q checksum mismatch", e.EventID)
	}
	return nil
}

func validateEventFields(event Event) error {
	if event.Schema != EventSchemaV1 || strings.TrimSpace(event.ProjectID) == "" ||
		strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.ActorID) == "" ||
		strings.TrimSpace(event.CommandID) == "" || strings.TrimSpace(event.IdempotencyKey) == "" ||
		strings.TrimSpace(event.Kind) == "" || event.Cursor.Epoch < 1 || event.Cursor.Sequence < 1 {
		return fmt.Errorf("replication: incomplete event")
	}
	if event.OccurredAt.IsZero() || event.OccurredAt.Location() != time.UTC {
		return fmt.Errorf("replication: event %q occurred_at must be UTC", event.EventID)
	}
	return nil
}

// canonicalPayload accepts a JSON object and preserves numeric lexemes via
// json.Number; decoding through float64 would silently corrupt large integer
// identifiers before they become part of the signed checksum.
func canonicalPayload(payload json.RawMessage) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("replication: decode payload: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("replication: payload must be an object")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("replication: canonicalize payload: %w", err)
	}
	return canonical, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("replication: payload has trailing JSON value")
		}
		return fmt.Errorf("replication: decode trailing payload: %w", err)
	}
	return nil
}

func eventHash(event Event) (string, error) {
	copy := event
	copy.Checksum = ""
	encoded, err := json.Marshal(copy)
	if err != nil {
		return "", fmt.Errorf("replication: encode event checksum: %w", err)
	}
	hash := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// SortEvents provides a stable ordering for implementations that load a
// journal from a filesystem or database query.
func SortEvents(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		if events[i].Cursor.Epoch != events[j].Cursor.Epoch {
			return events[i].Cursor.Epoch < events[j].Cursor.Epoch
		}
		return events[i].Cursor.Sequence < events[j].Cursor.Sequence
	})
}
