package replication

// Features implemented: state-store/topology, state-store/topology/offline-fallback, agent-coordination, state-store/journal-batching

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrEpochFenced   = errors.New("replication: authority epoch is fenced")
	ErrSequenceGap   = errors.New("replication: sequence gap")
	ErrChecksumChain = errors.New("replication: checksum chain mismatch")
	ErrFallbackWrite = errors.New("replication: Git fallback accepts communication events only")
	ErrReplicaAhead  = errors.New("replication: replica is ahead of source")
	ErrDiverged      = errors.New("replication: source and replica diverged")
	ErrRoleFenced    = errors.New("replication: endpoint role is fenced from authority writes")
)

// RoleFenceError carries the evidence needed to diagnose a split-brain write
// attempt without granting a replica an authority mutation seam. Both
// DALJournal and MemoryJournal hold their role/epoch lock for the
// precondition check AND the atomic step that makes an Append durably
// observable to a concurrent fence -- an unbatched append's whole
// transaction, or (when group-commit batching is enabled, see batch.go and
// this package's README "Batching" section) the brief enqueue-into-pending
// step, with the actual commit following once the window/threshold triggers
// a flush. A write that races a concurrent promotion always resolves to
// exactly one of: it (or, if batched, its whole pending batch) completes
// before the promotion's fence commits its checkpoint, or it observes the
// new role/epoch and fails here -- it can never fall through to a raw
// checksum-chain or sequence-gap error that would hide the role transition
// that actually explains the failure (see journal.go/dal_journal.go's mu
// doc comments, promotion.go's fence-first design, and appendBatcher's
// "Promotion interplay (flush-then-fence)" doc comment in batch.go).
type RoleFenceError struct {
	EndpointID     string
	Role           Role
	AuthorityEpoch int64
	EventEpoch     int64
}

func (e *RoleFenceError) Error() string {
	return fmt.Sprintf("%v: endpoint=%q role=%q authority_epoch=%d event_epoch=%d", ErrRoleFenced, e.EndpointID, e.Role, e.AuthorityEpoch, e.EventEpoch)
}

func (e *RoleFenceError) Unwrap() error { return ErrRoleFenced }

// Journal is the one persistence seam shared by Git/inGitDB and
// SQLite/DALgo. Append is idempotent by event ID and may only accept a
// contiguous event. The adapter owns durable commit/push evidence; the domain
// never writes a second backend synchronously.
type Journal interface {
	Append(context.Context, Event) error
	After(context.Context, Cursor) ([]Event, error)
	Head(context.Context) (Cursor, string, error)
}

// ReplicaIngestor is the explicit non-authority seam used by Replicate. Domain
// code should depend on Journal; the distinct interface makes replica writes a
// deliberate capability rather than an alternate domain Append path.
type ReplicaIngestor interface {
	IngestReplica(context.Context, Event) error
}

// PromotableJournal is the fencing/promotion seam Promote (promotion.go)
// operates over. It lets Promote work uniformly across *DALJournal (local
// role/epoch plus a DALgo transaction), *GitPushJournal (composes the same
// local write with durable CAS push, so a Git-backed endpoint's checkpoint is
// never stranded locally -- see git_push_journal.go), and *MemoryJournal (a
// fast harness for concurrency/-race tests). It is deliberately not embedded
// in Journal/ReplicaIngestor: ordinary replication code has no business
// fencing or promoting an endpoint.
type PromotableJournal interface {
	Journal
	ReplicaIngestor
	// EndpointID identifies this endpoint for fencing evidence, promotion
	// resume detection, and PromotionResult.
	EndpointID() string
	// RoleEpoch returns a consistent snapshot of this endpoint's local role
	// and authority epoch.
	RoleEpoch() (Role, int64)
	// HeadEvent returns the full latest durable event (not just cursor and
	// hash). ok is false for an empty journal. Promote's resumable-fence
	// detection inspects this to recognize a checkpoint a previous, partial
	// Promote call already left durably in place.
	HeadEvent(ctx context.Context) (Event, bool, error)
	// FenceAsReplica durably records the one promotion checkpoint event as
	// this endpoint's own next event -- chaining it onto THIS endpoint's own
	// current head, computed fresh inside the same locked/transactional scope
	// used to flip role/epoch, never from a snapshot read before that lock --
	// then downgrades this endpoint from active to replica. See
	// promotion.go's package doc comment for the full fence-first sequence
	// and its resumability contract.
	FenceAsReplica(ctx context.Context, request PromotionRequest, candidateEndpointID string) (Event, error)
	// PromoteToActive verifies checkpoint is already this endpoint's current
	// head (Promote's catch-up step establishes that before ever calling
	// this), sets this endpoint's replica set to replicaIDs, and flips this
	// endpoint from replica to active. It never appends the checkpoint
	// itself -- by the time it is called, ordinary replication (Replicate/
	// DrainOutbox/IngestReplica) already delivered it through the normal
	// idempotent seam.
	PromoteToActive(ctx context.Context, checkpoint Event, replicaIDs []string) error
}

