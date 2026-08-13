package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/backends/git, state-store/backends/sqlite, state-store/journal-batching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PromotionCheckpointKind marks the one event every endpoint a promotion
// touches durably records before either endpoint's role/epoch changes. Any
// endpoint that later observes it — directly, via Replicate, or via
// DrainOutbox — learns the new epoch through the exact same idempotent
// journal seam as every other event.
const PromotionCheckpointKind = "authority.promoted"

var (
	// ErrPromotionCandidateNotReplica guards the precondition that only a
	// replica can be the target of a FRESH promotion; promoting an
	// already-active endpoint a second time is never implicit. It is not
	// raised for a resumed promotion whose candidate the earlier, partial
	// call already promoted — see Promote's doc comment.
	ErrPromotionCandidateNotReplica = errors.New("replication: promotion candidate must be a replica")
	// ErrPromotionNotConverged reports a candidate whose cursor lags the
	// active (or the fence checkpoint, in the fence-first sequence). It is
	// distinct from ErrDiverged (same cursor, different checksum chain) and
	// ErrReplicaAhead (candidate cursor beyond active), both of which Promote
	// also refuses on.
	ErrPromotionNotConverged = errors.New("replication: promotion candidate is not converged with the active endpoint")
	// ErrPromotionEmptyJournals reports the one deterministic case Promote
	// refuses outright rather than attempting: an active and candidate that
	// both have no events at all. There is no meaningful "authority" to hand
	// off between two empty journals, and attempting it used to fail deep
	// inside the checkpoint append with a confusing ErrSequenceGap (the
	// checkpoint's epoch is never 1, which the first-event rule requires).
	// Establish initial authority through construction instead — build the
	// intended active with NewDALJournal/NewMemoryJournal's
	// Role: RoleActive, AuthorityEpoch: 1 — never through Promote. See
	// README.md's "Promotion" section for the rationale.
	ErrPromotionEmptyJournals = errors.New("replication: cannot promote between two empty journals; establish initial authority via construction, not Promote")
	// ErrFenceSourceNotActive guards FenceAsReplica's role precondition: only
	// the current active can be fenced. It is also returned when an
	// already-fenced endpoint is asked to fence again for a DIFFERENT
	// promotion (different candidate or idempotency key) than the one it is
	// already fenced for.
	ErrFenceSourceNotActive = errors.New("replication: fence source must be the active endpoint")
	// ErrPromoteTargetNotReplica guards PromoteToActive's role precondition
	// for a target that is neither already active at the checkpoint's epoch
	// (idempotent resume) nor a replica ready to be promoted.
	ErrPromoteTargetNotReplica = errors.New("replication: promotion target must be a replica")
	// ErrPromoteTargetWrongEpoch reports a promotion target that is already
	// active, but at a different epoch than the checkpoint being applied —
	// a genuine conflict, not a resumable retry of the same promotion.
	ErrPromoteTargetWrongEpoch = errors.New("replication: promotion target is active at a different epoch than the checkpoint")
	// ErrPromotionSourceNotActive reports that the active endpoint handed to
	// Promote is neither currently active NOR durably fenced for the exact
	// (candidate, idempotency key) this call names — i.e. it is not in any
	// state Promote can proceed or resume from.
	ErrPromotionSourceNotActive = errors.New("replication: promotion source is neither active nor fenced for this promotion")
)

// promotionCheckpointPayload is the structured payload of every
// PromotionCheckpointKind ("authority.promoted") event. A typed struct
// (rather than an anonymous map) lets both construction
// (buildPromotionCheckpoint) and resumable-fence inspection (ensureFenced)
// share one schema instead of two independently-maintained shapes.
type promotionCheckpointPayload struct {
	FormerActiveEndpointID string `json:"former_active_endpoint_id"`
	NewActiveEndpointID    string `json:"new_active_endpoint_id"`
	Reason                 string `json:"reason,omitempty"`
}

