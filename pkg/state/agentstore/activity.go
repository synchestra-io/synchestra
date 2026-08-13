package agentstore

// Features implemented: state-store, agent-coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// activityStore implements state.ActivityStore, the append-only feed backing
// project/repository agent views (agent-coordination "Active/recent project
// and repository views").
type activityStore struct{ store *Store }

var _ state.ActivityStore = activityStore{}

type activityRecordedPayload struct {
	Schema     string         `json:"schema"`
	ActivityID string         `json:"activity_id"`
	EffortID   string         `json:"effort_id,omitempty"`
	RunID      string         `json:"run_id,omitempty"`
	Kind       string         `json:"kind"`
	OccurredAt time.Time      `json:"occurred_at"`
	Detail     map[string]any `json:"detail,omitempty"`
}

func (a activityStore) Record(ctx context.Context, params state.ActivityRecordParams) (state.ActivityRecord, error) {
	if params.Kind == "" {
		return state.ActivityRecord{}, fmt.Errorf("agentstore: activity needs a kind")
	}
	now := a.store.options.Now()
	id := a.store.options.NewID()
	record := state.ActivityRecord{ID: id, EffortID: params.EffortID, RunID: params.RunID, Kind: params.Kind, OccurredAt: now, Detail: params.Detail}
	payload := activityRecordedPayload{
		Schema: schemaActivityRecordedV1, ActivityID: id, EffortID: params.EffortID, RunID: params.RunID,
		Kind: params.Kind, OccurredAt: now, Detail: params.Detail,
	}
	if _, err := a.store.appendWithRetry(ctx, KindActivityRecorded, params.EffortID, payload); err != nil {
		return state.ActivityRecord{}, err
	}
	return record, nil
}

func (a activityStore) List(ctx context.Context, filter state.ActivityFilter) ([]state.ActivityRecord, error) {
	events, err := a.store.loadEvents(ctx, KindActivityRecorded)
	if err != nil {
		return nil, err
	}
	result := make([]state.ActivityRecord, 0, len(events))
	for _, event := range events {
		var payload activityRecordedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("agentstore: decode %s: %w", KindActivityRecorded, err)
		}
		if filter.EffortID != "" && payload.EffortID != filter.EffortID {
			continue
		}
		if filter.RunID != "" && payload.RunID != filter.RunID {
			continue
		}
		if !filter.Since.IsZero() && payload.OccurredAt.Before(filter.Since) {
			continue
		}
		result = append(result, state.ActivityRecord{
			ID: payload.ActivityID, EffortID: payload.EffortID, RunID: payload.RunID, Kind: payload.Kind,
			OccurredAt: payload.OccurredAt, Detail: payload.Detail,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OccurredAt.Before(result[j].OccurredAt) })
	return result, nil
}
