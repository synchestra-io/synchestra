package agentstore

// Features implemented: state-store, agent-coordination

import (
	"context"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// TestEffortTransitionWalksTheLifecycleTable proves a legal end-to-end walk
// through the lifecycle state machine (planning -> active ->
// awaiting_cleanup -> completed -> archived) records each move as a typed,
// audited event and updates Effort.Status/Disposition accordingly.
func TestEffortTransitionWalksTheLifecycleTable(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	effort, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "r", Title: "fair split"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if effort.Status != state.LifecycleStatusPlanning {
		t.Fatalf("new effort status = %q, want planning", effort.Status)
	}

	for _, to := range []state.LifecycleStatus{
		state.LifecycleStatusActive, state.LifecycleStatusAwaitingCleanup,
	} {
		effort, err = store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: to})
		if err != nil {
			t.Fatalf("Transition to %q: %v", to, err)
		}
		if effort.Status != to {
			t.Fatalf("effort status = %q, want %q", effort.Status, to)
		}
	}

	effort, err = store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{
		To: state.LifecycleStatusCompleted, Disposition: state.CompletionDispositionLanded,
	})
	if err != nil {
		t.Fatalf("Transition to completed: %v", err)
	}
	if effort.Disposition != state.CompletionDispositionLanded {
		t.Fatalf("effort disposition = %q, want landed", effort.Disposition)
	}

	if effort, err = store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: state.LifecycleStatusArchived}); err != nil {
		t.Fatalf("Transition to archived: %v", err)
	}
	if effort.Status != state.LifecycleStatusArchived {
		t.Fatalf("effort status = %q, want archived", effort.Status)
	}

	// Get() reflects the same folded state as the Transition return values.
	got, err := store.Effort().Get(ctx, effort.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != state.LifecycleStatusArchived || got.Disposition != state.CompletionDispositionLanded {
		t.Fatalf("Get after archiving = %+v, want archived/landed", got)
	}
}

// TestEffortTransitionRefusesIllegalMoves proves an out-of-table move is
// refused with a typed error (state.ErrInvalidTransition), never silently
// accepted or left to a generic error.
func TestEffortTransitionRefusesIllegalMoves(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	effort, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "r", Title: "t"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// planning -> completed skips the entire table.
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{
		To: state.LifecycleStatusCompleted, Disposition: state.CompletionDispositionLanded,
	}); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("illegal transition error = %v, want ErrInvalidTransition", err)
	}

	// An empty target status is refused, typed the same as every other
	// illegal move.
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{}); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("Transition with an empty target error = %v, want ErrInvalidTransition", err)
	}

	// A terminal target without a disposition is refused.
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: state.LifecycleStatusActive}); err != nil {
		t.Fatalf("Transition to active: %v", err)
	}
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: state.LifecycleStatusAborted}); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("Transition to aborted without a disposition error = %v, want ErrInvalidTransition", err)
	}

	// A non-terminal target with a disposition set is also refused.
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{
		To: state.LifecycleStatusAwaitingMerge, Disposition: state.CompletionDispositionLanded,
	}); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("Transition to a non-terminal status with a disposition error = %v, want ErrInvalidTransition", err)
	}

	// archived is absorbing: nothing transitions out of it.
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{
		To: state.LifecycleStatusAborted, Disposition: state.CompletionDispositionDiscarded,
	}); err != nil {
		t.Fatalf("Transition to aborted: %v", err)
	}
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: state.LifecycleStatusArchived}); err != nil {
		t.Fatalf("Transition to archived: %v", err)
	}
	if _, err := store.Effort().Transition(ctx, effort.ID, state.EffortTransitionParams{To: state.LifecycleStatusActive}); !errors.Is(err, state.ErrInvalidTransition) {
		t.Fatalf("transition out of archived error = %v, want ErrInvalidTransition", err)
	}

	if _, err := store.Effort().Transition(ctx, "does-not-exist", state.EffortTransitionParams{To: state.LifecycleStatusActive}); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Transition on unknown effort error = %v, want ErrNotFound", err)
	}
}

// TestRunTransitionWalksTheLifecycleTable is the run-scoped twin of
// TestEffortTransitionWalksTheLifecycleTable: a Run starts active (it
// records an execution attempt already under way, not a pre-work planning
// stage), and Transition/handoff_pending -> superseded models a run that
// lost a handoff.
func TestRunTransitionWalksTheLifecycleTable(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	effort, err := store.Effort().Create(ctx, state.EffortCreateParams{ProjectID: "p", RepositoryID: "r", Title: "t"})
	if err != nil {
		t.Fatalf("Create effort: %v", err)
	}
	run, err := store.Run().Start(ctx, state.RunStartParams{EffortID: effort.ID, AgentFamily: state.AgentFamilyClaude, Role: state.RunRolePrimary})
	if err != nil {
		t.Fatalf("Start run: %v", err)
	}
	if run.Status != state.LifecycleStatusActive {
		t.Fatalf("new run status = %q, want active", run.Status)
	}

	run, err = store.Run().Transition(ctx, run.ID, state.RunTransitionParams{To: state.LifecycleStatusHandoffPending})
	if err != nil {
		t.Fatalf("Transition to handoff_pending: %v", err)
	}
	run, err = store.Run().Transition(ctx, run.ID, state.RunTransitionParams{To: state.LifecycleStatusSuperseded, Disposition: state.CompletionDispositionHandoff})
	if err != nil {
		t.Fatalf("Transition to superseded: %v", err)
	}
	if run.Disposition != state.CompletionDispositionHandoff {
		t.Fatalf("run disposition = %q, want handoff", run.Disposition)
	}

	// Finish (a fact recording) and Transition remain orthogonal: Finish
	// after superseded is still allowed, since it only ever checks EndedAt.
	if _, err := store.Run().Finish(ctx, run.ID, "handed off to successor"); err != nil {
		t.Fatalf("Finish: %v", err)
	}
}
