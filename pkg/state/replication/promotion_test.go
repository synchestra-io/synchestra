package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/backends/git, state-store/backends/sqlite, agent-coordination, state-store/journal-batching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dal-go/dalgo/dal"
	"github.com/dal-go/dalgo2sqlite"
)

// chainedEvent is a lightweight event builder for tests that only care about
// cursor/chain mechanics, not the fair-split domain fixture relayEvents(t)
// provides.
func chainedEvent(t *testing.T, projectID, eventID string, cursor Cursor, previousHash string) Event {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"n": eventID})
	if err != nil {
		t.Fatal(err)
	}
	event, err := NewEvent(Event{ProjectID: projectID, EventID: eventID, Cursor: cursor, OccurredAt: time.Now().UTC(),
		ActorID: "agent:test", CommandID: "cmd-" + eventID, IdempotencyKey: eventID, Kind: "message.sent", Payload: payload, PreviousHash: previousHash})
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func newMemoryPair(t *testing.T, projectID, activeID, candidateID string) (*MemoryJournal, *MemoryJournal) {
	t.Helper()
	active, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: projectID, EndpointID: activeID, Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: projectID, EndpointID: candidateID, Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	return active, candidate
}

// TestPromotionConvergenceCheckRefusesLagAndDivergence unit tests
// evaluateConvergence directly against fabricated cursors/hashes. Promote
// itself no longer calls this helper (the fence-first sequence's own
// cursor/checksum comparison in step 2 supersedes it), but it remains the
// general-purpose convergence predicate and is tested in its own right.
func TestPromotionConvergenceCheckRefusesLagAndDivergence(t *testing.T) {
	active := Cursor{Epoch: 1, Sequence: 4}
	cases := []struct {
		name      string
		candidate Cursor
		candHash  string
		activeH   string
		wantErr   error
	}{
		{"behind", Cursor{Epoch: 1, Sequence: 3}, "sha256:same", "sha256:same", ErrPromotionNotConverged},
		{"ahead", Cursor{Epoch: 1, Sequence: 5}, "sha256:same", "sha256:same", ErrReplicaAhead},
		{"diverged_at_equal_cursor", active, "sha256:candidate-fork", "sha256:active-canonical", ErrDiverged},
		{"converged", active, "sha256:same", "sha256:same", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := evaluateConvergence(tc.candidate, tc.candHash, active, tc.activeH)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("converged case returned error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("evaluateConvergence(%s) error = %v, want %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

// TestPromoteCatchesUpLaggingCandidateBeforePromoting proves the fence-first
// redesign's step 2: unlike the old design (which refused a lagging
// candidate outright), Promote now actively catches it up — fencing the
// active first, then delivering every event the candidate was missing,
// including the checkpoint itself, before promoting it.
func TestPromoteCatchesUpLaggingCandidateBeforePromoting(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-mirror"})
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	events := relayEvents(t)
	appendAll(t, active, events)
	// Deliberately do not replicate anything: candidate starts at cursor zero.

	result, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-catchup"})
	if err != nil {
		t.Fatalf("promote a fully-lagging candidate: %v", err)
	}
	if result.Checkpoint.Cursor.Epoch != 2 || result.Checkpoint.Kind != PromotionCheckpointKind {
		t.Fatalf("promotion result = %+v", result)
	}
	candidateEvents, err := candidate.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidateEvents) != len(events)+1 {
		t.Fatalf("candidate after catch-up promotion has %d events, want %d domain events plus the checkpoint", len(candidateEvents), len(events)+1)
	}
	if role, epoch := candidate.roleEpoch(); role != RoleActive || epoch != 2 {
		t.Fatalf("candidate role/epoch after promotion = %v/%d", role, epoch)
	}

	// Partial lag (all but the last event replicated) is caught up the same way.
	_, activeB := newSQLiteJournal(t, []string{"git-mirror-b"})
	_, candidateB := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror-b")
	appendAll(t, activeB, events)
	for _, event := range events[:len(events)-1] {
		if err := candidateB.IngestReplica(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Promote(ctx, activeB, candidateB, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-partial-catchup"}); err != nil {
		t.Fatalf("promote a partially-lagging candidate: %v", err)
	}
	if role, _ := candidateB.roleEpoch(); role != RoleActive {
		t.Fatalf("partially-caught-up candidate role = %v, want active", role)
	}
}

// TestPromoteRefusesDivergedCandidate proves a candidate whose existing chain
// does not match the active's — not merely behind, but built on a different
// history — is still refused, not silently overwritten by catch-up.
func TestPromoteRefusesDivergedCandidate(t *testing.T) {
	ctx := context.Background()
	active, candidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	// The candidate's very first event is NOT the active's real seed -- a
	// diverged foundation, not merely a lagging one.
	forked := chainedEvent(t, "p", "forked", Cursor{1, 1}, "")
	if err := candidate.IngestReplica(ctx, forked); err != nil {
		t.Fatal(err)
	}
	// The candidate's cursor is behind the checkpoint the fence-first design
	// will produce, so Promote's step 2 attempts an ordinary catch-up
	// delivery rather than a pre-flight "diverged at equal cursor" check
	// (that classification -- ErrDiverged -- only applies when the two sides
	// are AT the same cursor; here they are not, until delivery itself hits
	// the incompatible link). The incompatible chain surfaces as
	// ErrChecksumChain once the checkpoint (chained on the REAL active
	// history) is delivered onto the candidate's forked one.
	_, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "op", CommandID: "promote-diverged"})
	if !errors.Is(err, ErrChecksumChain) {
		t.Fatalf("promote a diverged candidate error = %v, want %v", err, ErrChecksumChain)
	}
	// The active is already fenced (fail-toward-zero-writers) and this state
	// is resumable once the operator resolves the candidate's forked history
	// (e.g. by rebuilding it from a fresh replica of the fenced active).
	if role, _ := active.RoleEpoch(); role != RoleReplica {
		t.Fatalf("active role after diverged-candidate refusal = %v, want replica (fenced)", role)
	}
}

// TestPromoteRefusesCandidateAheadOfActive proves a candidate whose cursor is
// beyond the fenced active's checkpoint is refused. Because the checkpoint is
// always the NEXT epoch, "ahead" can only mean the candidate already holds
// something beyond that exact epoch/sequence -- e.g. a stray event an
// operator mistake delivered to it -- not merely a higher sequence within the
// active's current (older) epoch, which is always "behind" the checkpoint by
// epoch alone. This test fabricates that scenario directly on the candidate.
func TestPromoteRefusesCandidateAheadOfActive(t *testing.T) {
	ctx := context.Background()
	active, candidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}
	// A fabricated epoch-2 event (matching the shape Promote's own fence will
	// produce) plus a stray event past it -- the candidate now claims to be
	// ahead of wherever the real fence checkpoint will land.
	fabricatedCheckpoint := chainedEvent(t, "p", "fabricated-checkpoint", Cursor{2, 1}, seed.Checksum)
	if err := candidate.IngestReplica(ctx, fabricatedCheckpoint); err != nil {
		t.Fatal(err)
	}
	strayAhead := chainedEvent(t, "p", "stray-ahead", Cursor{2, 2}, fabricatedCheckpoint.Checksum)
	if err := candidate.IngestReplica(ctx, strayAhead); err != nil {
		t.Fatal(err)
	}

	_, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "op", CommandID: "promote-ahead"})
	if !errors.Is(err, ErrReplicaAhead) {
		t.Fatalf("promote a candidate ahead of the checkpoint error = %v, want %v", err, ErrReplicaAhead)
	}
	// The active is already fenced by the time this is detected (step 1 runs
	// before step 2's ahead check) -- that is the fail-toward-zero-writers
	// invariant, and it is exactly why this state must be (and is) resumable.
	if role, _ := active.RoleEpoch(); role != RoleReplica {
		t.Fatalf("active role after ahead-candidate refusal = %v, want replica (fenced)", role)
	}
}