// PromotionRequest names the operator-supplied evidence Promote signs into
// the checkpoint event. ActorID and CommandID identify who/what invoked
// promotion (an operator, or an explicitly configured failover policy per
// state-store/topology's "Promotion is not automatic" rule); IdempotencyKey
// defaults to a deterministic value derived from the active and candidate
// endpoint IDs (not the epoch, which a resumed call cannot know in advance —
// see Promote's doc comment) when left blank, so a retried Promote call for
// the same (active, candidate) transition is itself idempotent rather than
// producing a second checkpoint or a second epoch bump.
type PromotionRequest struct {
	ActorID        string
	CommandID      string
	IdempotencyKey string
	Reason         string
	// ReplicaIDs is the replica set the newly-active candidate should fan
	// future outbox rows out to. Promote passes it to PromoteToActive, which
	// sets it under the same lock that flips the candidate's role — without
	// this, a promoted candidate keeps whatever (likely stale or empty)
	// replica set it was constructed with, silently dropping future
	// deliveries to replicas the operator expects to still be current
	// (including, typically, the just-fenced former active). Nil/empty means
	// "no replicas configured," matching a journal constructed without any.
	ReplicaIDs []string
}

// PromotionResult is the durable outcome of one Promote call.
type PromotionResult struct {
	// Checkpoint is the one event this promotion recorded (or, on a resumed
	// call, the event a prior partial attempt already recorded). Its
	// Cursor.Epoch is the new authority epoch — see the cleanup note this
	// replaces: a separate NewEpoch field would only ever restate
	// Checkpoint.Cursor.Epoch.
	Checkpoint Event
	// FormerActiveEndpointID and NewActiveEndpointID name the two endpoints
	// that changed role during this call.
	FormerActiveEndpointID string
	NewActiveEndpointID    string
	// FencedEndpointIDs lists every endpoint (the former active plus any
	// reachableReplicas that durably accepted the checkpoint) that recorded
	// it during this call, in the order they were fenced. The newly-promoted
	// candidate is intentionally not included here — it is promoted, not
	// fenced.
	FencedEndpointIDs []string
	// ReplicaOutcomes reports, per reachableReplicas entry (excluding the
	// active/candidate), whether checkpoint delivery succeeded. A non-nil Err
	// means that replica's outbox rows remain as pending evidence for a
	// later DrainOutbox/Replicate retry — it does NOT mean the promotion
	// itself failed: the fence/catch-up/promote swap this result reports on
	// already durably succeeded by the time fan-out runs. See Promote's doc
	// comment.
	ReplicaOutcomes []ReplicaDeliveryOutcome
	// Resumed is true when this call detected and continued a
	// fenced-but-not-yet-promoted state a prior, partial Promote call for the
	// exact same (active, candidate, idempotency key) left behind, rather
	// than performing a fresh fence.
	Resumed bool
}

// ReplicaDeliveryOutcome is one reachableReplicas entry's post-swap
// checkpoint-delivery result.
type ReplicaDeliveryOutcome struct {
	EndpointID string
	Err        error
}

