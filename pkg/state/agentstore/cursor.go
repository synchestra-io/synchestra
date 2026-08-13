package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// cursorStore implements state.CursorStore: named-consumer read-position
// bookmarks (spec/features/state-store/topology's "Replica cursor"), kept
// independent of the journal's own authoritative head so, e.g., a dashboard
// or a lagging mirror's reported position never overwrites the journal's
// real tail.
type cursorStore struct{ store *Store }

var _ state.CursorStore = cursorStore{}

type cursorAdvancedPayload struct {
	Schema     string       `json:"schema"`
	ConsumerID string       `json:"consumer_id"`
	Cursor     state.Cursor `json:"cursor"`
	AdvancedAt time.Time    `json:"advanced_at"`
}

func (s *Store) loadCursors(ctx context.Context) (map[string]state.Cursor, error) {
	events, err := s.loadEvents(ctx, KindCursorAdvanced)
	if err != nil {
		return nil, err
	}
	cursors := make(map[string]state.Cursor, len(events))
	for _, event := range events {
		var payload cursorAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("agentstore: decode %s: %w", KindCursorAdvanced, err)
		}
		// Events replay in journal order, so the last write for a consumer
		// wins; Advance itself refuses a backward move, but replay does not
		// need to re-validate history it already accepted.
		cursors[payload.ConsumerID] = payload.Cursor
	}
	return cursors, nil
}

func (c cursorStore) Get(ctx context.Context, consumerID string) (state.Cursor, error) {
	cursors, err := c.store.loadCursors(ctx)
	if err != nil {
		return state.Cursor{}, err
	}
	return cursors[consumerID], nil
}

func (c cursorStore) Advance(ctx context.Context, consumerID string, cursor state.Cursor) error {
	if strings.TrimSpace(consumerID) == "" {
		return fmt.Errorf("agentstore: cursor consumer id is required")
	}
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		// head/events are read together (Store.snapshot) so the monotonicity
		// check below and the append that follows are pinned to the same
		// observation — a plain "read then append" would let two concurrent
		// Advance calls for the same consumer both pass the check against a
		// stale current value (see lease.go's Acquire for the general case).
		head, headHash, events, err := c.store.snapshot(ctx, KindCursorAdvanced)
		if err != nil {
			return err
		}
		current, err := foldCursor(events, consumerID)
		if err != nil {
			return err
		}
		if cursorLess(cursor, current) {
			return fmt.Errorf("agentstore: cursor %+v for consumer %q would move backward from %+v: %w", cursor, consumerID, current, state.ErrConflict)
		}
		now := c.store.options.Now()
		payload := cursorAdvancedPayload{Schema: schemaCursorAdvancedV1, ConsumerID: consumerID, Cursor: cursor, AdvancedAt: now}
		if _, err := c.store.tryAppendAt(ctx, head, headHash, KindCursorAdvanced, consumerID, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("agentstore: advance cursor for %q exhausted retries: %w", consumerID, lastErr)
}

func foldCursor(events []replication.Event, consumerID string) (state.Cursor, error) {
	var cursor state.Cursor
	for _, event := range events {
		var payload cursorAdvancedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return state.Cursor{}, fmt.Errorf("agentstore: decode %s: %w", KindCursorAdvanced, err)
		}
		if payload.ConsumerID == consumerID {
			cursor = payload.Cursor
		}
	}
	return cursor, nil
}

func cursorLess(a, b state.Cursor) bool {
	if a.Epoch != b.Epoch {
		return a.Epoch < b.Epoch
	}
	return a.Sequence < b.Sequence
}