// TestPromoteRequiresCandidateToBeAReplica guards against promoting an
// endpoint that is already active, for a fresh (non-resumed) promotion.
func TestPromoteRequiresCandidateToBeAReplica(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-active-2"})
	_, notAReplica := newGitJournal(t, nil) // constructed as RoleActive
	_, err := Promote(ctx, active, notAReplica, nil, PromotionRequest{ActorID: "operator", CommandID: "promote-bad-role"})
	if !errors.Is(err, ErrPromotionCandidateNotReplica) {
		t.Fatalf("promote non-replica candidate error = %v, want %v", err, ErrPromotionCandidateNotReplica)
	}
}

// TestPromotionEmptyJournalsRefused proves finding #5: promoting between two
// journals that have never recorded anything is refused early with a typed,
// legible error rather than failing deep inside the checkpoint append with a
// confusing ErrSequenceGap (the checkpoint's epoch is never 1). Chosen
// resolution: (a) an explicit early typed error — see
// ErrPromotionEmptyJournals's doc comment and README.md's rationale.
func TestPromotionEmptyJournalsRefused(t *testing.T) {
	ctx := context.Background()
	active, candidate := newMemoryPair(t, "p", "active", "candidate")
	_, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "op", CommandID: "promote-empty"})
	if !errors.Is(err, ErrPromotionEmptyJournals) {
		t.Fatalf("promote two empty journals error = %v, want %v", err, ErrPromotionEmptyJournals)
	}
	if role, epoch := active.RoleEpoch(); role != RoleActive || epoch != 1 {
		t.Fatalf("active role/epoch mutated on empty-pair refusal: %v/%d", role, epoch)
	}
}