// BatchedJournal is implemented by journals that support group-commit
// batching (state-store/journal-batching): today, *DALJournal and
// *MemoryJournal. It is deliberately not embedded in Journal -- a plain
// Journal (e.g. a test double like journalFailer) never needs to expose
// batching internals -- so callers that specifically need to flush on
// shutdown or inspect the effective configuration type-assert for it,
// mirroring how PromotableJournal is kept separate from Journal.
type BatchedJournal interface {
	Journal
	// BatchSettings returns this journal's effective (resolved) group-
	// commit configuration -- see BatchSettings/resolveBatchSettings.
	BatchSettings() BatchSettings
	// Close flushes any pending batch, durably committing it before
	// returning (state-store/journal-batching#ac:close-flushes-pending-batch),
	// then marks the journal closed to further Append calls. A journal
	// constructed with batching disabled (both knobs zero) still
	// implements this trivially -- there is never anything pending to
	// flush. Safe to call more than once. A caller that owns a batched
	// journal's lifecycle (e.g. a one-shot CLI invocation) MUST call
	// Close before process exit for AC close-flushes-pending-batch's
	// "not delayed by the time window" promise to hold in practice --
	// see this package's README "Batching" section.
	Close(ctx context.Context) error
}

// ReplicaHealth makes lag or a failed replication visible rather than silently
// presenting a mirror as fresh.
type ReplicaHealth struct {
	EndpointID string
	Cursor     Cursor
	EventLag   int64
	// LastOK is stamped via time.Now().UTC(), deliberately: Replicate/
	// DrainOutbox/Promote's fan-out run across process and machine
	// boundaries, so a location-independent wall-clock timestamp is what
	// makes health reports comparable/loggable across them. Time.UTC()
	// documented-ly strips the monotonic clock reading a bare time.Now()
	// carries, which is fine here — LastOK is a health-reporting value for
	// display/comparison, never used for in-process Since()-based duration
	// math that would need the monotonic reading.
	LastOK    time.Time
	LastError string
}

// aheadOrDivergedError is the cursor/checksum comparison shared by Replicate
// (which otherwise accepts a downstream that simply lags) and Promote's
// evaluateConvergence (which refuses lag too): a downstream ahead of its
// source is always invalid, and a downstream at the same cursor with a
// different checksum is always a diverged chain. Keeping this one comparison
// in one place means the two call sites can never quietly disagree about what
// "ahead" or "diverged" means.
func aheadOrDivergedError(downstream Cursor, downstreamHash string, source Cursor, sourceHash string) error {
	switch {
	case compareCursor(downstream, source) > 0:
		return fmt.Errorf("%w: %v is ahead of %v", ErrReplicaAhead, downstream, source)
	case compareCursor(downstream, source) == 0 && !downstream.IsZero() && downstreamHash != sourceHash:
		return fmt.Errorf("%w at cursor %v", ErrDiverged, downstream)
	default:
		return nil
	}
}

// Replicate copies all contiguous events after the replica cursor. The source
// stays authoritative; partial delivery remains observable on the replica and
// can resume safely because Append is idempotent.
func Replicate(ctx context.Context, source, replica Journal, endpointID string) (ReplicaHealth, error) {
	if source == nil || replica == nil {
		return ReplicaHealth{EndpointID: endpointID}, fmt.Errorf("replication: replicate needs a source and a replica")
	}
	head, sourceHash, err := source.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, LastError: err.Error()}, err
	}
	cursor, replicaHash, err := replica.Head(ctx)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, LastError: err.Error()}, err
	}
	if err := aheadOrDivergedError(cursor, replicaHash, head, sourceHash); err != nil {
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, LastError: err.Error()}, err
	}
	events, err := source.After(ctx, cursor)
	if err != nil {
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, LastError: err.Error()}, err
	}
	ingestor, ok := replica.(ReplicaIngestor)
	if !ok {
		err := fmt.Errorf("replication: endpoint %q has no replica-ingest seam", endpointID)
		return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, EventLag: int64(len(events)), LastError: err.Error()}, err
	}
	health, _, err := ingestEvents(ctx, ingestor, endpointID, cursor, events)
	return health, err
}

