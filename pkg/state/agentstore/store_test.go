package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology, state-store/journal-batching

import (
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

func TestNewValidatesOptions(t *testing.T) {
	journal, err := replication.NewMemoryJournal(replication.MemoryJournalOptions{
		ProjectID: "p", EndpointID: "server", Role: replication.RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch,
	})
	if err != nil {
		t.Fatalf("NewMemoryJournal: %v", err)
	}
	cases := []struct {
		name string
		opts Options
	}{
		{"missing project id", Options{ActorID: "a", AuthorityEpoch: 1}},
		{"missing actor id", Options{ProjectID: "p", AuthorityEpoch: 1}},
		{"zero epoch", Options{ProjectID: "p", ActorID: "a"}},
		{"negative epoch", Options{ProjectID: "p", ActorID: "a", AuthorityEpoch: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(journal, tc.opts); err == nil {
				t.Fatalf("New(%+v) unexpectedly succeeded", tc.opts)
			}
		})
	}
	if _, err := New(nil, Options{ProjectID: "p", ActorID: "a", AuthorityEpoch: 1}); err == nil {
		t.Fatal("New with nil journal unexpectedly succeeded")
	}
}

func TestEventKindsIsSortedAndComplete(t *testing.T) {
	kinds := EventKinds()
	if len(kinds) == 0 {
		t.Fatal("EventKinds returned nothing")
	}
	for i := 1; i < len(kinds); i++ {
		if kinds[i-1] >= kinds[i] {
			t.Fatalf("EventKinds not sorted at %d: %q >= %q", i, kinds[i-1], kinds[i])
		}
	}
	want := map[string]bool{
		KindEffortCreated: true, KindRunStarted: true, KindRunFinished: true, KindRunCorrected: true,
		KindLeaseAcquired: true, KindLeaseRenewed: true, KindLeaseReleased: true,
		KindWorktreeClaimed: true, KindWorktreeRenewed: true, KindWorktreeReleased: true,
		KindActivityRecorded: true, KindCursorAdvanced: true, KindMessageSent: true, KindMessageAcknowledged: true,
	}
	got := map[string]bool{}
	for _, kind := range kinds {
		got[kind] = true
	}
	for kind := range want {
		if !got[kind] {
			t.Fatalf("EventKinds missing %q", kind)
		}
	}
}

// TestTopologyRejectsZeroOrMultipleActiveStaysGreen is a package-local
// reminder that state-store/topology#ac:topology-rejects-zero-or-multiple-active
// is replication.Topology.Validate's job (already covered by
// pkg/state/replication's own tests). agentstore does not reimplement
// topology validation — Store.New only validates that this instance's own
// AuthorityEpoch is well-formed, not the whole project's endpoint topology.
func TestTopologyRejectsZeroOrMultipleActiveStaysGreen(t *testing.T) {
	valid := replication.Topology{ProjectID: "p", Endpoints: []replication.Endpoint{
		{ID: "active", Backend: replication.BackendGit, Role: replication.RoleActive},
		{ID: "mirror", Backend: replication.BackendSQLite, Role: replication.RoleReplica, Purpose: replication.PurposeMirror},
	}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid topology unexpectedly rejected: %v", err)
	}
	zero := valid
	zero.Endpoints = []replication.Endpoint{valid.Endpoints[1]}
	if err := zero.Validate(); err == nil {
		t.Fatal("topology with zero active endpoints unexpectedly accepted")
	}
	multiple := valid
	multiple.Endpoints = append(append([]replication.Endpoint{}, valid.Endpoints...), replication.Endpoint{ID: "active-2", Backend: replication.BackendSQLite, Role: replication.RoleActive})
	if err := multiple.Validate(); err == nil {
		t.Fatal("topology with two active endpoints unexpectedly accepted")
	}
}