// TestPromoteFencesFormerActiveAndEnablesCandidateWrites is the fence-after-
// promotion proof for state-store/topology#ac:promotion-fences-former-active:
// after Promote, the former active immediately refuses further writes at its
// old epoch (RoleFenceError, since Promote flips its role before anything
// else could observe a stale state), and the newly promoted candidate accepts
// fresh writes continuing the same checksum chain at the new epoch.
func TestPromoteFencesFormerActiveAndEnablesCandidateWrites(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-mirror"})
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	events := relayEvents(t)
	appendAll(t, active, events)
	if _, err := Replicate(ctx, active, candidate, "git-mirror"); err != nil {
		t.Fatal(err)
	}
	assertPhysicalParity(t, active, candidate)

	result, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-1", Reason: "sqlite outage drill"})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if result.Checkpoint.Cursor.Epoch != 2 || result.Checkpoint.Kind != PromotionCheckpointKind {
		t.Fatalf("promotion result = %+v", result)
	}
	if result.Resumed {
		t.Fatalf("a fresh promotion must not report Resumed")
	}
	if len(result.FencedEndpointIDs) != 1 || result.FencedEndpointIDs[0] != "sqlite-active" {
		t.Fatalf("fenced endpoint ids = %v, want [sqlite-active]", result.FencedEndpointIDs)
	}
	if result.FormerActiveEndpointID != "sqlite-active" || result.NewActiveEndpointID != "git-mirror" {
		t.Fatalf("result endpoint ids = former=%q new=%q", result.FormerActiveEndpointID, result.NewActiveEndpointID)
	}

	// The former active self-fences on role alone: it refuses even a write
	// stamped with its own old (now-stale) epoch.
	stale := events[0]
	stale.EventID = "post-promotion-stale"
	stale.IdempotencyKey = "post-promotion-stale"
	stale.Checksum = ""
	stale, err = NewEvent(stale)
	if err != nil {
		t.Fatal(err)
	}
	err = active.Append(ctx, stale)
	var fence *RoleFenceError
	if !errors.Is(err, ErrRoleFenced) || !errors.As(err, &fence) {
		t.Fatalf("former active append error = %v, want role fence", err)
	}
	if fence.Role != RoleReplica || fence.AuthorityEpoch != 2 || fence.EndpointID != "sqlite-active" {
		t.Fatalf("former active fence evidence = %+v", fence)
	}
	// Its physical store also durably reflects the checkpoint, not just its
	// in-memory role field.
	activeHead, _, err := active.Head(ctx)
	if err != nil || activeHead != result.Checkpoint.Cursor {
		t.Fatalf("former active head after fencing = %+v, %v; want checkpoint cursor %+v", activeHead, err, result.Checkpoint.Cursor)
	}

	// The newly promoted candidate accepts fresh writes at the new epoch,
	// chained from the checkpoint.
	nextPayload, err := json.Marshal(map[string]any{"thread_id": "fair-split", "body": "post-promotion"})
	if err != nil {
		t.Fatal(err)
	}
	next, err := NewEvent(Event{ProjectID: "github.com/fair-split/relay", EventID: "post-promotion", Cursor: Cursor{Epoch: 2, Sequence: 2},
		OccurredAt: time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "c", IdempotencyKey: "post-promotion",
		Kind: "message.sent", Payload: nextPayload, PreviousHash: result.Checkpoint.Checksum})
	if err != nil {
		t.Fatal(err)
	}
	if err := candidate.Append(ctx, next); err != nil {
		t.Fatalf("promoted candidate append: %v", err)
	}
	head, _, err := candidate.Head(ctx)
	if err != nil || head != (Cursor{Epoch: 2, Sequence: 2}) {
		t.Fatalf("candidate head after promotion append = %+v, %v", head, err)
	}
}

