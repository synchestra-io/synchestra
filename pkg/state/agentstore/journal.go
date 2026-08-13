package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"
	"fmt"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// journalStore implements state.JournalStore: read-only access to the
// underlying replication.Journal, translated to backend-neutral domain
// types. It grants no second Append path — every domain write above goes
// through Effort/Run/Worktree/Message/Activity/Lease, which alone know how
// to build a well-formed event.
type journalStore struct{ store *Store }

var _ state.JournalStore = journalStore{}

func (j journalStore) Head(ctx context.Context) (state.Cursor, string, error) {
	cursor, hash, err := j.store.journal.Head(ctx)
	if err != nil {
		return state.Cursor{}, "", fmt.Errorf("agentstore: read journal head: %w", err)
	}
	return toStateCursor(cursor), hash, nil
}

func (j journalStore) After(ctx context.Context, cursor state.Cursor) ([]state.JournalEntry, error) {
	events, err := j.store.journal.After(ctx, toReplicationCursor(cursor))
	if err != nil {
		return nil, fmt.Errorf("agentstore: read journal tail: %w", err)
	}
	entries := make([]state.JournalEntry, 0, len(events))
	for _, event := range events {
		if event.ProjectID != j.store.options.ProjectID {
			continue
		}
		entries = append(entries, state.JournalEntry{
			EventID: event.EventID, Kind: event.Kind, Cursor: toStateCursor(event.Cursor),
			CorrelationID: event.CorrelationID, OccurredAt: event.OccurredAt,
		})
	}
	return entries, nil
}
