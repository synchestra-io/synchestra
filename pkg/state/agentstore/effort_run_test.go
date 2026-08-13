package agentstore

// Features implemented: state-store, agent-coordination

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

func TestEffortCreateGetList(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")

	effort, err := store.Effort().Create(ctx, state.EffortCreateParams{
		ProjectID: "p", RepositoryID: "relay", Title: "Fair Split relay", ScopeAreas: []string{"pkg/relay"},
		InitiatorRunID: "run-a", WorkLogRef: "worklog:effort-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if effort.ID == "" || effort.CreatedAt.IsZero() {
		t.Fatalf("unexpected effort: %+v", effort)
	}

	got, err := store.Effort().Get(ctx, effort.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, effort) {
		t.Fatalf("Get returned %+v, want %+v", got, effort)
	}

	if _, err := store.Effort().Get(ctx, "missing"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Get missing error = %v, want ErrNotFound", err)
	}

	if _, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "other", Title: "second"}); err != nil {
		t.Fatalf("second Create: %v", err)
	}
	list, err := store.Effort().List(ctx, state.EffortFilter{RepositoryID: "relay"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != effort.ID {
		t.Fatalf("List(RepositoryID=relay) = %+v", list)
	}
}

func TestRunLifecycleFactsAndModelProvenance(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	effort, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "relay", Title: "t"})
	if err != nil {
		t.Fatalf("Create effort: %v", err)
	}

	// A model must carry a provenance label.
	model := "claude-opus-4-6"
	if _, err := store.Run().Start(ctx, state.RunStartParams{EffortID: effort.ID, AgentFamily: state.AgentFamilyClaude, Model: &model, Role: state.RunRolePrimary}); err == nil {
		t.Fatal("Start with model but no provenance unexpectedly succeeded")
	}

	run, err := store.Run().Start(ctx, state.RunStartParams{
		EffortID: effort.ID, AgentFamily: state.AgentFamilyClaude, Model: &model,
		ModelProvenance: state.ModelProvenanceCallerDeclared, Role: state.RunRolePrimary,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if run.EndedAt != nil {
		t.Fatalf("freshly started run has EndedAt: %+v", run)
	}

	got, err := store.Run().Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Model == nil || *got.Model != model || got.ModelProvenance != state.ModelProvenanceCallerDeclared {
		t.Fatalf("unexpected run model: %+v", got)
	}

	// Optional model provenance is correctable without rewriting history: the
	// run.started event is untouched, a run.corrected event supersedes it.
	corrected := "claude-opus-4-7"
	updated, err := store.Run().Correct(ctx, run.ID, state.RunCorrection{
		Model: &corrected, ModelProvenance: state.ModelProvenanceRuntimeObserved, Reason: "runtime later exposed the real id",
	})
	if err != nil {
		t.Fatalf("Correct: %v", err)
	}
	if updated.Model == nil || *updated.Model != corrected || updated.ModelProvenance != state.ModelProvenanceRuntimeObserved {
		t.Fatalf("unexpected corrected run: %+v", updated)
	}
	events, err := store.Journal().After(ctx, state.Cursor{})
	if err != nil {
		t.Fatalf("Journal().After: %v", err)
	}
	startedSeen, correctedSeen := false, false
	for _, event := range events {
		if event.Kind == KindRunStarted {
			startedSeen = true
		}
		if event.Kind == KindRunCorrected {
			correctedSeen = true
		}
	}
	if !startedSeen || !correctedSeen {
		t.Fatalf("expected both run.started and run.corrected in journal, got %+v", events)
	}

	finished, err := store.Run().Finish(ctx, run.ID, "landed")
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if finished.EndedAt == nil || finished.TerminalReason != "landed" {
		t.Fatalf("unexpected finished run: %+v", finished)
	}
	if _, err := store.Run().Finish(ctx, run.ID, "landed-again"); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("second Finish error = %v, want ErrInvalidTransition", err)
	}

	list, err := store.Run().List(ctx, state.RunFilter{EffortID: effort.ID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List = %+v, want 1 run", list)
	}
}

func TestRunStartRequiresEffortID(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Run().Start(ctx, state.RunStartParams{AgentFamily: state.AgentFamilyCodex}); err == nil {
		t.Fatal("Start without effort id unexpectedly succeeded")
	}
}
