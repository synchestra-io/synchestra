package replication

// Features implemented: state-store/topology/offline-fallback
// Features depended on:  state-store/topology

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dal-go/dalgo/dal"
)

// FallbackEnvelope is an immutable offline communication record. Unlike Event,
// it has no authority cursor, previous hash, or journal checksum chain: the
// Git replica is not an authority and may not allocate any of those values.
type FallbackEnvelope struct {
	Schema         string          `json:"schema"`
	ProjectID      string          `json:"project_id"`
	EnvelopeID     string          `json:"envelope_id"`
	OccurredAt     time.Time       `json:"occurred_at"`
	ActorID        string          `json:"actor_id"`
	CommandID      string          `json:"command_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Kind           string          `json:"kind"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	Checksum       string          `json:"checksum"`
}

const FallbackEnvelopeSchemaV1 = "https://synchestra.io/schemas/fallback-envelope/v1"

var fallbackKinds = map[string]struct{}{
	"message.sent": {}, "message.acknowledged": {}, "worklog.checkpointed": {},
	"worklog.handoff_requested": {}, "repository.ref.updated": {},
	"agent.attention_requested": {}, "server.unreachable_observed": {},
}

func NewFallbackEnvelope(envelope FallbackEnvelope) (FallbackEnvelope, error) {
	if envelope.Schema == "" {
		envelope.Schema = FallbackEnvelopeSchemaV1
	}
	canonical, err := canonicalPayload(envelope.Payload)
	if err != nil {
		return FallbackEnvelope{}, err
	}
	envelope.Payload = canonical
	if err := envelope.validate(); err != nil {
		return FallbackEnvelope{}, err
	}
	envelope.OccurredAt = envelope.OccurredAt.UTC()
	envelope.Checksum = fallbackChecksum(envelope)
	return envelope, nil
}

func (e FallbackEnvelope) Verify() error {
	if err := e.validate(); err != nil {
		return err
	}
	canonical, err := canonicalPayload(e.Payload)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, e.Payload) || e.Checksum != fallbackChecksum(e) {
		return fmt.Errorf("replication: fallback envelope %q checksum or canonical payload mismatch", e.EnvelopeID)
	}
	return nil
}

func (e FallbackEnvelope) validate() error {
	if e.Schema != FallbackEnvelopeSchemaV1 || strings.TrimSpace(e.ProjectID) == "" || strings.TrimSpace(e.EnvelopeID) == "" ||
		strings.TrimSpace(e.ActorID) == "" || strings.TrimSpace(e.CommandID) == "" || strings.TrimSpace(e.IdempotencyKey) == "" ||
		e.OccurredAt.IsZero() || e.OccurredAt.Location() != time.UTC {
		return fmt.Errorf("replication: incomplete fallback envelope")
	}
	if _, ok := fallbackKinds[e.Kind]; !ok {
		return fmt.Errorf("%w: %q", ErrFallbackWrite, e.Kind)
	}
	return nil
}

func fallbackChecksum(envelope FallbackEnvelope) string {
	copy := envelope
	copy.Checksum = ""
	encoded, _ := json.Marshal(copy)
	hash := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// FallbackInbox persists a Git-only communication inbox. Reconciliation into
// the active authority is deliberately not implemented in this foundation;
// an importer must allocate fresh authoritative Event cursors exactly once.
type FallbackInbox interface {
	AppendFallback(context.Context, FallbackEnvelope) error
	FallbackEnvelopes(context.Context) ([]FallbackEnvelope, error)
}

type DALFallbackInbox struct {
	db        dal.DB
	projectID string
	message   string
}

// GitFallbackInboxPush wires the inbox through the same expected-base remote
// durability protocol as authority journal commits. It never turns inbox data
// into authority events; reconciliation remains planned.
type GitFallbackInboxPush struct {
	FallbackInbox
	remote *GitRemoteDurability
}

func NewGitFallbackInboxPush(inbox FallbackInbox, remote *GitRemoteDurability) (*GitFallbackInboxPush, error) {
	if inbox == nil || remote == nil {
		return nil, fmt.Errorf("replication: fallback inbox and Git remote are required")
	}
	return &GitFallbackInboxPush{FallbackInbox: inbox, remote: remote}, nil
}

func (i *GitFallbackInboxPush) AppendFallbackAndPush(ctx context.Context, envelope FallbackEnvelope) (GitCommitReceipt, error) {
	expected, err := i.remote.ExpectedBase(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if err := i.AppendFallback(ctx, envelope); err != nil {
		return GitCommitReceipt{}, err
	}
	pending, err := i.remote.RecordPending(ctx, expected)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	return i.remote.PushPending(ctx, pending)
}

func NewDALFallbackInbox(db dal.DB, projectID, commitMessage string) (*DALFallbackInbox, error) {
	if db == nil || strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("replication: fallback inbox database and project id are required")
	}
	return &DALFallbackInbox{db: db, projectID: projectID, message: commitMessage}, nil
}

func (i *DALFallbackInbox) AppendFallback(ctx context.Context, envelope FallbackEnvelope) error {
	if err := envelope.Verify(); err != nil {
		return err
	}
	if envelope.ProjectID != i.projectID {
		return fmt.Errorf("replication: fallback project %q does not match inbox project %q", envelope.ProjectID, i.projectID)
	}
	options := []dal.TransactionOption{}
	if i.message != "" {
		options = append(options, dal.TxWithMessage(fmt.Sprintf("%s: fallback %s", i.message, envelope.EnvelopeID)))
	}
	return i.db.RunReadwriteTransaction(ctx, func(ctx context.Context, tx dal.ReadwriteTransaction) error {
		current, err := loadFallbackEnvelopes(ctx, tx, i.projectID)
		if err != nil {
			return err
		}
		for _, existing := range current {
			if existing.EnvelopeID == envelope.EnvelopeID {
				if existing.Checksum != envelope.Checksum {
					return fmt.Errorf("replication: fallback envelope id %q reused", envelope.EnvelopeID)
				}
				return nil
			}
			if existing.IdempotencyKey == envelope.IdempotencyKey {
				return fmt.Errorf("replication: fallback idempotency key %q reused", envelope.IdempotencyKey)
			}
		}
		encoded, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		return setJSON(ctx, tx, fallbackCollection, eventKey(i.projectID, envelope.EnvelopeID), encoded)
	}, options...)
}

func (i *DALFallbackInbox) FallbackEnvelopes(ctx context.Context) ([]FallbackEnvelope, error) {
	return loadFallbackEnvelopes(ctx, i.db, i.projectID)
}

func loadFallbackEnvelopes(ctx context.Context, executor queryExecutor, projectID string) ([]FallbackEnvelope, error) {
	encoded, err := loadEncodedCollection(ctx, executor, fallbackCollection)
	if err != nil {
		return nil, err
	}
	result := make([]FallbackEnvelope, 0, len(encoded))
	for _, value := range encoded {
		var envelope FallbackEnvelope
		if err := json.Unmarshal([]byte(value), &envelope); err != nil {
			return nil, fmt.Errorf("replication: decode fallback envelope: %w", err)
		}
		if envelope.ProjectID == projectID {
			if err := envelope.Verify(); err != nil {
				return nil, err
			}
			result = append(result, envelope)
		}
	}
	sort.Slice(result, func(a, b int) bool {
		if result[a].OccurredAt.Equal(result[b].OccurredAt) {
			return result[a].EnvelopeID < result[b].EnvelopeID
		}
		return result[a].OccurredAt.Before(result[b].OccurredAt)
	})
	return result, nil
}
