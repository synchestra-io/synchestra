package agentstore

// Features implemented: state-store, agent-coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// effortStore implements state.EffortStore. Efforts have no uniqueness
// precondition (many efforts may target the same repository), so Create is a
// plain appendWithRetry — unlike worktree/lease acquisition, no domain
// precondition needs re-checking on a benign sequence race.
type effortStore struct{ store *Store }

var _ state.EffortStore = effortStore{}

type effortCreatedPayload struct {
	Schema         string    `json:"schema"`
	EffortID       string    `json:"effort_id"`
	ProjectID      string    `json:"project_id"`
	RepositoryID   string    `json:"repository_id"`
	Title          string    `json:"title"`
	ScopeAreas     []string  `json:"scope_areas,omitempty"`
	InitiatorRunID string    `json:"initiator_run_id"`
	WorkLogRef     string    `json:"work_log_ref,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Store) loadEfforts(ctx context.Context) (map[string]state.Effort, error) {
	events, err := s.loadEvents(ctx, KindEffortCreated)
	if err != nil {
		return nil, err
	}
	efforts := make(map[string]state.Effort, len(events))
	for _, event := range events {
		var payload effortCreatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("agentstore: decode %s: %w", KindEffortCreated, err)
		}
		efforts[payload.EffortID] = state.Effort{
			ID: payload.EffortID, ProjectID: payload.ProjectID, RepositoryID: payload.RepositoryID,
			Title: payload.Title, ScopeAreas: payload.ScopeAreas, InitiatorRunID: payload.InitiatorRunID,
			WorkLogRef: payload.WorkLogRef, CreatedAt: payload.CreatedAt,
		}
	}
	return efforts, nil
}

func (e effortStore) Create(ctx context.Context, params state.EffortCreateParams) (state.Effort, error) {
	if strings.TrimSpace(params.ProjectID) == "" || strings.TrimSpace(params.RepositoryID) == "" || strings.TrimSpace(params.Title) == "" {
		return state.Effort{}, fmt.Errorf("agentstore: effort needs project id, repository id, and title")
	}
	now := e.store.options.Now()
	id := e.store.options.NewID()
	scope := append([]string(nil), params.ScopeAreas...)
	effort := state.Effort{
		ID: id, ProjectID: params.ProjectID, RepositoryID: params.RepositoryID, Title: params.Title,
		ScopeAreas: scope, InitiatorRunID: params.InitiatorRunID, WorkLogRef: params.WorkLogRef, CreatedAt: now,
	}
	payload := effortCreatedPayload{
		Schema: schemaEffortCreatedV1, EffortID: id, ProjectID: params.ProjectID, RepositoryID: params.RepositoryID,
		Title: params.Title, ScopeAreas: scope, InitiatorRunID: params.InitiatorRunID, WorkLogRef: params.WorkLogRef, CreatedAt: now,
	}
	if _, err := e.store.appendWithRetry(ctx, KindEffortCreated, params.RepositoryID, payload); err != nil {
		return state.Effort{}, err
	}
	return effort, nil
}

func (e effortStore) Get(ctx context.Context, effortID string) (state.Effort, error) {
	efforts, err := e.store.loadEfforts(ctx)
	if err != nil {
		return state.Effort{}, err
	}
	effort, ok := efforts[effortID]
	if !ok {
		return state.Effort{}, fmt.Errorf("agentstore: effort %q: %w", effortID, state.ErrNotFound)
	}
	return effort, nil
}

func (e effortStore) List(ctx context.Context, filter state.EffortFilter) ([]state.Effort, error) {
	efforts, err := e.store.loadEfforts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]state.Effort, 0, len(efforts))
	for _, effort := range efforts {
		if filter.ProjectID != "" && effort.ProjectID != filter.ProjectID {
			continue
		}
		if filter.RepositoryID != "" && effort.RepositoryID != filter.RepositoryID {
			continue
		}
		result = append(result, effort)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}