// TestPromoteFencingSurvivesFormerActiveRestart is the durable, "reconnected
// after being offline" half of ac:promotion-fences-former-active: it closes
// and reopens the former active's physical SQLite file, constructing a
// completely fresh *DALJournal at the OLD role/epoch (as a naive restarted
// process would, with no idea promotion happened), and proves that process
// still gets fenced — not because any in-memory field remembers the
// promotion, but because validateNext refuses a stale-epoch write against the
// higher epoch now durably on disk.
func TestPromoteFencingSurvivesFormerActiveRestart(t *testing.T) {
	ctx := context.Background()
	sqlitePath, active := newSQLiteJournal(t, []string{"git-mirror"})
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	events := relayEvents(t)
	appendAll(t, active, events)
	if _, err := Replicate(ctx, active, candidate, "git-mirror"); err != nil {
		t.Fatal(err)
	}

	if _, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-2"}); err != nil {
		t.Fatal(err)
	}

	if database, ok := active.db.(*dalgo2sqlite.Database); ok {
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
	}
	_, reopened := newSQLiteJournalAt(t, sqlitePath, []string{"git-mirror"}, RoleActive, "sqlite-active")
	stale := events[0]
	stale.EventID = "post-restart-stale"
	stale.IdempotencyKey = "post-restart-stale"
	stale.Checksum = ""
	stale, err := NewEvent(stale)
	if err != nil {
		t.Fatal(err)
	}
	err = reopened.Append(ctx, stale)
	if !errors.Is(err, ErrEpochFenced) {
		t.Fatalf("reopened former active append error = %v, want %v (fenced by durable head, not memory)", err, ErrEpochFenced)
	}
}

// TestPromoteDeliversCheckpointToOtherReachableReplicas proves step 4 of the
// fence-first sequence ("record a signed promotion checkpoint in all
// reachable required replicas") for a replica that is neither the candidate
// nor the former active.
func TestPromoteDeliversCheckpointToOtherReachableReplicas(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-mirror", "sqlite-backup"})
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	_, backup := newSQLiteJournalWithRole(t, nil, RoleReplica, "sqlite-backup")
	events := relayEvents(t)
	appendAll(t, active, events)
	if _, err := Replicate(ctx, active, candidate, "git-mirror"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replicate(ctx, active, backup, "sqlite-backup"); err != nil {
		t.Fatal(err)
	}

	result, err := Promote(ctx, active, candidate, []PromotableJournal{backup}, PromotionRequest{ActorID: "operator", CommandID: "promote-3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FencedEndpointIDs) != 2 || result.FencedEndpointIDs[0] != "sqlite-active" || result.FencedEndpointIDs[1] != "sqlite-backup" {
		t.Fatalf("fenced endpoint ids = %v, want [sqlite-active sqlite-backup]", result.FencedEndpointIDs)
	}
	if len(result.ReplicaOutcomes) != 1 || result.ReplicaOutcomes[0].EndpointID != "sqlite-backup" || result.ReplicaOutcomes[0].Err != nil {
		t.Fatalf("replica outcomes = %+v, want one successful sqlite-backup outcome", result.ReplicaOutcomes)
	}
	backupHead, _, err := backup.Head(ctx)
	if err != nil || backupHead != result.Checkpoint.Cursor {
		t.Fatalf("backup replica head = %+v, %v; want checkpoint cursor %+v", backupHead, err, result.Checkpoint.Cursor)
	}
	// The backup stays a replica — only the candidate is switched to active.
	if role, _ := backup.RoleEpoch(); role != RoleReplica {
		t.Fatalf("backup role = %v, want replica", role)
	}
}

// TestPromoteReplicaFanOutFailureIsNonFatal proves finding #12: a reachable
// replica that fails to accept the checkpoint (e.g. an unreachable backup)
// does not fail the whole Promote call — the swap already durably succeeded
// by the time fan-out runs. The failure is reported per-replica in
// PromotionResult.ReplicaOutcomes instead.
func TestPromoteReplicaFanOutFailureIsNonFatal(t *testing.T) {
	ctx := context.Background()
	active, candidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}
	unreachable := &alwaysFailIngest{PromotableJournal: mustMemoryReplica(t, "p", "unreachable-backup"), err: errors.New("simulated network partition")}

	result, err := Promote(ctx, active, candidate, []PromotableJournal{unreachable}, PromotionRequest{ActorID: "op", CommandID: "promote-nonfatal-fanout"})
	if err != nil {
		t.Fatalf("promote with an unreachable replica must still succeed: %v", err)
	}
	if len(result.ReplicaOutcomes) != 1 || result.ReplicaOutcomes[0].EndpointID != "unreachable-backup" || result.ReplicaOutcomes[0].Err == nil {
		t.Fatalf("replica outcomes = %+v, want one failed unreachable-backup outcome", result.ReplicaOutcomes)
	}
	if len(result.FencedEndpointIDs) != 1 {
		t.Fatalf("fenced endpoint ids = %v, want only the former active (the unreachable replica must not be listed as fenced)", result.FencedEndpointIDs)
	}
	if role, _ := candidate.RoleEpoch(); role != RoleActive {
		t.Fatalf("candidate role after non-fatal fan-out failure = %v, want active (the swap still completed)", role)
	}
}

func mustMemoryReplica(t *testing.T, projectID, endpointID string) *MemoryJournal {
	t.Helper()
	j, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: projectID, EndpointID: endpointID, Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	return j
}

// alwaysFailIngest wraps a PromotableJournal and makes every IngestReplica
// call fail, simulating an unreachable reachableReplicas entry.
type alwaysFailIngest struct {
	PromotableJournal
	err error
}

func (a *alwaysFailIngest) IngestReplica(context.Context, Event) error { return a.err }

// TestPromoteEstablishesNewActiveReplicaSet proves finding #8: PromoteToActive
// sets the newly-active candidate's replica set from PromotionRequest.ReplicaIDs
// (including the fenced former active), and a subsequent domain append on the
// new active fans out to that new topology, not whatever (possibly stale or
// empty) set the candidate was originally constructed with.
func TestPromoteEstablishesNewActiveReplicaSet(t *testing.T) {
	ctx := context.Background()
	active, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1, ReplicaIDs: []string{"candidate"}, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "candidate", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}

	result, err := Promote(ctx, active, candidate, nil, PromotionRequest{
		ActorID: "op", CommandID: "promote-replica-set",
		ReplicaIDs: []string{"active", "sqlite-backup"}, // the fenced former active, plus another mirror
	})
	if err != nil {
		t.Fatal(err)
	}

	// A fresh domain write on the newly-active candidate must fan out to the
	// NEW topology, including the fenced former active.
	next := chainedEvent(t, "p", "post-promotion", Cursor{Epoch: result.Checkpoint.Cursor.Epoch, Sequence: 2}, result.Checkpoint.Checksum)
	if err := candidate.Append(ctx, next); err != nil {
		t.Fatalf("promoted candidate append: %v", err)
	}
	pendingForFormerActive, err := candidate.PendingOutbox(ctx, "active")
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingForFormerActive) != 1 || pendingForFormerActive[0].EventID != "post-promotion" {
		t.Fatalf("outbox for the fenced former active as a new replica = %#v, want the post-promotion event queued", pendingForFormerActive)
	}
	pendingForBackup, err := candidate.PendingOutbox(ctx, "sqlite-backup")
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingForBackup) != 1 || pendingForBackup[0].EventID != "post-promotion" {
		t.Fatalf("outbox for the newly-configured backup replica = %#v, want the post-promotion event queued", pendingForBackup)
	}
}