// ingestEvents applies events (already known to be the caller's exact
// contiguous catch-up set) to ingestor one at a time via the same idempotent
// ReplicaIngestor seam, updating cursor/lag as it goes and collecting the
// event IDs it successfully applied before any failure. Replicate and
// DrainOutbox share this one loop so their health bookkeeping can never
// drift apart; DrainOutbox additionally uses the returned applied IDs to
// batch its outbox ack into one call per drain.
func ingestEvents(ctx context.Context, ingestor ReplicaIngestor, endpointID string, cursor Cursor, events []Event) (ReplicaHealth, []string, error) {
	remaining := int64(len(events))
	applied := make([]string, 0, len(events))
	for _, event := range events {
		if err := ingestor.IngestReplica(ctx, event); err != nil {
			return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, EventLag: remaining, LastError: err.Error()}, applied, err
		}
		applied = append(applied, event.EventID)
		cursor = event.Cursor
		remaining--
	}
	return ReplicaHealth{EndpointID: endpointID, Cursor: cursor, EventLag: remaining, LastOK: time.Now().UTC()}, applied, nil
}

func compareCursor(a, b Cursor) int {
	if a.Epoch != b.Epoch {
		if a.Epoch < b.Epoch {
			return -1
		}
		return 1
	}
	if a.Sequence < b.Sequence {
		return -1
	}
	if a.Sequence > b.Sequence {
		return 1
	}
	return 0
}

// validatedReplicaIDs trims, sorts, and rejects blank or duplicate replica
// IDs. It is shared by NewDALJournal/NewMemoryJournal (initial construction)
// and PromoteToActive (establishing the newly-active endpoint's replica set,
// state-store/topology#ac:promotion-fences-former-active's stale-fan-out
// fix) so both paths enforce the exact same topology hygiene.
func validatedReplicaIDs(replicaIDs []string) ([]string, error) {
	ids := append([]string(nil), replicaIDs...)
	sort.Strings(ids)
	for i, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("replication: replica id is required")
		}
		if i > 0 && ids[i-1] == id {
			return nil, fmt.Errorf("replication: duplicate replica id %q", id)
		}
	}
	return ids, nil
}

// MemoryJournal is a strict conformance harness implementation. It is not a
// production backend: production Git and SQLite adapters must satisfy the
// Journal contract using inGitDB/DALgo respectively. It mirrors DALJournal's
// project scoping, role/epoch fencing (including fence-first promotion),
// outbox validation, and group-commit batching so fast in-memory tests --
// including -race concurrency tests that would be impractical against a
// real DB/Git backend -- exercise the same invariants the physical adapters
// do (state-store/journal-batching's "MemoryJournal gets the same batching
// semantics" parity requirement).
type MemoryJournal struct {
	// mu guards role/epoch, mirroring DALJournal.mu -- see its doc comment
	// and RoleFenceError's. It is deliberately separate from dataMu (below):
	// FenceAsReplica/PromoteToActive hold mu for the whole checkpoint-then-
	// transition sequence, including a nested dataMu-guarded flush/append,
	// which would deadlock if mu and dataMu were the same lock.
	mu         sync.RWMutex
	role       Role
	epoch      int64
	projectID  string
	endpointID string

	// dataMu guards the actual journal state below, mirroring the
	// serialization a real DB transaction provides for DALJournal.
	// appendData/commitBatch/PendingOutbox/AckOutbox/After/Head/HeadEvent
	// all acquire this directly; none of them touch mu, so a batch flush
	// invoked from inside FenceAsReplica (which already holds mu) never
	// re-enters it.
	dataMu     sync.Mutex
	events     []Event
	byID       map[string]Event
	byKey      map[string]string
	head       Cursor
	headHash   string
	replicaIDs []string
	// outbox mirrors DALJournal's durable per-replica outbox rows in memory:
	// one pending entry per (replicaID, eventID) written by the same append
	// that commits the event, removed only by AckOutbox. It exists so
	// DrainOutbox has a fast, deterministic OutboxSource for conformance and
	// concurrency tests that do not need a real SQLite/Git backend.
	outbox map[string]map[string]Event

	batchSettings BatchSettings
	batcher       *appendBatcher
}

