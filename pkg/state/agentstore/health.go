package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import (
	"context"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// healthStore implements state.HealthStore. This Store instance always
// reports its own configured Role/AuthorityEpoch: a Store constructed for a
// replica endpoint (Options with Role semantics beyond this package's scope —
// see the package README's Open Questions) would report replica health the
// same way once that wiring exists; Task 1 only wires the active-endpoint
// health an operator needs to see the fence this package enforces.
type healthStore struct{ store *Store }

var _ state.HealthStore = healthStore{}

func (h healthStore) Report(ctx context.Context) (state.HealthReport, error) {
	cursor, _, err := h.store.journal.Head(ctx)
	if err != nil {
		return state.HealthReport{
			EndpointID: h.store.options.EndpointID, Role: "active", AuthorityEpoch: h.store.options.AuthorityEpoch,
			LastError: err.Error(),
		}, nil
	}
	return state.HealthReport{
		EndpointID: h.store.options.EndpointID, Role: "active", AuthorityEpoch: h.store.options.AuthorityEpoch,
		Cursor: toStateCursor(cursor), ReplicaLag: 0, LastOK: h.store.options.Now(),
	}, nil
}