// TestFenceAsReplicaRefusesNonActiveRole proves finding #10: FenceAsReplica
// has an explicit role precondition — it refuses on a replica (the missing
// guard the finding identified), not just silently misbehaving.
func TestFenceAsReplicaRefusesNonActiveRole(t *testing.T) {
	ctx := context.Background()
	replica, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "already-replica", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	_, err = replica.FenceAsReplica(ctx, PromotionRequest{ActorID: "op", CommandID: "c", IdempotencyKey: "k"}, "candidate")
	if !errors.Is(err, ErrFenceSourceNotActive) {
		t.Fatalf("fence a non-active endpoint error = %v, want %v", err, ErrFenceSourceNotActive)
	}
}

// TestPromoteToActiveSentinelErrorsDistinguishBenignFromDangerous proves
// finding #9: PromoteToActive's precondition failures are typed sentinels an
// operator/caller can distinguish with errors.Is, not bare fmt.Errorf.
func TestPromoteToActiveSentinelErrorsDistinguishBenignFromDangerous(t *testing.T) {
	ctx := context.Background()
	checkpoint := chainedEvent(t, "p", "checkpoint", Cursor{2, 1}, "sha256:whatever")

	t.Run("target_is_active_at_a_different_epoch", func(t *testing.T) {
		active, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "already-active", Role: RoleActive, AuthorityEpoch: 5, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
		if err != nil {
			t.Fatal(err)
		}
		err = active.PromoteToActive(ctx, checkpoint, nil)
		if !errors.Is(err, ErrPromoteTargetWrongEpoch) {
			t.Fatalf("error = %v, want %v", err, ErrPromoteTargetWrongEpoch)
		}
	})

	t.Run("target_has_no_role_precondition_match", func(t *testing.T) {
		// A journal in neither role this test can construct directly is not
		// reachable through the public constructor, so exercise the other
		// documented refusal: a replica whose head does not yet match the
		// checkpoint (not caught up).
		replica, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "not-caught-up", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
		if err != nil {
			t.Fatal(err)
		}
		err = replica.PromoteToActive(ctx, checkpoint, nil)
		if !errors.Is(err, ErrPromotionNotConverged) {
			t.Fatalf("error = %v, want %v", err, ErrPromotionNotConverged)
		}
	})
}

