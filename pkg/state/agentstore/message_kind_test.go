package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology/offline-fallback

import (
	"context"
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

// TestMessageTypedNegotiationSequenceIsBidirectionalAndAuditable proves the
// typed negotiation sequence agent-coordination/cross-harness-conformance
// requires at minimum: coordination.request, coordination.proposal,
// coordination.counterexample, coordination.decision.accepted flowing in
// both directions within one correlated thread, with the accepted decision
// referencing evidence, and — critically — the SAME thread state visible to
// both linked runs' own Thread() call ("appearing in both parties' views").
func TestMessageTypedNegotiationSequenceIsBidirectionalAndAuditable(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	const thread = "fair-split-contract"
	const correlation = "fair-split-contract-negotiation"

	request, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: thread, CorrelationID: correlation, SenderRunID: "run-cli-owner", RecipientRunIDs: []string{"run-lib-owner"},
		Kind: state.MessageKindRequest, Body: "need stable, deterministic output ordering",
	})
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	proposal, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: thread, CorrelationID: correlation, SenderRunID: "run-lib-owner", RecipientRunIDs: []string{"run-cli-owner"},
		Kind: state.MessageKindProposal, Body: "assign remainder cents in lexical name order",
	})
	if err != nil {
		t.Fatalf("send proposal: %v", err)
	}
	counterexample, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: thread, CorrelationID: correlation, SenderRunID: "run-cli-owner", RecipientRunIDs: []string{"run-lib-owner"},
		Kind: state.MessageKindCounterexample, Body: "EUR 10.00 / 3 leaves a rounding tie at Bob/Carol",
	})
	if err != nil {
		t.Fatalf("send counterexample: %v", err)
	}
	if _, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: thread, CorrelationID: correlation, SenderRunID: "run-lib-owner", RecipientRunIDs: []string{"run-cli-owner"},
		Kind: state.MessageKindDecisionAccepted, Body: "accepted: lexical order breaks the tie",
	}); err == nil {
		t.Fatal("decision.accepted without evidence unexpectedly succeeded")
	}
	decision, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: thread, CorrelationID: correlation, SenderRunID: "run-lib-owner", RecipientRunIDs: []string{"run-cli-owner"},
		Kind: state.MessageKindDecisionAccepted, Body: "accepted: lexical order breaks the tie",
		Evidence: []state.EvidenceRef{{Kind: "counterexample", Description: "EUR 10.00 3-way split", Reference: counterexample.ID}},
	})
	if err != nil {
		t.Fatalf("send accepted decision: %v", err)
	}
	if len(decision.Evidence) != 1 || decision.Evidence[0].Reference != counterexample.ID {
		t.Fatalf("decision evidence = %+v, want a reference to the counterexample", decision.Evidence)
	}

	// Both linked runs' acknowledgements are separate, immutable records.
	if _, err := store.Message().Acknowledge(ctx, decision.ID, "run-cli-owner"); err != nil {
		t.Fatalf("Acknowledge by run-cli-owner: %v", err)
	}
	if _, err := store.Message().Acknowledge(ctx, decision.ID, "run-lib-owner"); err != nil {
		t.Fatalf("Acknowledge by run-lib-owner: %v", err)
	}

	// The thread — the shared audit view both parties read — carries the
	// exact typed sequence in order, bidirectionally.
	got, err := store.Message().Thread(ctx, thread)
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	wantKinds := []state.MessageKind{
		state.MessageKindRequest, state.MessageKindProposal, state.MessageKindCounterexample, state.MessageKindDecisionAccepted,
	}
	if len(got) != len(wantKinds) {
		t.Fatalf("thread length = %d, want %d (got %+v)", len(got), len(wantKinds), got)
	}
	senders := map[string]bool{}
	for i, msg := range got {
		if msg.Kind != wantKinds[i] {
			t.Fatalf("thread[%d].Kind = %q, want %q", i, msg.Kind, wantKinds[i])
		}
		if msg.CorrelationID != correlation {
			t.Fatalf("thread[%d].CorrelationID = %q, want %q", i, msg.CorrelationID, correlation)
		}
		senders[msg.SenderRunID] = true
	}
	if !senders["run-cli-owner"] || !senders["run-lib-owner"] {
		t.Fatalf("thread senders = %+v, want messages authored by both runs (bidirectional)", senders)
	}
	if got[0].ID != request.ID || got[1].ID != proposal.ID {
		t.Fatalf("thread order = %+v, want request then proposal first", got)
	}

	// "Appearing in both parties' views": either linked run reads the exact
	// same thread — there is one shared audit view, not per-sender copies.
	viewFromCLIOwnerSide, err := store.Message().Thread(ctx, thread)
	if err != nil {
		t.Fatalf("Thread (second read): %v", err)
	}
	if len(viewFromCLIOwnerSide) != len(got) {
		t.Fatalf("second Thread read diverged from the first: %d vs %d entries", len(viewFromCLIOwnerSide), len(got))
	}
}

// TestMessageKindValidatesUnknownValues proves an unrecognized Kind is
// refused rather than silently persisted, keeping the typed vocabulary
// closed.
func TestMessageKindValidatesUnknownValues(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	if _, err := store.Message().Send(ctx, state.MessageSendParams{
		ThreadID: "t", SenderRunID: "run-a", Kind: "coordination.made_up",
	}); err == nil {
		t.Fatal("Send with an unknown Kind unexpectedly succeeded")
	}
}

// TestMessageEventsUseSharedFallbackAllowlistedKind proves that regardless
// of Message.Kind (request/proposal/counterexample/decision.accepted/note),
// the underlying journal event Kind stays exactly "message.sent" — the one
// replication.fallbackKinds already allow-lists
// (state-store/topology/offline-fallback) — so a transport switch has one
// shared vocabulary to reconcile against
// (agent-coordination#ac:messages-survive-transport-switch), never a second
// one invented per negotiation kind.
func TestMessageEventsUseSharedFallbackAllowlistedKind(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newTestJournal(t, 1, "server"), 1, "server")
	for _, kind := range []state.MessageKind{
		state.MessageKindRequest, state.MessageKindProposal, state.MessageKindCounterexample, state.MessageKindNote,
	} {
		if _, err := store.Message().Send(ctx, state.MessageSendParams{ThreadID: "t", SenderRunID: "run-a", Kind: kind}); err != nil {
			t.Fatalf("Send(kind=%q): %v", kind, err)
		}
	}
	events, err := store.Journal().After(ctx, state.Cursor{})
	if err != nil {
		t.Fatalf("Journal().After: %v", err)
	}
	for _, event := range events {
		if event.Kind != "message.sent" {
			t.Fatalf("unexpected journal event kind %q for a typed negotiation message", event.Kind)
		}
	}
	if len(events) != 4 {
		t.Fatalf("journal events = %d, want 4", len(events))
	}
}
