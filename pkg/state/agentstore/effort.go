package agentstore

// Features implemented: state-store, agent-coordination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// effortStore implements state.EffortStore. Efforts have no uniqueness
// precondition (many efforts may target the same repository), so Create is a
// plain appendWithRetry — unlike worktree/lease acquisition, no domain
// precondition needs re-checking on a benign sequence race. Transition does
// have a precondition (the current Status must allow the requested move), so
// it uses snapshot+tryAppendAt like lease.go's Acquire and cursor.go's
// Advance.
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

// effortTransitionedPayload is task-3's audited lifecycle-transition event.
// It is a follow-up event, never a rewrite of effortCreatedPayload's
// immutable record.
type effortTransitionedPayload struct {
	Schema       string                      `json:"schema"`
	EffortID     string                      `json:"effort_id"`
	From         state.LifecycleStatus       `json:"from"`
	To           state.LifecycleStatus       `json:"to"`
	Reason       string                      `json:"reason,omitempty"`
	Disposition  state.CompletionDisposition `json:"disposition,omitempty"`
	TransitionAt time.Time                   `json:"transition_at"`
}

func (s *Store) loadEfforts(ctx context.Context) (map[string]state.Effort, error) {
	events, err := s.loadEvents(ctx, KindEffortCreated, KindEffortTransitioned)
	if err != nil {
		return nil, err
	}
	return foldEfforts(events)
}

// foldEfforts folds a pre-filtered event slice (KindEffortCreated,
// KindEffortTransitioned) into the current effort-by-ID projection. It is
// shared by loadEfforts (a fresh journal read) and Transition (an
// already-snapshotted read pinned to the same observation its append is
// validated against).
func foldEfforts(events []replication.Event) (map[string]state.Effort, error) {
	efforts := make(map[string]state.Effort, len(events))
	for _, event := range events {
		switch event.Kind {
		case KindEffortCreated:
			var payload effortCreatedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindEffortCreated, err)
			}
			efforts[payload.EffortID] = state.Effort{
				ID: payload.EffortID, ProjectID: payload.ProjectID, RepositoryID: payload.RepositoryID,
				Title: payload.Title, ScopeAreas: payload.ScopeAreas, InitiatorRunID: payload.InitiatorRunID,
				WorkLogRef: payload.WorkLogRef, CreatedAt: payload.CreatedAt,
				Status: state.LifecycleStatusPlanning,
			}
		case KindEffortTransitioned:
			var payload effortTransitionedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindEffortTransitioned, err)
			}
			if effort, ok := efforts[payload.EffortID]; ok {
				effort.Status = payload.To
				// A non-terminal target (e.g. superseded -> archived) never
				// carries a Disposition (validateLifecycleTransition
				// refuses one); preserve whatever terminal disposition was
				// already recorded rather than clobbering it with the zero
				// value.
				if payload.Disposition != "" {
					effort.Disposition = payload.Disposition
				}
				efforts[payload.EffortID] = effort
			}
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
		Status: state.LifecycleStatusPlanning,
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

// Transition appends a typed, audited lifecycle-transition event (task-3).
// It re-derives the effort's current status on every retry of a lost
// sequence race — the same snapshot+tryAppendAt shape lease.go's Acquire
// and cursor.go's Advance use — so a transition can never be validated
// against a stale "from" status.
func (e effortStore) Transition(ctx context.Context, effortID string, params state.EffortTransitionParams) (state.Effort, error) {
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		head, headHash, events, err := e.store.snapshot(ctx, KindEffortCreated, KindEffortTransitioned)
		if err != nil {
			return state.Effort{}, err
		}
		efforts, err := foldEfforts(events)
		if err != nil {
			return state.Effort{}, err
		}
		effort, ok := efforts[effortID]
		if !ok {
			return state.Effort{}, fmt.Errorf("agentstore: effort %q: %w", effortID, state.ErrNotFound)
		}
		if err := validateLifecycleTransition("effort", effortID, effort.Status, params.To, params.Disposition); err != nil {
			return state.Effort{}, err
		}
		now := e.store.options.Now()
		payload := effortTransitionedPayload{
			Schema: schemaEffortTransitionedV1, EffortID: effortID, From: effort.Status, To: params.To,
			Reason: params.Reason, Disposition: params.Disposition, TransitionAt: now,
		}
		if _, err := e.store.tryAppendAt(ctx, head, headHash, KindEffortTransitioned, effortID, payload); err != nil {
			if errors.Is(err, errSequenceRace) {
				lastErr = err
				continue
			}
			return state.Effort{}, err
		}
		effort.Status = params.To
		if params.Disposition != "" {
			effort.Disposition = params.Disposition
		}
		return effort, nil
	}
	return state.Effort{}, fmt.Errorf("agentstore: transition effort %q exhausted retries: %w", effortID, lastErr)
}