// TestAppendRacingFenceSurfacesRoleFenceError is finding #11's narrow,
// DALJournal-level -race proof: a domain Append racing FenceAsReplica on the
// real backend (not just the MemoryJournal harness) always resolves to
// either success or a *RoleFenceError — never a raw checksum-chain or
// sequence-gap error that would hide the role transition that actually
// explains the failure.
func TestAppendRacingFenceSurfacesRoleFenceError(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, nil)
	seed := relayEvents(t)[0]
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	var writerErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		next := chainedEvent(t, "github.com/fair-split/relay", "racer", Cursor{1, 2}, seed.Checksum)
		writerErr = active.Append(ctx, next)
	}()

	_, err := active.FenceAsReplica(ctx, PromotionRequest{ActorID: "op", CommandID: "fence-race", IdempotencyKey: "fence-race"}, "candidate")
	wg.Wait()
	if err != nil {
		t.Fatalf("fence: %v", err)
	}
	if writerErr != nil {
		var fence *RoleFenceError
		if !errors.As(writerErr, &fence) {
			t.Fatalf("racing writer error = %v, want either success or *RoleFenceError", writerErr)
		}
	}
}

// TestPromoteConcurrentAppendDuringPromotionNeverProducesDualActive is
// required test 1/4 for the redesign: a writer goroutine appends to the
// active in a tight loop while Promote runs concurrently. Every write must
// resolve to exactly "succeeded before the fence" or "*RoleFenceError" —
// never the old bug's ErrChecksumChain that permanently broke the fence and
// produced a durable dual-active.
func TestPromoteConcurrentAppendDuringPromotionNeverProducesDualActive(t *testing.T) {
	ctx := context.Background()
	active, candidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}

	// Pre-build every candidate write up front: chainedEvent calls t.Fatal
	// internally, which is only safe from the test's own goroutine, not the
	// racing writer goroutine below.
	const attempts = 20000
	events := make([]Event, attempts)
	previous := seed.Checksum
	for i := 0; i < attempts; i++ {
		events[i] = chainedEvent(t, "p", fmt.Sprintf("write-%d", i+2), Cursor{1, int64(i + 2)}, previous)
		previous = events[i].Checksum
	}

	var wg sync.WaitGroup
	var readyOnce sync.Once
	ready := make(chan struct{})
	var firstNonFenceErr error
	var appended, fenced int
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer readyOnce.Do(func() { close(ready) })
		for _, event := range events {
			err := active.Append(ctx, event)
			if err == nil {
				appended++
				if appended == 50 {
					readyOnce.Do(func() { close(ready) })
				}
				continue
			}
			var fence *RoleFenceError
			if errors.As(err, &fence) {
				fenced++
			} else {
				firstNonFenceErr = err
			}
			return
		}
	}()

	// Give the writer a real head start so Promote's fence has a genuine
	// chance to land mid-stream rather than winning trivially before the
	// writer's first attempt.
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
	}

	result, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator", CommandID: "promote-race"})
	wg.Wait()
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if firstNonFenceErr != nil {
		t.Fatalf("writer saw a non-RoleFenceError failure racing promotion: %v", firstNonFenceErr)
	}
	if appended == 0 {
		t.Fatalf("writer never got a single append in before the fence; the race window this test targets was not exercised")
	}
	t.Logf("writer appended %d/%d events before observing fenced=%v", appended, attempts, fenced > 0)

	role, epoch := active.RoleEpoch()
	if role != RoleReplica {
		t.Fatalf("active role after race = %v, want replica", role)
	}
	roleC, epochC := candidate.RoleEpoch()
	if roleC != RoleActive || epochC != epoch {
		t.Fatalf("candidate role/epoch after race = %v/%d, want active at the active's fenced epoch %d", roleC, epochC, epoch)
	}
	activeEvents, err := active.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	candidateEvents, err := candidate.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(activeEvents) != len(candidateEvents) {
		t.Fatalf("active has %d events, candidate has %d; want an identical chain (no split-brain divergence)", len(activeEvents), len(candidateEvents))
	}
	for i := range activeEvents {
		if activeEvents[i].Checksum != candidateEvents[i].Checksum {
			t.Fatalf("chain mismatch at position %d: active=%q candidate=%q", i, activeEvents[i].Checksum, candidateEvents[i].Checksum)
		}
	}
	if result.Checkpoint.Kind != PromotionCheckpointKind {
		t.Fatalf("checkpoint kind = %q", result.Checkpoint.Kind)
	}
}

