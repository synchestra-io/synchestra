package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology/offline-fallback

import (
	"context"
	"errors"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

func TestMessageSendThreadAndAcknowledge(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")

	first, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: "fair-split", SenderRunID: "run-a", RecipientRunIDs: []string{"run-b"}, Body: "proposal",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	second, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: "fair-split", SenderRunID: "run-b", RecipientRunIDs: []string{"run-a"}, Body: "counter",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !second.SentAt.After(first.SentAt) {
		t.Fatalf("second message did not sort after first: %+v, %+v", first, second)
	}

	thread, err := store.Message().Thread(ctx, "fair-split")
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	if len(thread) != 2 || thread[0].ID != first.ID || thread[1].ID != second.ID {
		t.Fatalf("Thread order = %+v", thread)
	}

	ack, err := store.Message().Acknowledge(ctx, first.ID, "run-b")
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if ack.MessageID != first.ID || ack.RecipientRunID != "run-b" {
		t.Fatalf("unexpected ack: %+v", ack)
	}

	if _, err := store.Message().Acknowledge(ctx, "missing", "run-b"); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("Acknowledge missing message error = %v, want ErrNotFound", err)
	}

	// Verify the audited-thread events use the shared fallback-allowlisted
	// kinds (see pkg/state/replication/fallback_inbox.go's fallbackKinds).
	events, err := store.Journal().After(ctx, state.Cursor{})
	if err != nil {
		t.Fatalf("Journal().After: %v", err)
	}
	kinds := map[string]int{}
	for _, event := range events {
		kinds[event.Kind]++
	}
	if kinds["message.sent"] != 2 || kinds["message.acknowledged"] != 1 {
		t.Fatalf("unexpected kind counts: %+v", kinds)
	}
}

func TestActivityRecordAndFilter(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Activity().Record(ctx, state.ActivityRecordParams{EffortID: "e1", RunID: "run-a", Kind: "claim.acquired"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := store.Activity().Record(ctx, state.ActivityRecordParams{EffortID: "e2", RunID: "run-b", Kind: "message.sent"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := store.Activity().Record(ctx, state.ActivityRecordParams{EffortID: ""}); err == nil {
		t.Fatal("Record without kind unexpectedly succeeded")
	}

	list, err := store.Activity().List(ctx, state.ActivityFilter{EffortID: "e1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].RunID != "run-a" {
		t.Fatalf("List(EffortID=e1) = %+v", list)
	}
}
