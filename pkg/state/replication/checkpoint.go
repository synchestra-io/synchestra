package replication

// Features implemented: state-store/topology
// Features depended on:  state-store/topology, state-store/backends/git, state-store/backends/sqlite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CheckpointSchemaV1 versions Checkpoint's on-disk/on-wire shape the same way
// EventSchemaV1 versions Event's.
const CheckpointSchemaV1 = "https://synchestra.io/schemas/replication-checkpoint/v1"

var (
	// ErrCheckpointEmpty reports an attempt to checkpoint (or restore from) a
	// journal with no events at all -- mirroring ErrPromotionEmptyJournals'
	// rationale: there is no meaningful point-in-time snapshot of nothing.
	ErrCheckpointEmpty = errors.New("replication: cannot checkpoint an empty journal")
	// ErrCheckpointCursorNotFound reports a requested checkpoint cursor that
	// does not correspond to an actual event in the source journal (neither
	// its head nor any earlier contiguous cursor).
	ErrCheckpointCursorNotFound = errors.New("replication: no event recorded exactly at the requested checkpoint cursor")
	// ErrCheckpointInvalid reports a checkpoint that failed self-verification
	// -- its declared Cursor/ContentHash/EventCount does not match what its
	// own Events actually chain to. Restore refuses to replay an invalid
	// checkpoint rather than silently reconstructing a target from
	// inconsistent data.
	ErrCheckpointInvalid = errors.New("replication: checkpoint failed integrity verification")
	// ErrRestoreTargetNotEmpty guards the topology doctrine that a restore
	// only ever populates a FRESH endpoint. Restoring into a journal that
	// already has events would silently mix two histories; ordinary
	// replication (Replicate/DrainOutbox) is the seam for catching an
	// existing replica up, not Restore.
	ErrRestoreTargetNotEmpty = errors.New("replication: restore target must be an empty journal")
)

// Checkpoint is a durable, portable, point-in-time snapshot of a journal: the
// cursor it was taken at, the deterministic content hash the checksum chain
// already assigns that cursor, and the complete ordered event log a fresh
// endpoint needs to replay to reconstruct the identical domain projection at
// that cursor. It deliberately does NOT carry a separately-serialized domain
// projection: Synchestra is event-sourced (REQ: commit-with-journal), so any
// backend's domain state at Cursor is fully determined by replaying Events in
// order through the same idempotent ReplicaIngestor seam Replicate/
// DrainOutbox already use -- storing a second, independently-derived copy of
// that same fact would only be able to drift from it, never add information.
// ContentHash is not a fresh hash Checkpoint invents: it is the LAST included
// event's own checksum, which the checksum chain (REQ: commit-with-journal)
// already makes a tamper-evident summary of everything at or before it.
type Checkpoint struct {
	Schema      string    `json:"schema"`
	ProjectID   string    `json:"project_id"`
	SourceID    string    `json:"source_endpoint_id"`
	Cursor      Cursor    `json:"cursor"`
	ContentHash string    `json:"content_hash"`
	EventCount  int       `json:"event_count"`
	CreatedAt   time.Time `json:"created_at"`
	Events      []Event   `json:"events"`
}

