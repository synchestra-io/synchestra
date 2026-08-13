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

// runStore implements state.RunStore. Like effortStore, runs have no
// uniqueness precondition, so writes use appendWithRetry directly.
type runStore struct{ store *Store }

var _ state.RunStore = runStore{}

type runStartedPayload struct {
	Schema          string                `json:"schema"`
	RunID           string                `json:"run_id"`
	EffortID        string                `json:"effort_id"`
	AgentFamily     state.AgentFamily     `json:"agent_family"`
	Model           *string               `json:"model,omitempty"`
	ModelProvenance state.ModelProvenance `json:"model_provenance,omitempty"`
	Role            state.RunRole         `json:"role"`
	ParentRunID     string                `json:"parent_run_id,omitempty"`
	StartedAt       time.Time             `json:"started_at"`
}

type runFinishedPayload struct {
	Schema         string    `json:"schema"`
	RunID          string    `json:"run_id"`
	TerminalReason string    `json:"terminal_reason"`
	EndedAt        time.Time `json:"ended_at"`
}

type runCorrectedPayload struct {
	Schema          string                `json:"schema"`
	RunID           string                `json:"run_id"`
	Model           *string               `json:"model,omitempty"`
	ModelProvenance state.ModelProvenance `json:"model_provenance,omitempty"`
	Reason          string                `json:"reason,omitempty"`
	CorrectedAt     time.Time             `json:"corrected_at"`
}

func (s *Store) loadRuns(ctx context.Context) (map[string]state.Run, error) {
	events, err := s.loadEvents(ctx, KindRunStarted, KindRunFinished, KindRunCorrected)
	if err != nil {
		return nil, err
	}
	runs := make(map[string]state.Run, len(events))
	for _, event := range events {
		switch event.Kind {
		case KindRunStarted:
			var payload runStartedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindRunStarted, err)
			}
			runs[payload.RunID] = state.Run{
				ID: payload.RunID, EffortID: payload.EffortID, AgentFamily: payload.AgentFamily,
				Model: payload.Model, ModelProvenance: payload.ModelProvenance, Role: payload.Role,
				ParentRunID: payload.ParentRunID, StartedAt: payload.StartedAt,
			}
		case KindRunFinished:
			var payload runFinishedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindRunFinished, err)
			}
			if run, ok := runs[payload.RunID]; ok {
				endedAt := payload.EndedAt
				run.EndedAt = &endedAt
				run.TerminalReason = payload.TerminalReason
				runs[payload.RunID] = run
			}
		case KindRunCorrected:
			var payload runCorrectedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, fmt.Errorf("agentstore: decode %s: %w", KindRunCorrected, err)
			}
			if run, ok := runs[payload.RunID]; ok {
				run.Model = payload.Model
				run.ModelProvenance = payload.ModelProvenance
				runs[payload.RunID] = run
			}
		}
	}
	return runs, nil
}

func (r runStore) Start(ctx context.Context, params state.RunStartParams) (state.Run, error) {
	if strings.TrimSpace(params.EffortID) == "" {
		return state.Run{}, fmt.Errorf("agentstore: run needs an effort id")
	}
	if err := validateModelProvenance(params.Model, params.ModelProvenance); err != nil {
		return state.Run{}, err
	}
	now := r.store.options.Now()
	id := r.store.options.NewID()
	run := state.Run{
		ID: id, EffortID: params.EffortID, AgentFamily: params.AgentFamily, Model: params.Model,
		ModelProvenance: params.ModelProvenance, Role: params.Role, ParentRunID: params.ParentRunID, StartedAt: now,
	}
	payload := runStartedPayload{
		Schema: schemaRunStartedV1, RunID: id, EffortID: params.EffortID, AgentFamily: params.AgentFamily,
		Model: params.Model, ModelProvenance: params.ModelProvenance, Role: params.Role, ParentRunID: params.ParentRunID, StartedAt: now,
	}
	if _, err := r.store.appendWithRetry(ctx, KindRunStarted, params.EffortID, payload); err != nil {
		return state.Run{}, err
	}
	return run, nil
}

func (r runStore) Get(ctx context.Context, runID string) (state.Run, error) {
	runs, err := r.store.loadRuns(ctx)
	if err != nil {
		return state.Run{}, err
	}
	run, ok := runs[runID]
	if !ok {
		return state.Run{}, fmt.Errorf("agentstore: run %q: %w", runID, state.ErrNotFound)
	}
	return run, nil
}

func (r runStore) List(ctx context.Context, filter state.RunFilter) ([]state.Run, error) {
	runs, err := r.store.loadRuns(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]state.Run, 0, len(runs))
	for _, run := range runs {
		if filter.EffortID != "" && run.EffortID != filter.EffortID {
			continue
		}
		result = append(result, run)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.Before(result[j].StartedAt) })
	return result, nil
}

func (r runStore) Finish(ctx context.Context, runID string, terminalReason string) (state.Run, error) {
	run, err := r.Get(ctx, runID)
	if err != nil {
		return state.Run{}, err
	}
	if run.EndedAt != nil {
		return state.Run{}, fmt.Errorf("agentstore: run %q already finished: %w", runID, state.ErrInvalidTransition)
	}
	now := r.store.options.Now()
	payload := runFinishedPayload{Schema: schemaRunFinishedV1, RunID: runID, TerminalReason: terminalReason, EndedAt: now}
	if _, err := r.store.appendWithRetry(ctx, KindRunFinished, run.EffortID, payload); err != nil {
		return state.Run{}, err
	}
	run.EndedAt = &now
	run.TerminalReason = terminalReason
	return run, nil
}

func (r runStore) Correct(ctx context.Context, runID string, correction state.RunCorrection) (state.Run, error) {
	run, err := r.Get(ctx, runID)
	if err != nil {
		return state.Run{}, err
	}
	if err := validateModelProvenance(correction.Model, correction.ModelProvenance); err != nil {
		return state.Run{}, err
	}
	now := r.store.options.Now()
	payload := runCorrectedPayload{
		Schema: schemaRunCorrectedV1, RunID: runID, Model: correction.Model, ModelProvenance: correction.ModelProvenance,
		Reason: correction.Reason, CorrectedAt: now,
	}
	if _, err := r.store.appendWithRetry(ctx, KindRunCorrected, run.EffortID, payload); err != nil {
		return state.Run{}, err
	}
	run.Model = correction.Model
	run.ModelProvenance = correction.ModelProvenance
	return run, nil
}

// validateModelProvenance enforces "a model value is never guessed": a
// non-nil model must carry a provenance label, and a nil model must not.
func validateModelProvenance(model *string, provenance state.ModelProvenance) error {
	if model == nil {
		if provenance != "" {
			return fmt.Errorf("agentstore: model provenance must be empty when model is unknown")
		}
		return nil
	}
	if provenance != state.ModelProvenanceRuntimeObserved && provenance != state.ModelProvenanceCallerDeclared {
		return fmt.Errorf("agentstore: model %q needs runtime_observed or caller_declared provenance", *model)
	}
	return nil
}