// TestPromoteConcurrentDoublePromoteExactlyOneCandidateWins is required test
// 2/4: two goroutines race Promote against the SAME active with DIFFERENT
// candidates (the old bug: both could flip active at the same epoch, loser
// unrecoverable). The fence-first design serializes both attempts on step 1's
// role/epoch guard before either candidate is touched, so exactly one can
// ever win.
func TestPromoteConcurrentDoublePromoteExactlyOneCandidateWins(t *testing.T) {
	ctx := context.Background()
	active, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	candidateA, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "candidate-a", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	candidateB, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "candidate-b", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidateA.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidateB.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		name string
		res  PromotionResult
		err  error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		res, err := Promote(ctx, active, candidateA, nil, PromotionRequest{ActorID: "op", CommandID: "promote-a"})
		results <- outcome{"a", res, err}
	}()
	go func() {
		defer wg.Done()
		<-start
		res, err := Promote(ctx, active, candidateB, nil, PromotionRequest{ActorID: "op", CommandID: "promote-b"})
		results <- outcome{"b", res, err}
	}()
	close(start)
	wg.Wait()
	close(results)

	var winners, losers []outcome
	for r := range results {
		if r.err == nil {
			winners = append(winners, r)
		} else {
			losers = append(losers, r)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("want exactly one winner, got %d: winners=%v losers=%v", len(winners), winners, losers)
	}
	if len(losers) != 1 {
		t.Fatalf("want exactly one loser, got %d", len(losers))
	}
	loserErr := losers[0].err
	if !errors.Is(loserErr, ErrFenceSourceNotActive) && !errors.Is(loserErr, ErrPromotionSourceNotActive) && !errors.Is(loserErr, ErrPromotionCandidateNotReplica) {
		t.Fatalf("loser error = %v, want a clean typed refusal (ErrFenceSourceNotActive/ErrPromotionSourceNotActive/ErrPromotionCandidateNotReplica)", loserErr)
	}

	winnerCandidate, loserCandidate := candidateA, candidateB
	if winners[0].name == "b" {
		winnerCandidate, loserCandidate = candidateB, candidateA
	}
	if role, _ := winnerCandidate.RoleEpoch(); role != RoleActive {
		t.Fatalf("winning candidate role = %v, want active", role)
	}
	if role, _ := loserCandidate.RoleEpoch(); role != RoleReplica {
		t.Fatalf("losing candidate role = %v, want unchanged replica (never flipped -- the split-brain this test proves fixed)", role)
	}
	if role, _ := active.RoleEpoch(); role != RoleReplica {
		t.Fatalf("active role = %v, want replica (fenced exactly once)", role)
	}
}

// dbFailingFirstN wraps a dal.DB and fails the first n
// RunReadwriteTransaction calls outright, before the worker runs (so nothing
// is written and nothing needs to roll back), then delegates normally
// forever after -- a clean, one-shot transient failure.
type dbFailingFirstN struct {
	dal.DB
	mu        sync.Mutex
	remaining int
}

func (d *dbFailingFirstN) RunReadwriteTransaction(ctx context.Context, worker dal.RWTxWorker, options ...dal.TransactionOption) error {
	d.mu.Lock()
	if d.remaining > 0 {
		d.remaining--
		d.mu.Unlock()
		return errors.New("simulated transient transaction failure")
	}
	d.mu.Unlock()
	return d.DB.RunReadwriteTransaction(ctx, worker, options...)
}

// TestPromoteResumesAfterTransientFenceFailureOnRealBackend is the real-DALgo-
// backend companion to TestPromoteResumesAfterTransientFenceFailure: the
// injected failure happens inside DALJournal.append's actual
// RunReadwriteTransaction call (via dbFailingFirstN), not a test double
// standing in for the whole promotable seam, proving the resumability
// contract holds through the real SQLite transaction path too.
func TestPromoteResumesAfterTransientFenceFailureOnRealBackend(t *testing.T) {
	ctx := context.Background()
	_, baseActive := newSQLiteJournal(t, nil)
	seed := relayEvents(t)[0]
	if err := baseActive.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	failing := &dbFailingFirstN{DB: baseActive.db, remaining: 1}
	active, err := NewDALJournal(failing, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "sqlite-active", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}

	request := PromotionRequest{ActorID: "op", CommandID: "promote-real-flaky-fence", IdempotencyKey: "promotion:sqlite-active:git-mirror"}
	if _, err := Promote(ctx, active, candidate, nil, request); err == nil {
		t.Fatal("expected the injected transient fence failure to surface")
	}
	if role, epoch := active.roleEpoch(); role != RoleActive || epoch != 1 {
		t.Fatalf("active role/epoch after transient fence failure = %v/%d, want completely untouched", role, epoch)
	}

	result, err := Promote(ctx, active, candidate, nil, request)
	if err != nil {
		t.Fatalf("retry after transient fence failure: %v", err)
	}
	if result.Resumed {
		t.Fatalf("a plain retry after an untouched fence failure is a fresh attempt, not a resume")
	}
	if role, _ := candidate.roleEpoch(); role != RoleActive {
		t.Fatalf("candidate role after retry = %v, want active", role)
	}
}

