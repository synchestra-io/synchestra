package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/backends/git, state-store/backends/sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PromotionCheckpointKind marks the one event both the promoted candidate and
// the former active durably record before either endpoint's role/epoch
// changes. Any endpoint that later observes it — directly, via Replicate, or
// via DrainOutbox — learns the new epoch through the exact same idempotent
// journal seam as every other event.
const PromotionCheckpointKind = "authority.promoted"

var (
	// ErrPromotionCandidateNotReplica guards the precondition that only a
	// replica can be promoted; promoting an already-active endpoint a second
	// time is never implicit.
	ErrPromotionCandidateNotReplica = errors.New("replication: promotion candidate must be a replica")
	// ErrPromotionNotConverged reports a candidate whose cursor lags the
	// active. It is distinct from ErrDiverged (same cursor, different
	// checksum chain) and ErrReplicaAhead (candidate cursor beyond active),
	// both of which Promote also refuses on.
	ErrPromotionNotConverged = errors.New("replication: promotion candidate is not converged with the active endpoint")
)

// PromotionRequest names the operator-supplied evidence Promote signs into
// the checkpoint event. ActorID and CommandID identify who/what invoked
// promotion (an operator, or an explicitly configured failover policy per
// state-store/topology's "Promotion is not automatic" rule); IdempotencyKey
// defaults to a deterministic value derived from the candidate and new epoch
// when left blank, so a retried Promote call for the same transition is
// itself idempotent rather than producing a second checkpoint.
type PromotionRequest struct {
	ActorID        string
	CommandID      string
	IdempotencyKey string
	Reason         string
}

// PromotionResult is the durable outcome of one explicit promotion.
type PromotionResult struct {
	NewEpoch   int64
	Checkpoint Event
	// FencedEndpointIDs lists every endpoint (the former active plus any
	// reachableReplicas) that durably recorded the checkpoint during this
	// call, in the order they were fenced.
	FencedEndpointIDs []string
}

// Promote implements the explicit administrative promotion workflow from
// state-store/topology's "Promotion and Recovery" section: verify the
// candidate replica is caught up with the current active (refusing
// otherwise), sign one promotion checkpoint event at the next authority
// epoch, durably record it on the candidate (which becomes the new active)
// and on the former active (which becomes a fenced replica), and deliver it
// to any other reachable required replicas so every endpoint that observes
// it refuses further writes below the new epoch. It proves
// state-store/topology#ac:promotion-fences-former-active.
//
// Promote deliberately does not drain or catch up the candidate itself — the
// spec's step 2 ("drain and verify the candidate mirror") is the caller's
// job, ordinarily via DrainOutbox or Replicate immediately before calling
// Promote. Promote only verifies convergence and performs the atomic
// checkpoint/role switch; refusing an unconverged candidate is what makes
// that composition safe.
//
// Promote requires a reachable handle on the current active (active must be
// non-nil). In the founder-MVP single-host SQLite topology this is always
// true: the active's physical file is on the same host as the operator
// running promotion even when the server process is down. A genuinely
// unreachable active (a different host that stays partitioned) is not fenced
// by this call; it self-fences the normal way once it reconnects and
// receives the checkpoint through Replicate/DrainOutbox/IngestReplica before
// resuming any authority Append — see validateNext's epoch comparison, which
// this call's durable checkpoint write is what makes trigger correctly.
func Promote(ctx context.Context, active, candidate *DALJournal, reachableReplicas []*DALJournal, request PromotionRequest) (PromotionResult, error) {
	if active == nil || candidate == nil {
		return PromotionResult{}, fmt.Errorf("replication: promotion needs an active and a candidate endpoint")
	}
	if strings.TrimSpace(request.ActorID) == "" || strings.TrimSpace(request.CommandID) == "" {
		return PromotionResult{}, fmt.Errorf("replication: promotion request needs an actor id and command id")
	}
	if active.projectID != candidate.projectID {
		return PromotionResult{}, fmt.Errorf("replication: promotion active project %q does not match candidate project %q", active.projectID, candidate.projectID)
	}
	activeRole, activeEpoch := active.roleEpoch()
	if activeRole != RoleActive {
		return PromotionResult{}, fmt.Errorf("replication: promotion source %q must be the active endpoint, has role %q", active.endpointID, activeRole)
	}
	candidateRole, _ := candidate.roleEpoch()
	if candidateRole != RoleReplica {
		return PromotionResult{}, fmt.Errorf("%w: candidate %q has role %q", ErrPromotionCandidateNotReplica, candidate.endpointID, candidateRole)
	}

	activeHead, activeHash, err := active.Head(ctx)
	if err != nil {
		return PromotionResult{}, err
	}
	candidateHead, candidateHash, err := candidate.Head(ctx)
	if err != nil {
		return PromotionResult{}, err
	}
	if err := evaluateConvergence(candidateHead, candidateHash, activeHead, activeHash); err != nil {
		return PromotionResult{}, err
	}

	newEpoch := activeEpoch + 1
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("promotion:%s:%s:%d", candidate.projectID, candidate.endpointID, newEpoch)
	}
	payload, err := json.Marshal(map[string]any{
		"former_active_endpoint_id": active.endpointID,
		"new_active_endpoint_id":    candidate.endpointID,
		"reason":                    request.Reason,
	})
	if err != nil {
		return PromotionResult{}, fmt.Errorf("replication: encode promotion checkpoint payload: %w", err)
	}
	checkpoint, err := NewEvent(Event{
		ProjectID:      candidate.projectID,
		EventID:        idempotencyKey,
		Cursor:         Cursor{Epoch: newEpoch, Sequence: 1},
		OccurredAt:     time.Now().UTC(),
		ActorID:        request.ActorID,
		CommandID:      request.CommandID,
		IdempotencyKey: idempotencyKey,
		Kind:           PromotionCheckpointKind,
		PreviousHash:   candidateHash,
		Payload:        payload,
	})
	if err != nil {
		return PromotionResult{}, err
	}

	if err := candidate.PromoteToActive(ctx, checkpoint); err != nil {
		return PromotionResult{}, fmt.Errorf("replication: promote candidate %q: %w", candidate.endpointID, err)
	}

	result := PromotionResult{NewEpoch: newEpoch, Checkpoint: checkpoint}
	if err := active.FenceAsReplica(ctx, checkpoint); err != nil {
		return result, fmt.Errorf("replication: fence former active %q: %w", active.endpointID, err)
	}
	result.FencedEndpointIDs = append(result.FencedEndpointIDs, active.endpointID)

	for _, replica := range reachableReplicas {
		if replica == nil || replica == candidate || replica == active {
			continue
		}
		if err := replica.IngestReplica(ctx, checkpoint); err != nil {
			return result, fmt.Errorf("replication: deliver promotion checkpoint to %q: %w", replica.endpointID, err)
		}
		result.FencedEndpointIDs = append(result.FencedEndpointIDs, replica.endpointID)
	}
	return result, nil
}