var (
	_ OutboxSource      = (*MemoryJournal)(nil)
	_ OutboxSource      = (*DALJournal)(nil)
	_ PromotableJournal = (*MemoryJournal)(nil)
	_ PromotableJournal = (*DALJournal)(nil)
	_ PromotableJournal = (*GitPushJournal)(nil)
	_ BatchedJournal    = (*MemoryJournal)(nil)
	_ BatchedJournal    = (*DALJournal)(nil)
)

// MemoryJournalOptions configures a harness journal with the same required
// fields DALJournalOptions requires, so both types validate identically.
type MemoryJournalOptions struct {
	ProjectID      string
	EndpointID     string
	Role           Role
	AuthorityEpoch int64
	ReplicaIDs     []string
	// MaxBatchItems and MaxBatchDelayMS configure group-commit batching
	// (state-store/journal-batching). Nil means "use the documented
	// default" (100 items / 1000ms); an explicit pointer to 0 disables
	// that dimension; both explicitly 0 restores immediate per-event
	// commits, byte-identical to the pre-batching journal. See
	// BatchSettings/resolveBatchSettings and this package's README.
	MaxBatchItems   *int
	MaxBatchDelayMS *int
}

// NewMemoryJournal builds a harness journal, validating exactly the fields
// NewDALJournal validates (non-blank project/endpoint ID, a real role, a
// positive authority epoch, and non-blank/non-duplicate replica IDs) so a
// test cannot silently construct a journal DALJournal would have refused.
func NewMemoryJournal(options MemoryJournalOptions) (*MemoryJournal, error) {
	if strings.TrimSpace(options.ProjectID) == "" {
		return nil, fmt.Errorf("replication: memory journal project id is required")
	}
	if strings.TrimSpace(options.EndpointID) == "" {
		return nil, fmt.Errorf("replication: memory journal endpoint id is required")
	}
	if options.Role != RoleActive && options.Role != RoleReplica {
		return nil, fmt.Errorf("replication: memory journal endpoint %q has invalid role %q", options.EndpointID, options.Role)
	}
	if options.AuthorityEpoch < 1 {
		return nil, fmt.Errorf("replication: memory journal authority epoch must be positive")
	}
	ids, err := validatedReplicaIDs(options.ReplicaIDs)
	if err != nil {
		return nil, err
	}
	outbox := make(map[string]map[string]Event, len(ids))
	for _, id := range ids {
		outbox[id] = make(map[string]Event)
	}
	j := &MemoryJournal{
		projectID: options.ProjectID, endpointID: options.EndpointID, role: options.Role, epoch: options.AuthorityEpoch,
		byID: make(map[string]Event), byKey: make(map[string]string), replicaIDs: ids, outbox: outbox,
		batchSettings: resolveBatchSettings(options.MaxBatchItems, options.MaxBatchDelayMS),
	}
	if !j.batchSettings.disabled() {
		j.batcher = newAppendBatcher(j.batchSettings, j.commitBatch)
	}
	return j, nil
}

func (j *MemoryJournal) EndpointID() string { return j.endpointID }

// BatchSettings returns this journal's effective group-commit configuration.
func (j *MemoryJournal) BatchSettings() BatchSettings { return j.batchSettings }

// Close flushes any pending batch (state-store/journal-batching#ac:close-flushes-pending-batch);
// see BatchedJournal's doc comment for the full contract.
func (j *MemoryJournal) Close(ctx context.Context) error {
	if j.batcher == nil {
		return nil
	}
	return j.batcher.Close(ctx)
}

func (j *MemoryJournal) RoleEpoch() (Role, int64) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.role, j.epoch
}

func (j *MemoryJournal) HeadEvent(_ context.Context) (Event, bool, error) {
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	if len(j.events) == 0 {
		return Event{}, false, nil
	}
	return j.events[len(j.events)-1], true, nil
}