// flakyPromotable wraps a PromotableJournal so its FIRST call to
// FenceAsReplica or IngestReplica (independently counted) fails transiently,
// then delegates normally. It is used to prove Promote's resumability
// contract without needing to reach into a real DB's transaction machinery
// for the MemoryJournal-based tests.
type flakyPromotable struct {
	PromotableJournal
	failFence  int
	failIngest int
}

func (f *flakyPromotable) FenceAsReplica(ctx context.Context, request PromotionRequest, candidateEndpointID string) (Event, error) {
	if f.failFence > 0 {
		f.failFence--
		return Event{}, errors.New("simulated transient fence failure")
	}
	return f.PromotableJournal.FenceAsReplica(ctx, request, candidateEndpointID)
}

func (f *flakyPromotable) IngestReplica(ctx context.Context, event Event) error {
	if f.failIngest > 0 {
		f.failIngest--
		return errors.New("simulated transient ingest failure")
	}
	return f.PromotableJournal.IngestReplica(ctx, event)
}

// TestPromoteResumesAfterTransientFenceFailure is required test 3/4: the
// fence step itself fails once (before anything durable commits), leaving
// the active completely untouched, and a plain retry of the identical
// request succeeds normally.
func TestPromoteResumesAfterTransientFenceFailure(t *testing.T) {
	ctx := context.Background()
	realActive, candidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := realActive.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := candidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}
	flaky := &flakyPromotable{PromotableJournal: realActive, failFence: 1}
	request := PromotionRequest{ActorID: "op", CommandID: "promote-flaky-fence", IdempotencyKey: "promotion:active:candidate"}

	_, err := Promote(ctx, flaky, candidate, nil, request)
	if err == nil {
		t.Fatal("expected the injected transient fence failure to surface")
	}
	if role, epoch := realActive.RoleEpoch(); role != RoleActive || epoch != 1 {
		t.Fatalf("active role/epoch after transient fence failure = %v/%d, want completely untouched", role, epoch)
	}

	result, err := Promote(ctx, flaky, candidate, nil, request)
	if err != nil {
		t.Fatalf("retry after transient fence failure: %v", err)
	}
	if result.Resumed {
		t.Fatalf("a plain retry after an untouched fence failure is a fresh attempt, not a resume")
	}
	if role, _ := candidate.RoleEpoch(); role != RoleActive {
		t.Fatalf("candidate role after retry = %v, want active", role)
	}
}

// TestPromoteResumesAfterTransientPromoteFailure is required test 4/4: the
// fence step succeeds durably, but delivering the checkpoint to the
// candidate during catch-up fails once transiently -- leaving ZERO active
// endpoints, exactly the state Promote's doc comment requires to be
// resumable. Re-invoking Promote with the same request must detect the
// fenced-but-unpromoted state and continue, reporting Resumed.
func TestPromoteResumesAfterTransientPromoteFailure(t *testing.T) {
	ctx := context.Background()
	active, realCandidate := newMemoryPair(t, "p", "active", "candidate")
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	if err := active.Append(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if err := realCandidate.IngestReplica(ctx, seed); err != nil {
		t.Fatal(err)
	}
	// The candidate is already fully caught up on domain events; only the
	// checkpoint itself remains to be delivered during catch-up.
	flakyCandidate := &flakyPromotable{PromotableJournal: realCandidate, failIngest: 1}
	request := PromotionRequest{ActorID: "op", CommandID: "promote-flaky-catchup", IdempotencyKey: "promotion:active:candidate"}

	_, err := Promote(ctx, active, flakyCandidate, nil, request)
	if err == nil {
		t.Fatal("expected the injected transient catch-up failure to surface")
	}
	// Zero active endpoints right now: the fence succeeded (active is now a
	// replica) but the candidate never got promoted.
	if role, _ := active.RoleEpoch(); role != RoleReplica {
		t.Fatalf("active role after fence-then-catchup-failure = %v, want replica (fenced)", role)
	}
	if role, _ := realCandidate.RoleEpoch(); role != RoleReplica {
		t.Fatalf("candidate role after fence-then-catchup-failure = %v, want still replica (never promoted)", role)
	}

	result, err := Promote(ctx, active, flakyCandidate, nil, request)
	if err != nil {
		t.Fatalf("resume after transient catch-up failure: %v", err)
	}
	if !result.Resumed {
		t.Fatalf("resumed promotion must report Resumed=true")
	}
	if role, _ := realCandidate.RoleEpoch(); role != RoleActive {
		t.Fatalf("candidate role after resumed promotion = %v, want active", role)
	}
	if role, _ := active.RoleEpoch(); role != RoleReplica {
		t.Fatalf("active role after resumed promotion = %v, want still replica", role)
	}
}