// NewCheckpoint captures source's state at "at" -- or its current head, when
// at is the zero Cursor -- into a portable Checkpoint. It refuses to
// checkpoint an empty journal (ErrCheckpointEmpty) or a cursor beyond the
// source's current head (there is nothing durable there yet to snapshot).
func NewCheckpoint(ctx context.Context, source Journal, sourceEndpointID string, at Cursor) (Checkpoint, error) {
	if source == nil {
		return Checkpoint{}, fmt.Errorf("replication: checkpoint needs a source journal")
	}
	if strings.TrimSpace(sourceEndpointID) == "" {
		return Checkpoint{}, fmt.Errorf("replication: checkpoint needs a source endpoint id")
	}
	head, headHash, err := source.Head(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	if head.IsZero() {
		return Checkpoint{}, ErrCheckpointEmpty
	}
	target := at
	if target.IsZero() {
		target = head
	}
	if compareCursor(target, head) > 0 {
		return Checkpoint{}, fmt.Errorf("replication: checkpoint cursor %v is ahead of source head %v", target, head)
	}

	all, err := source.After(ctx, Cursor{})
	if err != nil {
		return Checkpoint{}, err
	}
	SortEvents(all)

	events := make([]Event, 0, len(all))
	hash := ""
	for _, event := range all {
		if compareCursor(event.Cursor, target) > 0 {
			break
		}
		events = append(events, event)
		hash = event.Checksum
	}
	if len(events) == 0 || events[len(events)-1].Cursor != target {
		return Checkpoint{}, fmt.Errorf("%w: %v", ErrCheckpointCursorNotFound, target)
	}
	if target == head {
		// Head() is the authoritative source for its own hash; prefer it
		// over the re-derived value in case source's own bookkeeping ever
		// diverges from what After() replays (a defensive symmetry with
		// NewEvent/Verify's own checksum-chain discipline).
		hash = headHash
	}

	return Checkpoint{
		Schema:      CheckpointSchemaV1,
		ProjectID:   events[0].ProjectID,
		SourceID:    sourceEndpointID,
		Cursor:      target,
		ContentHash: hash,
		EventCount:  len(events),
		CreatedAt:   time.Now().UTC(),
		Events:      events,
	}, nil
}

// Verify proves a Checkpoint is internally consistent: every event verifies
// on its own terms (Event.Verify), the events chain together exactly the way
// a live journal's appendData would require (contiguous sequence, unbroken
// PreviousHash chain), and the checkpoint's own declared Cursor/ContentHash/
// EventCount match what that chain actually produces. Rather than
// re-deriving those chaining rules a second time (a second place they could
// drift from appendData's), Verify replays Events into a throwaway
// RoleReplica MemoryJournal through the same IngestReplica seam Restore
// itself uses -- so a checkpoint that would be rejected by a real replica is
// rejected here too, by construction, not by parallel maintenance.
func (c Checkpoint) Verify() error {
	if c.Schema != CheckpointSchemaV1 {
		return fmt.Errorf("%w: unsupported schema %q", ErrCheckpointInvalid, c.Schema)
	}
	if strings.TrimSpace(c.ProjectID) == "" || strings.TrimSpace(c.SourceID) == "" {
		return fmt.Errorf("%w: missing project or source endpoint id", ErrCheckpointInvalid)
	}
	if len(c.Events) == 0 || len(c.Events) != c.EventCount {
		return fmt.Errorf("%w: declared event_count %d does not match %d events", ErrCheckpointInvalid, c.EventCount, len(c.Events))
	}
	zero := 0
	scratch, err := NewMemoryJournal(MemoryJournalOptions{
		ProjectID: c.ProjectID, EndpointID: "checkpoint-verify", Role: RoleReplica,
		AuthorityEpoch: c.Events[0].Cursor.Epoch, MaxBatchItems: &zero, MaxBatchDelayMS: &zero,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCheckpointInvalid, err)
	}
	ctx := context.Background()
	for i, event := range c.Events {
		if event.ProjectID != c.ProjectID {
			return fmt.Errorf("%w: event %d project %q does not match checkpoint project %q", ErrCheckpointInvalid, i, event.ProjectID, c.ProjectID)
		}
		if err := scratch.IngestReplica(ctx, event); err != nil {
			return fmt.Errorf("%w: event %d (%s): %v", ErrCheckpointInvalid, i, event.EventID, err)
		}
	}
	head, hash, err := scratch.Head(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCheckpointInvalid, err)
	}
	if head != c.Cursor || hash != c.ContentHash {
		return fmt.Errorf("%w: reconstructed cursor/hash %v/%s does not match declared %v/%s", ErrCheckpointInvalid, head, hash, c.Cursor, c.ContentHash)
	}
	return nil
}

// Restore replays checkpoint into target, a freshly constructed and
// currently empty replica endpoint. It never accepts a target that already
// has events (ErrRestoreTargetNotEmpty) -- ordinary replication is the seam
// for catching up an existing replica, not this one -- and it structurally
// cannot populate an active endpoint: it writes exclusively through the
// ReplicaIngestor seam, which every RoleActive journal in this package
// refuses with *RoleFenceError. This is how "a restored endpoint joins as a
// replica, never silently as an active" (state-store/topology's Promotion
// and Recovery section) is actually enforced rather than merely documented
// -- a caller that mistakenly constructs target with Role: RoleActive gets a
// fail-closed error on the very first event, not a silent bulk-write onto an
// authority. Restore itself never promotes; pair it with Promote (the same
// fence-first workflow an ordinary caught-up mirror uses) once the restored
// endpoint should become active.
func Restore(ctx context.Context, checkpoint Checkpoint, target Journal, targetEndpointID string) (ReplicaHealth, error) {
	if target == nil {
		return ReplicaHealth{EndpointID: targetEndpointID}, fmt.Errorf("replication: restore needs a target journal")
	}
	if strings.TrimSpace(targetEndpointID) == "" {
		return ReplicaHealth{}, fmt.Errorf("replication: restore needs a target endpoint id")
	}
	if err := checkpoint.Verify(); err != nil {
		return ReplicaHealth{EndpointID: targetEndpointID}, err
	}
	head, _, err := target.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: targetEndpointID}, err
	}
	if !head.IsZero() {
		return ReplicaHealth{EndpointID: targetEndpointID, Cursor: head}, fmt.Errorf("%w: target already has events at %v", ErrRestoreTargetNotEmpty, head)
	}
	ingestor, ok := target.(ReplicaIngestor)
	if !ok {
		err := fmt.Errorf("replication: restore target %q has no replica-ingest seam", targetEndpointID)
		return ReplicaHealth{EndpointID: targetEndpointID}, err
	}
	health, _, err := ingestEvents(ctx, ingestor, targetEndpointID, Cursor{}, checkpoint.Events)
	return health, err
}