// Append enqueues event into the pending batch (or, with batching disabled,
// appends it immediately) and blocks until its batch durably commits (REQ
// group-commit-not-buffering). The role/epoch precondition check and the
// enqueue step both happen while mu is held, so a concurrent FenceAsReplica
// -- which needs mu exclusively -- can never observe this event as "should
// have been rejected" after the fact; see appendBatcher's "Promotion
// interplay" doc comment in batch.go for the full argument.
func (j *MemoryJournal) Append(ctx context.Context, event Event) error {
	j.mu.RLock()
	if j.role != RoleActive || event.Cursor.Epoch != j.epoch {
		j.mu.RUnlock()
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	if j.batcher == nil {
		defer j.mu.RUnlock()
		j.dataMu.Lock()
		defer j.dataMu.Unlock()
		return j.appendData(event)
	}
	done, flushNow, generation, err := j.batcher.enqueue(event)
	j.mu.RUnlock()
	if err != nil {
		return err
	}
	if flushNow {
		// This call's own outcome (including a shared commit-level failure)
		// is delivered below via wait(ctx, done), not this return value.
		_ = j.batcher.flush(ctx, &generation)
	}
	return j.batcher.wait(ctx, done)
}

func (j *MemoryJournal) IngestReplica(_ context.Context, event Event) error {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if j.role != RoleReplica {
		return &RoleFenceError{EndpointID: j.endpointID, Role: j.role, AuthorityEpoch: j.epoch, EventEpoch: event.Cursor.Epoch}
	}
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	return j.appendData(event)
}

// FenceAsReplica is MemoryJournal's parity implementation of the fence-first
// promotion seam: see PromotableJournal's doc comment and promotion.go for
// the full contract. Holding mu for the entire precondition-check,
// flush-then-fence drain, head-read, and append means a concurrent Append/
// IngestReplica can never interleave with a fence the way the pre-redesign
// race allowed. If batching is enabled, the pending batch is flushed here --
// BEFORE the checkpoint is built or appended -- so every event legitimately
// enqueued before this call acquired mu durably commits at the OLD epoch
// first; see appendBatcher's "Promotion interplay (flush-then-fence)" doc
// comment in batch.go.
func (j *MemoryJournal) FenceAsReplica(ctx context.Context, request PromotionRequest, candidateEndpointID string) (Event, error) {
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
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	checkpoint, err := buildPromotionCheckpoint(j.projectID, j.endpointID, candidateEndpointID, request, j.headHash, j.epoch+1)
	if err != nil {
		return Event{}, err
	}
	if err := j.appendData(checkpoint); err != nil {
		return Event{}, err
	}
	j.role, j.epoch = RoleReplica, checkpoint.Cursor.Epoch
	return checkpoint, nil
}

// PromoteToActive is MemoryJournal's parity implementation; see
// PromotableJournal's doc comment for the contract.
func (j *MemoryJournal) PromoteToActive(_ context.Context, checkpoint Event, replicaIDs []string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	if j.role == RoleActive {
		if j.epoch != checkpoint.Cursor.Epoch {
			return fmt.Errorf("%w: %q is already active at epoch %d, checkpoint is epoch %d", ErrPromoteTargetWrongEpoch, j.endpointID, j.epoch, checkpoint.Cursor.Epoch)
		}
		// Idempotent resume no-op: already promoted for this exact checkpoint.
	} else {
		if j.role != RoleReplica {
			return fmt.Errorf("%w: %q has role %q", ErrPromoteTargetNotReplica, j.endpointID, j.role)
		}
		if j.head != checkpoint.Cursor || j.headHash != checkpoint.Checksum {
			return fmt.Errorf("%w: %q head %v/%s does not match checkpoint %v/%s; catch it up before promoting", ErrPromotionNotConverged, j.endpointID, j.head, j.headHash, checkpoint.Cursor, checkpoint.Checksum)
		}
		j.role, j.epoch = RoleActive, checkpoint.Cursor.Epoch
	}
	ids, err := validatedReplicaIDs(replicaIDs)
	if err != nil {
		return err
	}
	j.replicaIDs = ids
	if j.outbox == nil {
		j.outbox = make(map[string]map[string]Event, len(ids))
	}
	for _, id := range ids {
		if _, exists := j.outbox[id]; !exists {
			j.outbox[id] = make(map[string]Event)
		}
	}
	return nil
}

// PendingOutbox returns replicaID's undelivered rows in cursor order. An
// unconfigured replicaID simply has no rows, matching DALJournal's behavior
// of only fanning events out to the replicas it was constructed with.
func (j *MemoryJournal) PendingOutbox(_ context.Context, replicaID string) ([]Event, error) {
	if strings.TrimSpace(replicaID) == "" {
		return nil, fmt.Errorf("replication: outbox replica id is required")
	}
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	pending := j.outbox[replicaID]
	result := make([]Event, 0, len(pending))
	for _, event := range pending {
		result = append(result, event)
	}
	SortEvents(result)
	return result, nil
}

// AckOutbox deletes replicaID's rows for eventIDs. Deleting an absent map key
// is a no-op in Go, so acknowledging an already-gone row (including twice) is
// always safe. Accepting multiple event IDs lets DrainOutbox batch every ack
// from one drain into a single call, mirroring DALJournal's one-transaction
// batched ack.
func (j *MemoryJournal) AckOutbox(_ context.Context, replicaID string, eventIDs ...string) error {
	if strings.TrimSpace(replicaID) == "" {
		return fmt.Errorf("replication: outbox ack requires a replica id")
	}
	for _, eventID := range eventIDs {
		if strings.TrimSpace(eventID) == "" {
			return fmt.Errorf("replication: outbox ack requires a non-blank event id")
		}
	}
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	for _, eventID := range eventIDs {
		delete(j.outbox[replicaID], eventID)
	}
	return nil
}

// commitBatch is MemoryJournal's batchCommitFunc (see batch.go): it applies
// every event in order under one dataMu critical section, exactly mirroring
// DALJournal.commitBatch's one-transaction, per-event-failure-does-not-abort
// semantics -- appendData already advances (or, on failure, leaves
// untouched) the shared j.head/j.headHash exactly the way DALJournal's
// commitOneLocked advances its running (head, hash) pair.
func (j *MemoryJournal) commitBatch(_ context.Context, events []Event) ([]error, error) {
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	results := make([]error, len(events))
	for i, event := range events {
		if err := j.appendData(event); err != nil {
			results[i] = err
		}
	}
	return results, nil
}

// appendData assumes dataMu is already held by the caller (Append,
// IngestReplica, FenceAsReplica, commitBatch).
func (j *MemoryJournal) appendData(event Event) error {
	if err := event.Verify(); err != nil {
		return err
	}
	if event.ProjectID != j.projectID {
		return fmt.Errorf("replication: event project %q does not match journal project %q", event.ProjectID, j.projectID)
	}
	if existing, exists := j.byID[event.EventID]; exists {
		if existing.Checksum != event.Checksum {
			return fmt.Errorf("replication: event id %q reused with different checksum", event.EventID)
		}
		return nil
	}
	if existingID, exists := j.byKey[event.IdempotencyKey]; exists {
		return fmt.Errorf("replication: idempotency key %q already belongs to event %q", event.IdempotencyKey, existingID)
	}
	if j.head.IsZero() {
		if event.Cursor.Epoch != 1 || event.Cursor.Sequence != 1 || event.PreviousHash != "" {
			return fmt.Errorf("%w: first event must be 1/1 with no previous hash", ErrSequenceGap)
		}
	} else {
		if event.Cursor.Epoch < j.head.Epoch {
			return ErrEpochFenced
		}
		if event.Cursor.Epoch == j.head.Epoch && event.Cursor.Sequence != j.head.Sequence+1 {
			return fmt.Errorf("%w: got %d, want %d", ErrSequenceGap, event.Cursor.Sequence, j.head.Sequence+1)
		}
		if event.Cursor.Epoch > j.head.Epoch && (event.Cursor.Epoch != j.head.Epoch+1 || event.Cursor.Sequence != 1) {
			return fmt.Errorf("%w: new epoch %d must start at sequence 1", ErrSequenceGap, event.Cursor.Epoch)
		}
		if event.PreviousHash != j.headHash {
			return ErrChecksumChain
		}
	}
	j.events = append(j.events, event)
	j.byID[event.EventID] = event
	j.byKey[event.IdempotencyKey] = event.EventID
	j.head, j.headHash = event.Cursor, event.Checksum
	for _, replicaID := range j.replicaIDs {
		if j.outbox[replicaID] == nil {
			j.outbox[replicaID] = make(map[string]Event)
		}
		j.outbox[replicaID][event.EventID] = event
	}
	return nil
}

func (j *MemoryJournal) After(_ context.Context, cursor Cursor) ([]Event, error) {
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	result := make([]Event, 0, len(j.events))
	for _, event := range j.events {
		if event.Cursor.Epoch > cursor.Epoch || (event.Cursor.Epoch == cursor.Epoch && event.Cursor.Sequence > cursor.Sequence) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (j *MemoryJournal) Head(_ context.Context) (Cursor, string, error) {
	j.dataMu.Lock()
	defer j.dataMu.Unlock()
	return j.head, j.headHash, nil
}