// Promote implements the explicit administrative promotion workflow from
// state-store/topology's "Promotion and Recovery" section, using a
// fence-FIRST sequence:
//
//  1. Fence the active first. FenceAsReplica durably records the one
//     checkpoint event as the active's own next event — chaining it onto the
//     active's OWN current head, re-derived fresh inside FenceAsReplica's
//     locked/transactional scope, never from a snapshot Promote read earlier
//     — then downgrades the active to a replica. After this step the former
//     active refuses further domain writes: the system fails toward ZERO
//     writers on any later failure in this call, never two. This closes the
//     four confirmed split-brain defects of the old fence-AFTER-promote
//     order: a concurrent Append during the promotion window can no longer
//     permanently break the fence (Append and FenceAsReplica now hold the
//     same role/epoch lock for their entire duration, so they can never
//     interleave); two concurrent Promote calls for different candidates now
//     serialize on this step's role/epoch guard before either candidate is
//     touched, so at most one can ever win; a transient failure here leaves
//     the active untouched (nothing to roll back — see below); and this
//     step's own re-derived head makes it the single, non-bypassable
//     re-validation point, closing the old TOCTOU where the only check ran
//     before the irreversible candidate flip.
//
//  2. Re-read the candidate's head and catch it up. If the candidate lags
//     the fenced active's new head (which now includes the checkpoint),
//     Promote delivers the remaining events — including the checkpoint
//     itself — via Replicate, sourcing from the now-fenced (but still
//     readable) former active. A fenced endpoint keeps serving Head/After
//     normally; only Append is refused.
//
//  3. Promote the candidate using the SAME checkpoint, now durably present
//     on it via ordinary replication. PromoteToActive only verifies the
//     checkpoint is already the candidate's head (established by step 2) and
//     flips its role — it never appends the checkpoint a second time.
//
//  4. Fan out the checkpoint to any other reachable replicas. A delivery
//     failure here is non-fatal: Promote still reports success for the
//     already-durable swap (see PromotionResult.ReplicaOutcomes) rather than
//     erroring out a promotion that durably succeeded because one lagging
//     backup was unreachable.
//
// # Resumability
//
// A Promote call that fails after step 1 leaves the former active durably
// fenced with zero endpoints currently active — by design, per the
// fail-toward-zero-writers invariant above — and this state MUST be, and is,
// resumable: re-invoking Promote with the SAME active, candidate, and
// (explicit or defaulted) IdempotencyKey detects the durable checkpoint
// FenceAsReplica already left on the active (via HeadEvent, matching Kind,
// IdempotencyKey, and the checkpoint payload's NewActiveEndpointID against
// this call's candidate) and continues from step 2 rather than re-fencing,
// refusing, or double-charging the epoch. PromotionResult.Resumed reports
// which happened. A retry that instead fails transiently INSIDE step 1
// itself (before the durable write commits) leaves the active completely
// untouched — role and epoch unchanged — so a plain retry is simply a fresh
// attempt, not a resume.
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
func Promote(ctx context.Context, active, candidate PromotableJournal, reachableReplicas []PromotableJournal, request PromotionRequest) (PromotionResult, error) {
	if active == nil || candidate == nil {
		return PromotionResult{}, fmt.Errorf("replication: promotion needs an active and a candidate endpoint")
	}
	if strings.TrimSpace(request.ActorID) == "" || strings.TrimSpace(request.CommandID) == "" {
		return PromotionResult{}, fmt.Errorf("replication: promotion request needs an actor id and command id")
	}

	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("promotion:%s:%s", active.EndpointID(), candidate.EndpointID())
	}
	req := request
	req.IdempotencyKey = idempotencyKey

	activeRoleBeforeFence, _ := active.RoleEpoch()
	if activeRoleBeforeFence == RoleActive {
		// Fresh promotion attempt: enforce the preconditions that only make
		// sense before anything has been touched. A resumed call (active
		// already fenced) skips these — see ensureFenced and
		// PromoteToActive's own idempotent-resume handling below.
		candidateRole, _ := candidate.RoleEpoch()
		if candidateRole != RoleReplica {
			return PromotionResult{}, fmt.Errorf("%w: candidate %q has role %q", ErrPromotionCandidateNotReplica, candidate.EndpointID(), candidateRole)
		}
		activeHead, _, err := active.Head(ctx)
		if err != nil {
			return PromotionResult{}, fmt.Errorf("replication: read active %q head: %w", active.EndpointID(), err)
		}
		candidateHeadPre, _, err := candidate.Head(ctx)
		if err != nil {
			return PromotionResult{}, fmt.Errorf("replication: read candidate %q head: %w", candidate.EndpointID(), err)
		}
		if activeHead.IsZero() && candidateHeadPre.IsZero() {
			return PromotionResult{}, fmt.Errorf("%w: active %q and candidate %q both have no events", ErrPromotionEmptyJournals, active.EndpointID(), candidate.EndpointID())
		}
	}

	// --- Step 1: fence the active first (or resume a prior fence). ---
	checkpoint, resumed, err := ensureFenced(ctx, active, candidate.EndpointID(), req)
	if err != nil {
		return PromotionResult{}, err
	}
	result := PromotionResult{
		Checkpoint: checkpoint, Resumed: resumed,
		FormerActiveEndpointID: active.EndpointID(), NewActiveEndpointID: candidate.EndpointID(),
		FencedEndpointIDs: []string{active.EndpointID()},
	}

	// --- Step 2: catch the candidate up to the fenced active's new head. ---
	candidateHead, candidateHash, err := candidate.Head(ctx)
	if err != nil {
		return result, fmt.Errorf("replication: read candidate %q head: %w", candidate.EndpointID(), err)
	}
	if candidateHead != checkpoint.Cursor {
		// Replicate's own ahead/diverged guard (aheadOrDivergedError, shared
		// with evaluateConvergence below) refuses a candidate ahead of the
		// checkpoint before touching it; a candidate merely behind gets every
		// missing event delivered, including the checkpoint itself. A
		// candidate whose existing chain is outright incompatible (not
		// simply behind, but built differently) surfaces here as
		// ErrChecksumChain from the underlying append, once delivery reaches
		// the incompatible event.
		if _, err := Replicate(ctx, active, candidate, candidate.EndpointID()); err != nil {
			return result, fmt.Errorf("replication: catch up promotion candidate %q: %w", candidate.EndpointID(), err)
		}
		candidateHead, candidateHash, err = candidate.Head(ctx)
		if err != nil {
			return result, fmt.Errorf("replication: read candidate %q head after catch-up: %w", candidate.EndpointID(), err)
		}
	}
	// Defensive final check reusing the exact convergence predicate
	// Replicate's own guard is built from. After a successful catch-up above
	// this should always pass; a candidate that started (or ends) exactly at
	// the checkpoint's cursor with a different checksum is still caught here
	// rather than silently promoted.
	if err := evaluateConvergence(candidateHead, candidateHash, checkpoint.Cursor, checkpoint.Checksum); err != nil {
		return result, fmt.Errorf("replication: promotion candidate %q did not converge on checkpoint %v/%s: %w", candidate.EndpointID(), checkpoint.Cursor, checkpoint.Checksum, err)
	}

	// --- Step 3: promote the candidate using the same, now-durable checkpoint. ---
	if err := candidate.PromoteToActive(ctx, checkpoint, request.ReplicaIDs); err != nil {
		return result, fmt.Errorf("replication: promote candidate %q: %w", candidate.EndpointID(), err)
	}

	// --- Step 4: fan out to other reachable replicas; failures are non-fatal. ---
	outcomes := make([]ReplicaDeliveryOutcome, 0, len(reachableReplicas))
	for _, replica := range reachableReplicas {
		if replica == nil {
			continue
		}
		id := replica.EndpointID()
		if id == candidate.EndpointID() || id == active.EndpointID() {
			continue
		}
		if err := replica.IngestReplica(ctx, checkpoint); err != nil {
			outcomes = append(outcomes, ReplicaDeliveryOutcome{EndpointID: id, Err: err})
			continue
		}
		result.FencedEndpointIDs = append(result.FencedEndpointIDs, id)
		outcomes = append(outcomes, ReplicaDeliveryOutcome{EndpointID: id})
	}
	result.ReplicaOutcomes = outcomes
	return result, nil
}