// evaluateConvergence refuses promotion of a candidate that is behind, ahead
// of, or diverged from the active — the "candidate replica is not converged
// (lag > 0 / diverged)" refusal. It is a pure function over cursors/hashes so
// every case (behind, ahead, diverged-at-equal-cursor, converged) can be unit
// tested without constructing real journals.
func evaluateConvergence(candidateHead Cursor, candidateHash string, activeHead Cursor, activeHash string) error {
	switch {
	case compareCursor(candidateHead, activeHead) > 0:
		return fmt.Errorf("%w: candidate %v is ahead of active %v", ErrReplicaAhead, candidateHead, activeHead)
	case compareCursor(candidateHead, activeHead) < 0:
		return fmt.Errorf("%w: candidate %v lags active %v", ErrPromotionNotConverged, candidateHead, activeHead)
	case !candidateHead.IsZero() && candidateHash != activeHash:
		return fmt.Errorf("%w: at cursor %v", ErrDiverged, candidateHead)
	default:
		return nil
	}
}

// PromoteToActive is the candidate's half of Promote: it durably records the
// checkpoint a caller has already proven convergence for as this endpoint's
// own next event, then flips this endpoint's local role and epoch so its
// exported Append seam accepts new writes. The RoleReplica precondition below
// means an already-active endpoint can never be "promoted" a second time by
// accident; a genuine re-promotion needs a fresh checkpoint built from a new
// Promote call.
func (j *DALJournal) PromoteToActive(ctx context.Context, checkpoint Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.role != RoleReplica {
		return fmt.Errorf("replication: promotion target %q must be a replica, has role %q", j.endpointID, j.role)
	}
	if checkpoint.Cursor.Epoch <= j.epoch {
		return fmt.Errorf("replication: promotion checkpoint epoch %d does not exceed current epoch %d", checkpoint.Cursor.Epoch, j.epoch)
	}
	if err := j.append(ctx, checkpoint); err != nil {
		return err
	}
	j.role = RoleActive
	j.epoch = checkpoint.Cursor.Epoch
	return nil
}

// FenceAsReplica is the former active's half of Promote: it records the same
// signed checkpoint the candidate received — proving both endpoints agree on
// exactly one promotion — then downgrades this endpoint to a replica at the
// new epoch. After this returns, Append refuses on role alone (the very first
// condition in Append's guard); it never has to fall through to
// validateNext's epoch comparison against a freshly loaded head. That is what
// makes a stale-epoch write fail fast for a process that still holds this
// same *DALJournal, and what makes it fail correctly — via validateNext, once
// this durable write is on disk — for a process that reopens the same
// physical store fresh after a restart without yet knowing about promotion.
func (j *DALJournal) FenceAsReplica(ctx context.Context, checkpoint Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if checkpoint.Cursor.Epoch <= j.epoch {
		return fmt.Errorf("replication: fence checkpoint epoch %d does not exceed current epoch %d", checkpoint.Cursor.Epoch, j.epoch)
	}
	if err := j.append(ctx, checkpoint); err != nil {
		return err
	}
	j.role = RoleReplica
	j.epoch = checkpoint.Cursor.Epoch
	return nil
}