// VerifyResult is VerifyConvergence's report: the cursor the comparison was
// made at and each side's checksum there, plus the boolean verdict.
type VerifyResult struct {
	AtCursor   Cursor
	SourceHash string
	TargetHash string
	Equivalent bool
}

// VerifyConvergence is the divergence-detecting "synchestra state verify"
// primitive from state-store/topology's Health section ("`synchestra state
// verify` compares deterministic collection checksums at a selected
// cursor"). It compares source and target's checksums at "at" -- defaulting,
// when at is the zero Cursor, to the LOWER of their two current heads, so two
// endpoints mid-replication (one still catching up) can still be compared at
// a cursor they both actually have, rather than failing outright. It never
// decides ahead/behind lag on its own (that remains Replicate's/Promote's
// aheadOrDivergedError/evaluateConvergence's job); it answers exactly one
// question: at the cursor both sides share, do their checksum chains agree?
// This is the check both the backend-comparison benchmark
// (state-store/topology#ac:backend-comparison-is-equivalent) and a restored
// endpoint's post-restore validation use to confirm real convergence rather
// than merely "the same number of events."
func VerifyConvergence(ctx context.Context, source, target Journal, at Cursor) (VerifyResult, error) {
	if source == nil || target == nil {
		return VerifyResult{}, fmt.Errorf("replication: verify needs a source and a target journal")
	}
	resolvedAt := at
	if resolvedAt.IsZero() {
		sourceHead, _, err := source.Head(ctx)
		if err != nil {
			return VerifyResult{}, err
		}
		targetHead, _, err := target.Head(ctx)
		if err != nil {
			return VerifyResult{}, err
		}
		resolvedAt = sourceHead
		if compareCursor(targetHead, resolvedAt) < 0 {
			resolvedAt = targetHead
		}
		if resolvedAt.IsZero() {
			return VerifyResult{}, fmt.Errorf("%w: both source and target are empty", ErrCheckpointEmpty)
		}
	}
	sourceHash, err := hashAtCursor(ctx, source, resolvedAt)
	if err != nil {
		return VerifyResult{AtCursor: resolvedAt}, fmt.Errorf("replication: verify source: %w", err)
	}
	targetHash, err := hashAtCursor(ctx, target, resolvedAt)
	if err != nil {
		return VerifyResult{AtCursor: resolvedAt}, fmt.Errorf("replication: verify target: %w", err)
	}
	return VerifyResult{AtCursor: resolvedAt, SourceHash: sourceHash, TargetHash: targetHash, Equivalent: sourceHash == targetHash}, nil
}

// hashAtCursor returns journal's checksum exactly at cursor "at": its Head
// hash when at is the current head (the common, cheap case), or the matching
// historical event's own checksum otherwise. It is shared by
// VerifyConvergence's source and target lookups so both sides resolve "the
// hash at a cursor" identically.
func hashAtCursor(ctx context.Context, journal Journal, at Cursor) (string, error) {
	head, headHash, err := journal.Head(ctx)
	if err != nil {
		return "", err
	}
	if at == head {
		return headHash, nil
	}
	if compareCursor(at, head) > 0 {
		return "", fmt.Errorf("%w: cursor %v is ahead of head %v", ErrReplicaAhead, at, head)
	}
	events, err := journal.After(ctx, Cursor{})
	if err != nil {
		return "", err
	}
	SortEvents(events)
	for _, event := range events {
		if event.Cursor == at {
			return event.Checksum, nil
		}
	}
	return "", fmt.Errorf("%w: no event at cursor %v", ErrCheckpointCursorNotFound, at)
}