// ensureFenced performs (or resumes) Promote's step 1. If active is
// currently the active endpoint, it fences it fresh. If active is already a
// replica, this can ONLY be a legitimate resume of an earlier, partial
// Promote call for this exact (candidate, idempotency key) — anything else
// (a replica fenced for a different promotion, or one that was never this
// promotion's active in the first place) is refused with
// ErrPromotionSourceNotActive rather than silently reused.
func ensureFenced(ctx context.Context, active PromotableJournal, candidateEndpointID string, request PromotionRequest) (checkpoint Event, resumed bool, err error) {
	role, _ := active.RoleEpoch()
	switch role {
	case RoleActive:
		checkpoint, err := active.FenceAsReplica(ctx, request, candidateEndpointID)
		if err != nil {
			return Event{}, false, fmt.Errorf("replication: fence active %q: %w", active.EndpointID(), err)
		}
		return checkpoint, false, nil
	case RoleReplica:
		head, ok, err := active.HeadEvent(ctx)
		if err != nil {
			return Event{}, false, fmt.Errorf("replication: read fenced active %q head: %w", active.EndpointID(), err)
		}
		if !ok || head.Kind != PromotionCheckpointKind || head.IdempotencyKey != strings.TrimSpace(request.IdempotencyKey) {
			return Event{}, false, fmt.Errorf("%w: %q is a replica, not fenced for this promotion", ErrPromotionSourceNotActive, active.EndpointID())
		}
		var payload promotionCheckpointPayload
		if err := json.Unmarshal(head.Payload, &payload); err != nil {
			return Event{}, false, fmt.Errorf("replication: decode fenced checkpoint payload: %w", err)
		}
		if payload.NewActiveEndpointID != candidateEndpointID {
			return Event{}, false, fmt.Errorf("%w: %q is fenced for a different candidate %q", ErrPromotionSourceNotActive, active.EndpointID(), payload.NewActiveEndpointID)
		}
		return head, true, nil
	default:
		return Event{}, false, fmt.Errorf("%w: %q has unrecognized role %q", ErrPromotionSourceNotActive, active.EndpointID(), role)
	}
}

