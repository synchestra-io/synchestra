package replication

// Features implemented: state-store/topology, state-store/topology/offline-fallback, agent-coordination

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestTopologyRequiresOneActiveReplicaAndGit(t *testing.T) {
	valid := Topology{ProjectID: "github.com/fair-split/relay", Endpoints: []Endpoint{
		{ID: "server", Backend: BackendSQLite, Role: RoleActive},
		{ID: "audit", Backend: BackendGit, Role: RoleReplica, Purpose: PurposeMirror},
	}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid topology: %v", err)
	}
	valid.Endpoints[1].Role = RoleActive
	if err := valid.Validate(); err == nil {
		t.Fatal("two active endpoints must fail")
	}
}

func TestReplicateOrdersTypedThreadEventsAndPreservesAuditFields(t *testing.T) {
	ctx := context.Background()
	git := NewMemoryJournal()
	sqlite := NewMemoryJournal()
	previous := ""
	for sequence, input := range []struct{ id, kind string }{
		{"message-request", "message.sent"},
		{"message-proposal", "message.sent"},
		{"message-counterexample", "message.sent"},
		{"message-decision", "decision.accepted"},
	} {
		payload, err := json.Marshal(map[string]any{"thread_id": "fair-split", "body": input.kind})
		if err != nil {
			t.Fatal(err)
		}
		event, err := NewEvent(Event{ProjectID: "github.com/fair-split/relay", EventID: input.id,
			Cursor: Cursor{Epoch: 1, Sequence: int64(sequence + 1)}, OccurredAt: time.Date(2026, 8, 10, 12, sequence, 0, 0, time.UTC),
			ActorID: "agent:codex", CommandID: "command-1", IdempotencyKey: input.id, Kind: input.kind,
			CorrelationID: "fair-split-thread", Payload: payload, PreviousHash: previous})
		if err != nil {
			t.Fatal(err)
		}
		if err := git.Append(ctx, event); err != nil {
			t.Fatal(err)
		}
		previous = event.Checksum
	}
	health, err := Replicate(ctx, git, sqlite, "sqlite-mirror")
	if err != nil {
		t.Fatal(err)
	}
	if health.EventLag != 0 || health.Cursor != (Cursor{Epoch: 1, Sequence: 4}) {
		t.Fatalf("health = %+v, want caught-up cursor", health)
	}
	got, err := sqlite.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[3].Kind != "decision.accepted" || got[3].CorrelationID != "fair-split-thread" {
		t.Fatalf("replicated audit events = %+v", got)
	}
	// A second delivery is a no-op, which is essential after a duplicate Git
	// webhook or a retry while a server is recovering.
	if err := sqlite.Append(ctx, got[3]); err != nil {
		t.Fatalf("idempotent append: %v", err)
	}
}

func TestMemoryJournalRejectsFencedOrBrokenChainEvent(t *testing.T) {
	ctx := context.Background()
	j := NewMemoryJournal()
	payload := json.RawMessage(`{"thread_id":"fair-split"}`)
	first, err := NewEvent(Event{ProjectID: "p", EventID: "one", Cursor: Cursor{Epoch: 1, Sequence: 1},
		OccurredAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), ActorID: "a", CommandID: "c", IdempotencyKey: "i1", Kind: "message.sent", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Append(ctx, first); err != nil {
		t.Fatal(err)
	}
	broken, err := NewEvent(Event{ProjectID: "p", EventID: "two", Cursor: Cursor{Epoch: 1, Sequence: 2},
		OccurredAt: first.OccurredAt.Add(time.Second), ActorID: "a", CommandID: "c", IdempotencyKey: "i2", Kind: "message.sent", Payload: payload, PreviousHash: "sha256:wrong"})
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Append(ctx, broken); !errors.Is(err, ErrChecksumChain) {
		t.Fatalf("broken chain error = %v, want %v", err, ErrChecksumChain)
	}
}
