package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"fmt"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// lifecycleTransitions is the effort/run lifecycle state machine
// (spec/features/agent-coordination's "State machine, merge target, and
// cleanup" section): the set of (from -> to) moves EffortStore.Transition
// and RunStore.Transition allow. Both sub-stores share exactly this table —
// the spec states "Effort and run lifecycle states are..." once, not two
// separate vocabularies — so it lives here rather than being duplicated in
// effort.go/run.go.
//
// archived has no outgoing entry: it is absorbing. A same-state move (e.g.
// active -> active) is deliberately never in this table; Transition always
// records a real state change, not a no-op refresh — a caller that only
// wants to touch RenewedAt-shaped liveness uses Worktree()/Lease() Renew.
var lifecycleTransitions = map[state.LifecycleStatus]map[state.LifecycleStatus]bool{
	state.LifecycleStatusPlanning: {
		state.LifecycleStatusActive:  true,
		state.LifecycleStatusAborted: true,
	},
	state.LifecycleStatusActive: {
		state.LifecycleStatusHandoffPending:  true,
		state.LifecycleStatusAwaitingMerge:   true,
		state.LifecycleStatusAwaitingPush:    true,
		state.LifecycleStatusAwaitingCleanup: true,
		state.LifecycleStatusAborted:         true,
		state.LifecycleStatusFailed:          true,
	},
	state.LifecycleStatusHandoffPending: {
		state.LifecycleStatusSuperseded: true, // successor accepted; this run is done
		state.LifecycleStatusActive:     true, // handoff fell through; original run keeps writing
		state.LifecycleStatusAborted:    true,
	},
	state.LifecycleStatusAwaitingMerge: {
		state.LifecycleStatusAwaitingPush:    true,
		state.LifecycleStatusAwaitingCleanup: true,
		state.LifecycleStatusAborted:         true,
		state.LifecycleStatusFailed:          true,
	},
	state.LifecycleStatusAwaitingPush: {
		state.LifecycleStatusAwaitingCleanup: true,
		state.LifecycleStatusAborted:         true,
		state.LifecycleStatusFailed:          true,
	},
	state.LifecycleStatusAwaitingCleanup: {
		state.LifecycleStatusCompleted: true,
		state.LifecycleStatusAborted:   true,
		state.LifecycleStatusFailed:    true,
	},
	state.LifecycleStatusCompleted: {
		state.LifecycleStatusArchived: true,
	},
	state.LifecycleStatusAborted: {
		state.LifecycleStatusArchived: true,
	},
	state.LifecycleStatusFailed: {
		state.LifecycleStatusArchived: true,
	},
	state.LifecycleStatusSuperseded: {
		state.LifecycleStatusArchived: true,
	},
	// LifecycleStatusArchived: absorbing, no entry.
}

// terminalLifecycleStatuses is the subset of LifecycleStatus that requires a
// CompletionDisposition on the transition that reaches it. superseded is
// terminal for the OUTGOING side of a handoff (its disposition is normally
// CompletionDispositionHandoff) even though the effort/work itself
// continues under the successor run.
var terminalLifecycleStatuses = map[state.LifecycleStatus]bool{
	state.LifecycleStatusCompleted:  true,
	state.LifecycleStatusAborted:    true,
	state.LifecycleStatusFailed:     true,
	state.LifecycleStatusSuperseded: true,
}

// validateLifecycleTransition refuses an illegal (from -> to) move by
// wrapping state.ErrInvalidTransition, and requires a CompletionDisposition
// on any move into a terminal status. kind/id/subject only shape the error
// message (e.g. "effort", "run").
func validateLifecycleTransition(subject, id string, from, to state.LifecycleStatus, disposition state.CompletionDisposition) error {
	if to == "" {
		return fmt.Errorf("agentstore: %s %q transition needs a target status", subject, id)
	}
	allowed := lifecycleTransitions[from]
	if !allowed[to] {
		return fmt.Errorf("agentstore: %s %q cannot transition from %q to %q: %w", subject, id, from, to, state.ErrInvalidTransition)
	}
	if terminalLifecycleStatuses[to] && disposition == "" {
		return fmt.Errorf("agentstore: %s %q transition to terminal status %q needs a disposition", subject, id, to)
	}
	if !terminalLifecycleStatuses[to] && disposition != "" {
		return fmt.Errorf("agentstore: %s %q transition to non-terminal status %q must not set a disposition", subject, id, to)
	}
	return nil
}