// buildPromotionCheckpoint constructs (but does not append) the one
// checkpoint event a fence-first promotion uses on every endpoint it
// touches. previousHash and newEpoch MUST be read fresh inside the fencing
// endpoint's own locked/transactional scope (see DALJournal.FenceAsReplica
// and MemoryJournal.FenceAsReplica) — never from a snapshot read before that
// lock, which is exactly the TOCTOU the fence-first redesign exists to
// close. The candidate and any other replica receive the identical event,
// byte-for-byte, through the ordinary replication seam.
func buildPromotionCheckpoint(projectID, formerActiveEndpointID, newActiveEndpointID string, request PromotionRequest, previousHash string, newEpoch int64) (Event, error) {
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("promotion:%s:%s", formerActiveEndpointID, newActiveEndpointID)
	}
	payload, err := json.Marshal(promotionCheckpointPayload{
		FormerActiveEndpointID: formerActiveEndpointID,
		NewActiveEndpointID:    newActiveEndpointID,
		Reason:                 request.Reason,
	})
	if err != nil {
		return Event{}, fmt.Errorf("replication: encode promotion checkpoint payload: %w", err)
	}
	return NewEvent(Event{
		ProjectID:      projectID,
		EventID:        idempotencyKey,
		Cursor:         Cursor{Epoch: newEpoch, Sequence: 1},
		OccurredAt:     time.Now().UTC(),
		ActorID:        request.ActorID,
		CommandID:      request.CommandID,
		IdempotencyKey: idempotencyKey,
		Kind:           PromotionCheckpointKind,
		PreviousHash:   previousHash,
		Payload:        payload,
	})
}

// evaluateConvergence refuses a candidate that is behind, ahead of, or
// diverged from the active — the "candidate replica is not converged
// (lag > 0 / diverged)" refusal. It is a pure function over cursors/hashes so
// every case (behind, ahead, diverged-at-equal-cursor, converged) can be unit
// tested without constructing real journals. It is no longer called from
// Promote itself (the fence-first sequence's own cursor/checksum comparison
// in step 2 supersedes it — a candidate must match the checkpoint exactly,
// not merely "the active" at some snapshot in time), but stays exported
// behavior/tested in its own right as the general-purpose convergence
// predicate the doc comment describes.
func evaluateConvergence(candidateHead Cursor, candidateHash string, activeHead Cursor, activeHash string) error {
	if err := aheadOrDivergedError(candidateHead, candidateHash, activeHead, activeHash); err != nil {
		return err
	}
	if compareCursor(candidateHead, activeHead) < 0 {
		return fmt.Errorf("%w: candidate %v lags active %v", ErrPromotionNotConverged, candidateHead, activeHead)
	}
	return nil
}

