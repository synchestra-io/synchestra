package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology/offline-fallback

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// messageStore implements state.MessageStore. It deliberately reuses the
// exact "message.sent"/"message.acknowledged" Kind strings the Git fallback
// allowlist already recognizes (replication.fallbackKinds in
// pkg/state/replication/fallback_inbox.go), so transport-switch reconciliation
// (task-3 scope, agent-coordination#ac:messages-survive-transport-switch)
// has a single shared vocabulary to reconcile against rather than a second
// one invented here. Typed negotiation (Message.Kind/Evidence — request,
// proposal, counterexample, decision.accepted, per agent-coordination/
// cross-harness-conformance) rides inside that same envelope's payload
// rather than minting new journal/fallback Kind strings.
type messageStore struct{ store *Store }

var _ state.MessageStore = messageStore{}

// evidenceRefPayload is the wire shape of state.EvidenceRef.
type evidenceRefPayload struct {
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
	Reference   string `json:"reference,omitempty"`
}

func toEvidenceRefPayloads(refs []state.EvidenceRef) []evidenceRefPayload {
	if len(refs) == 0 {
		return nil
	}
	out := make([]evidenceRefPayload, len(refs))
	for i, ref := range refs {
		out[i] = evidenceRefPayload{Kind: ref.Kind, Description: ref.Description, Reference: ref.Reference}
	}
	return out
}

func fromEvidenceRefPayloads(payloads []evidenceRefPayload) []state.EvidenceRef {
	if len(payloads) == 0 {
		return nil
	}
	out := make([]state.EvidenceRef, len(payloads))
	for i, payload := range payloads {
		out[i] = state.EvidenceRef{Kind: payload.Kind, Description: payload.Description, Reference: payload.Reference}
	}
	return out
}

type messageSentPayload struct {
	Schema          string               `json:"schema"`
	MessageID       string               `json:"message_id"`
	EffortID        string               `json:"effort_id,omitempty"`
	ThreadID        string               `json:"thread_id"`
	SenderRunID     string               `json:"sender_run_id"`
	RecipientRunIDs []string             `json:"recipient_run_ids,omitempty"`
	CorrelationID   string               `json:"correlation_id,omitempty"`
	RepositoryID    string               `json:"repository_id,omitempty"`
	ClaimID         string               `json:"claim_id,omitempty"`
	Body            string               `json:"body,omitempty"`
	Kind            state.MessageKind    `json:"kind,omitempty"`
	Evidence        []evidenceRefPayload `json:"evidence,omitempty"`
	SentAt          time.Time            `json:"sent_at"`
}

type messageAckPayload struct {
	Schema         string    `json:"schema"`
	MessageID      string    `json:"message_id"`
	RecipientRunID string    `json:"recipient_run_id"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

func (s *Store) loadMessages(ctx context.Context) (map[string]state.Message, error) {
	events, err := s.loadEvents(ctx, KindMessageSent)
	if err != nil {
		return nil, err
	}
	messages := make(map[string]state.Message, len(events))
	for _, event := range events {
		var payload messageSentPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("agentstore: decode %s: %w", KindMessageSent, err)
		}
		messages[payload.MessageID] = state.Message{
			ID: payload.MessageID, EffortID: payload.EffortID, ThreadID: payload.ThreadID, SenderRunID: payload.SenderRunID,
			RecipientRunIDs: payload.RecipientRunIDs, CorrelationID: payload.CorrelationID, RepositoryID: payload.RepositoryID,
			ClaimID: payload.ClaimID, Body: payload.Body, SentAt: payload.SentAt,
			Kind: payload.Kind, Evidence: fromEvidenceRefPayloads(payload.Evidence),
		}
	}
	return messages, nil
}

// validateMessageKind enforces that a MessageKindDecisionAccepted message —
// the typed negotiation sequence's terminal record — cites at least one
// EvidenceRef, per agent-coordination/cross-harness-conformance: "The
// accepted decision references the counterexample/test evidence". Every
// other Kind (including the zero value, general-purpose default) is
// unconstrained.
func validateMessageKind(kind state.MessageKind, evidence []state.EvidenceRef) error {
	switch kind {
	case "", state.MessageKindRequest, state.MessageKindProposal, state.MessageKindCounterexample, state.MessageKindNote:
		return nil
	case state.MessageKindDecisionAccepted:
		if len(evidence) == 0 {
			return fmt.Errorf("agentstore: a %s message needs at least one evidence reference", state.MessageKindDecisionAccepted)
		}
		return nil
	default:
		return fmt.Errorf("agentstore: unknown message kind %q", kind)
	}
}

func (m messageStore) Send(ctx context.Context, params state.MessageSendParams) (state.Message, error) {
	if strings.TrimSpace(params.ThreadID) == "" || strings.TrimSpace(params.SenderRunID) == "" {
		return state.Message{}, fmt.Errorf("agentstore: message needs a thread id and sender run id")
	}
	if err := validateMessageKind(params.Kind, params.Evidence); err != nil {
		return state.Message{}, err
	}
	now := m.store.options.Now()
	id := m.store.options.NewID()
	recipients := append([]string(nil), params.RecipientRunIDs...)
	evidence := append([]state.EvidenceRef(nil), params.Evidence...)
	msg := state.Message{
		ID: id, EffortID: params.EffortID, ThreadID: params.ThreadID, SenderRunID: params.SenderRunID,
		RecipientRunIDs: recipients, CorrelationID: params.CorrelationID, RepositoryID: params.RepositoryID,
		ClaimID: params.ClaimID, Body: params.Body, SentAt: now, Kind: params.Kind, Evidence: evidence,
	}
	payload := messageSentPayload{
		Schema: schemaMessageSentV2, MessageID: id, EffortID: params.EffortID, ThreadID: params.ThreadID,
		SenderRunID: params.SenderRunID, RecipientRunIDs: recipients, CorrelationID: params.CorrelationID,
		RepositoryID: params.RepositoryID, ClaimID: params.ClaimID, Body: params.Body, SentAt: now,
		Kind: params.Kind, Evidence: toEvidenceRefPayloads(evidence),
	}
	correlation := params.CorrelationID
	if correlation == "" {
		correlation = params.ThreadID
	}
	if _, err := m.store.appendWithRetry(ctx, KindMessageSent, correlation, payload); err != nil {
		return state.Message{}, err
	}
	return msg, nil
}

func (m messageStore) Acknowledge(ctx context.Context, messageID, recipientRunID string) (state.MessageAck, error) {
	messages, err := m.store.loadMessages(ctx)
	if err != nil {
		return state.MessageAck{}, err
	}
	msg, ok := messages[messageID]
	if !ok {
		return state.MessageAck{}, fmt.Errorf("agentstore: message %q: %w", messageID, state.ErrNotFound)
	}
	now := m.store.options.Now()
	ack := state.MessageAck{MessageID: messageID, RecipientRunID: recipientRunID, AcknowledgedAt: now}
	payload := messageAckPayload{Schema: schemaMessageAcknowledgedV1, MessageID: messageID, RecipientRunID: recipientRunID, AcknowledgedAt: now}
	correlation := msg.CorrelationID
	if correlation == "" {
		correlation = msg.ThreadID
	}
	if _, err := m.store.appendWithRetry(ctx, KindMessageAcknowledged, correlation, payload); err != nil {
		return state.MessageAck{}, err
	}
	return ack, nil
}

func (m messageStore) Thread(ctx context.Context, threadID string) ([]state.Message, error) {
	messages, err := m.store.loadMessages(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]state.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.ThreadID == threadID {
			result = append(result, msg)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SentAt.Before(result[j].SentAt) })
	return result, nil
}
