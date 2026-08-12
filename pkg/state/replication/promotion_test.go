package replication

// Features implemented: state-store/topology

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dal-go/dalgo2sqlite"
)

// TestPromotionConvergenceCheckRefusesLagAndDivergence unit tests
// evaluateConvergence directly against fabricated cursors/hashes, covering
// every refusal case Promote must enforce ("candidate replica is not
// converged (lag > 0 / diverged)") plus the converged pass-through, without
// needing real journals for each case.
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

// TestPromoteRefusesWhenCandidateIsNotConverged proves Promote itself refuses
// (rather than only the internal helper) when the candidate lags the active,
// and that a refused promotion mutates neither endpoint's role or epoch.
func TestPromoteRefusesWhenCandidateIsNotConverged(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-mirror"})
	_, candidate := newGitJournalWithRole(t, nil, RoleReplica, "git-mirror")
	events := relayEvents(t)
	appendAll(t, active, events)
	// Deliberately do not replicate: candidate lags at cursor zero.

	_, err := Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-lag"})
	if !errors.Is(err, ErrPromotionNotConverged) {
		t.Fatalf("lagging candidate promote error = %v, want %v", err, ErrPromotionNotConverged)
	}
	if role, epoch := active.roleEpoch(); role != RoleActive || epoch != 1 {
		t.Fatalf("active role/epoch mutated on refusal: %v/%d", role, epoch)
	}
	if role, epoch := candidate.roleEpoch(); role != RoleReplica || epoch != 1 {
		t.Fatalf("candidate role/epoch mutated on refusal: %v/%d", role, epoch)
	}

	// Replicate all but the last event: still lagging by one, still refused.
	partial := events[:len(events)-1]
	for _, event := range partial {
		if err := candidate.IngestReplica(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	_, err = Promote(ctx, active, candidate, nil, PromotionRequest{ActorID: "operator:alex", CommandID: "promote-partial-lag"})
	if !errors.Is(err, ErrPromotionNotConverged) {
		t.Fatalf("partially-caught-up candidate promote error = %v, want %v", err, ErrPromotionNotConverged)
	}
	if role, _ := candidate.roleEpoch(); role != RoleReplica {
		t.Fatalf("candidate role mutated on partial-lag refusal: %v", role)
	}
}

// TestPromoteRequiresCandidateToBeAReplica guards against promoting an
// endpoint that is already active.
func TestPromoteRequiresCandidateToBeAReplica(t *testing.T) {
	ctx := context.Background()
	_, active := newSQLiteJournal(t, []string{"git-active-2"})
	_, notAReplica := newGitJournal(t, nil) // constructed as RoleActive
	_, err := Promote(ctx, active, notAReplica, nil, PromotionRequest{ActorID: "operator", CommandID: "promote-bad-role"})
	if !errors.Is(err, ErrPromotionCandidateNotReplica) {
		t.Fatalf("promote non-replica candidate error = %v, want %v", err, ErrPromotionCandidateNotReplica)
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
	if result.NewEpoch != 2 || result.Checkpoint.Kind != PromotionCheckpointKind {
		t.Fatalf("promotion result = %+v", result)
	}
	if len(result.FencedEndpointIDs) != 1 || result.FencedEndpointIDs[0] != "sqlite-active" {
		t.Fatalf("fenced endpoint ids = %v, want [sqlite-active]", result.FencedEndpointIDs)
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

// TestPromoteDeliversCheckpointToOtherReachableReplicas proves step 3 of the
// spec's promotion workflow ("record a signed promotion checkpoint in all
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

	result, err := Promote(ctx, active, candidate, []*DALJournal{backup}, PromotionRequest{ActorID: "operator", CommandID: "promote-3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FencedEndpointIDs) != 2 || result.FencedEndpointIDs[0] != "sqlite-active" || result.FencedEndpointIDs[1] != "sqlite-backup" {
		t.Fatalf("fenced endpoint ids = %v, want [sqlite-active sqlite-backup]", result.FencedEndpointIDs)
	}
	backupHead, _, err := backup.Head(ctx)
	if err != nil || backupHead != result.Checkpoint.Cursor {
		t.Fatalf("backup replica head = %+v, %v; want checkpoint cursor %+v", backupHead, err, result.Checkpoint.Cursor)
	}
	// The backup stays a replica — only the candidate is switched to active.
	// Delivering the checkpoint via the ordinary IngestReplica seam durably
	// records the new epoch in its head (asserted above) without needing
	// PromoteToActive/FenceAsReplica's local role/epoch transition, which is
	// reserved for the two endpoints actually changing authority.
	if role, _ := backup.roleEpoch(); role != RoleReplica {
		t.Fatalf("backup role = %v, want replica", role)
	}
}