// FenceAsReplica is DALJournal's implementation of PromotableJournal's
// fence-first seam: see that interface's doc comment and Promote's doc
// comment for the full contract. Holding j.mu for the role check, the
// flush-then-fence batch drain, the fresh head read, the append, AND the
// role/epoch flip is what makes this atomic with respect to a concurrent
// Append (see the Append/IngestReplica doc comment above). If batching is
// enabled, the pending batch is flushed FIRST -- before the fresh head read,
// before the checkpoint is built, and before it is appended -- so every
// event legitimately enqueued before this call acquired j.mu durably
// commits at the OLD epoch, and the checkpoint's PreviousHash chains onto a
// head that already reflects them. See appendBatcher's "Promotion interplay
// (flush-then-fence)" doc comment in batch.go and this package's README
// "Batching" section.
func (j *DALJournal) FenceAsReplica(ctx context.Context, request PromotionRequest, candidateEndpointID string) (Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.role != RoleActive {
		return Event{}, fmt.Errorf("%w: %q has role %q, not active", ErrFenceSourceNotActive, j.endpointID, j.role)
	}
	if j.batcher != nil {
		// A failed pre-fence flush already delivered its error to each
		// blocked Append's own done channel; whether to also abort the
		// fence itself on a failed drain is a separate design question, out
		// of scope for this batcher-level fix (state-store/journal-batching#ac:close-flushes-pending-batch's
		// Close-error-propagation), so this preserves the existing
		// fence-proceeds-regardless behavior.
		_ = j.batcher.flush(ctx, nil)
	}
	_, hash, err := loadHead(ctx, j.db, j.projectID)
	if err != nil {
		return Event{}, err
	}
	checkpoint, err := buildPromotionCheckpoint(j.projectID, j.endpointID, candidateEndpointID, request, hash, j.epoch+1)
	if err != nil {
		return Event{}, err
	}
	if err := j.append(ctx, checkpoint); err != nil {
		return Event{}, err
	}
	j.role, j.epoch = RoleReplica, checkpoint.Cursor.Epoch
	return checkpoint, nil
}

// PromoteToActive is DALJournal's implementation of PromotableJournal's
// fence-first seam: see that interface's doc comment and Promote's doc
// comment for the full contract. It never appends checkpoint itself — by the
// time Promote calls this, ordinary replication already delivered it — it
// only verifies that checkpoint is this endpoint's current durable head (or
// that this endpoint is already active at the checkpoint's epoch, the
// idempotent-resume case) and flips role/epoch/replicaIDs together under the
// same lock Append/IngestReplica respect.
func (j *DALJournal) PromoteToActive(ctx context.Context, checkpoint Event, replicaIDs []string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.role == RoleActive {
		if j.epoch != checkpoint.Cursor.Epoch {
			return fmt.Errorf("%w: %q is already active at epoch %d, checkpoint is epoch %d", ErrPromoteTargetWrongEpoch, j.endpointID, j.epoch, checkpoint.Cursor.Epoch)
		}
		// Idempotent resume no-op: already promoted for this exact checkpoint.
	} else {
		if j.role != RoleReplica {
			return fmt.Errorf("%w: %q has role %q", ErrPromoteTargetNotReplica, j.endpointID, j.role)
		}
		head, hash, err := loadHead(ctx, j.db, j.projectID)
		if err != nil {
			return err
		}
		if head != checkpoint.Cursor || hash != checkpoint.Checksum {
			return fmt.Errorf("%w: %q head %v/%s does not match checkpoint %v/%s; catch it up before promoting", ErrPromotionNotConverged, j.endpointID, head, hash, checkpoint.Cursor, checkpoint.Checksum)
		}
		j.role, j.epoch = RoleActive, checkpoint.Cursor.Epoch
	}
	ids, err := validatedReplicaIDs(replicaIDs)
	if err != nil {
		return err
	}
	j.replicaIDs = ids
	return nil
}
