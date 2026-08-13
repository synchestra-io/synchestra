// Package agentstore is the backend-neutral implementation of the
// state.AgentStore contract (see pkg/state/agent.go). replication.Journal
// already abstracts Git/inGitDB and SQLite/DALgo (state-store/topology), so
// the effort/run/worktree-claim/message/activity/lease/health domain logic
// above it is written exactly once here; each backend's Store.Agent() method
// supplies a replication.Journal (see Options) and returns a *Store rather
// than reimplementing claim/fence semantics. This keeps every backend's
// Agent() wiring thin and deterministic, per Task 1 of
// spec/plans/synchestra-coordination-foundation.md.
package agentstore

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology, state-store/topology/offline-fallback

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/synchestra-io/synchestra/pkg/state"
	"github.com/synchestra-io/synchestra/pkg/state/replication"
)

// maxAppendRetries bounds how many times a benign sequence race (a
// concurrent writer advanced the shared journal tail between our Head read
// and Append) is retried before giving up.
const maxAppendRetries = 8

// errSequenceRace is an internal sentinel: it never escapes this package.
// Every caller either retries (appendWithRetry) or, for operations with a
// domain uniqueness precondition, re-derives its projection and retries at
// the domain level (see lease.go).
var errSequenceRace = errors.New("agentstore: journal sequence race")

// Store is the backend-neutral Agent() sub-store. See the package doc.
type Store struct {
	journal replication.Journal
	options Options
}

// Options configures a Store instance.
type Options struct {
	// ProjectID scopes every event this Store emits and reads. Required.
	ProjectID string
	// EndpointID identifies this Store instance for health reporting.
	// Defaults to ActorID.
	EndpointID string
	// ActorID is stamped as the acting identity on every emitted event (e.g.
	// "server", "cli:<host>"). Required.
	ActorID string
	// AuthorityEpoch is the epoch this Store instance is authorized to write
	// at. For a Journal backed by replication.DALJournal/GitPushJournal this
	// must match the Journal's own configured epoch, or every write is
	// rejected as fenced (state.ErrLeaseFenced) once the Journal's head
	// reflects a promotion. Required, must be >= 1.
	AuthorityEpoch int64
	// NewID generates a new record/event ID. Defaults to a random 16-byte
	// hex token. Exposed for deterministic tests.
	NewID func() string
	// Now returns the current time, which must be UTC (replication.Event
	// requires it). Defaults to time.Now().UTC(). Exposed for deterministic
	// tests.
	Now func() time.Time
}

// New constructs a Store over an already-configured replication.Journal. The
// Journal owns durability and fencing (role/epoch); New only validates that
// this Store instance is configured to use it correctly.
func New(journal replication.Journal, options Options) (*Store, error) {
	if journal == nil {
		return nil, fmt.Errorf("agentstore: journal is required")
	}
	if options.ProjectID == "" {
		return nil, fmt.Errorf("agentstore: project id is required")
	}
	if options.ActorID == "" {
		return nil, fmt.Errorf("agentstore: actor id is required")
	}
	if options.AuthorityEpoch < 1 {
		return nil, fmt.Errorf("agentstore: authority epoch must be positive")
	}
	if options.EndpointID == "" {
		options.EndpointID = options.ActorID
	}
	if options.NewID == nil {
		options.NewID = newRandomID
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Store{journal: journal, options: options}, nil
}

func newRandomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand.Read only fails if the OS entropy source is broken;
		// panicking here matches the standard library's own guidance for
		// that condition rather than silently returning a weak/empty ID.
		panic(fmt.Sprintf("agentstore: read random id: %v", err))
	}
	return hex.EncodeToString(buf)
}

func (s *Store) Effort() state.EffortStore     { return effortStore{s} }
func (s *Store) Run() state.RunStore           { return runStore{s} }
func (s *Store) Worktree() state.WorktreeStore { return worktreeStore{s} }
func (s *Store) Message() state.MessageStore   { return messageStore{s} }
func (s *Store) Activity() state.ActivityStore { return activityStore{s} }
func (s *Store) Journal() state.JournalStore   { return journalStore{s} }
func (s *Store) Cursor() state.CursorStore     { return cursorStore{s} }
func (s *Store) Lease() state.LeaseStore       { return leaseStore{s} }
func (s *Store) Health() state.HealthStore     { return healthStore{s} }

// Close flushes any pending group-commit batch on the underlying journal
// (state-store/journal-batching#ac:close-flushes-pending-batch) via a
// duck-typed check, mirroring how replication.GitPushJournal.Close passes
// through to its wrapped Journal: agentstore depends only on the
// backend-neutral replication.Journal interface, which does not itself
// declare Close (only BatchedJournal does), so this is how a batching-aware
// journal's Close is reached without agentstore importing
// replication.BatchedJournal directly. A Journal without batching support
// (or with it disabled) has nothing pending, so this is a safe no-op for
// those. See state.AgentStore.Close's doc comment for the "why" (task-3).
func (s *Store) Close(ctx context.Context) error {
	if closer, ok := s.journal.(interface{ Close(context.Context) error }); ok {
		return closer.Close(ctx)
	}
	return nil
}

// Compile-time interface check.
var _ state.AgentStore = (*Store)(nil)

// snapshot pairs a Kind-filtered event replay with the exact journal head it
// was observed alongside: Head is read first, then After, so any event that
// commits between the two reads only ever makes events more complete than
// head suggests, never the reverse (see tryAppendAt). A caller with a domain
// precondition (lease/claim uniqueness, cursor monotonicity) derives its
// decision from the returned events and then appends via tryAppendAt(ctx,
// head, headHash, ...): if the true head has moved at all since this
// snapshot — for any resource, not just the caller's own — the append fails
// and the caller must retry with a fresh snapshot. This is what makes the
// projection-then-append sequence a real compare-and-swap instead of two
// independently-stale reads; see lease.go's Acquire and cursor.go's Advance.
func (s *Store) snapshot(ctx context.Context, kinds ...string) (replication.Cursor, string, []replication.Event, error) {
	head, headHash, err := s.journal.Head(ctx)
	if err != nil {
		return replication.Cursor{}, "", nil, fmt.Errorf("agentstore: read journal head: %w", err)
	}
	events, err := s.loadEvents(ctx, kinds...)
	if err != nil {
		return replication.Cursor{}, "", nil, err
	}
	return head, headHash, events, nil
}

// tryAppendAt builds and appends one event pinned to a specific (head,
// headHash) snapshot, without re-reading the journal. It does not retry.
func (s *Store) tryAppendAt(ctx context.Context, head replication.Cursor, headHash string, kind, correlationID string, payload any) (replication.Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return replication.Event{}, fmt.Errorf("agentstore: encode %s payload: %w", kind, err)
	}
	id := s.options.NewID()
	event, err := replication.NewEvent(replication.Event{
		ProjectID:      s.options.ProjectID,
		EventID:        id,
		Cursor:         replication.Cursor{Epoch: s.options.AuthorityEpoch, Sequence: nextSequence(head, s.options.AuthorityEpoch)},
		OccurredAt:     s.options.Now(),
		ActorID:        s.options.ActorID,
		CommandID:      id,
		IdempotencyKey: id,
		Kind:           kind,
		CorrelationID:  correlationID,
		Payload:        json.RawMessage(encoded),
		PreviousHash:   headHash,
	})
	if err != nil {
		return replication.Event{}, fmt.Errorf("agentstore: build %s event: %w", kind, err)
	}
	if err := s.journal.Append(ctx, event); err != nil {
		return replication.Event{}, translateJournalError(err)
	}
	return event, nil
}

// tryAppend performs one Head-read, build, and Append. It does not retry: a
// caller with a domain precondition (lease/claim uniqueness, cursor
// monotonicity) must use snapshot+tryAppendAt so a lost race re-derives its
// projection from the same point it appends against (see lease.go,
// cursor.go); a caller without one should use appendWithRetry.
func (s *Store) tryAppend(ctx context.Context, kind, correlationID string, payload any) (replication.Event, error) {
	head, headHash, err := s.journal.Head(ctx)
	if err != nil {
		return replication.Event{}, fmt.Errorf("agentstore: read journal head: %w", err)
	}
	return s.tryAppendAt(ctx, head, headHash, kind, correlationID, payload)
}

// appendWithRetry retries tryAppend while the failure is a benign sequence
// race. It must only be used for writes with no domain precondition
// (effort/run/message/activity facts, and the worktree/lease follow-up
// events issued after their own exclusive acquisition already succeeded) —
// a caller that must re-validate a precondition on every retry
// (Lease().Acquire, Cursor().Advance) uses snapshot+tryAppendAt directly.
func (s *Store) appendWithRetry(ctx context.Context, kind, correlationID string, payload any) (replication.Event, error) {
	var lastErr error
	for attempt := 0; attempt < maxAppendRetries; attempt++ {
		event, err := s.tryAppend(ctx, kind, correlationID, payload)
		if err == nil {
			return event, nil
		}
		if !errors.Is(err, errSequenceRace) {
			return replication.Event{}, err
		}
		lastErr = err
	}
	return replication.Event{}, fmt.Errorf("agentstore: append %s exhausted retries: %w", kind, lastErr)
}

// nextSequence picks the sequence this Store's configured epoch should use
// for the next event, given the journal's current head. It intentionally
// does not special-case a stale epoch: constructing Cursor{epoch, 1} for a
// Store whose epoch has been fenced out still fails validation inside
// Journal.Append (translated to state.ErrLeaseFenced), which is the
// behaviour state-store/topology#ac:promotion-fences-former-active requires.
func nextSequence(head replication.Cursor, epoch int64) int64 {
	if head.IsZero() {
		return 1
	}
	if head.Epoch == epoch {
		return head.Sequence + 1
	}
	return 1
}

// translateJournalError maps replication's fencing/race errors onto the
// domain-level vocabulary this package's callers use. Every other error
// (including a genuine idempotency-key collision, which this package avoids
// by minting a fresh key per event) passes through unchanged.
func translateJournalError(err error) error {
	var fenceErr *replication.RoleFenceError
	if errors.As(err, &fenceErr) {
		return fmt.Errorf("agentstore: %w: %v", state.ErrLeaseFenced, err)
	}
	if errors.Is(err, replication.ErrEpochFenced) || errors.Is(err, replication.ErrRoleFenced) {
		return fmt.Errorf("agentstore: %w: %v", state.ErrLeaseFenced, err)
	}
	if errors.Is(err, replication.ErrSequenceGap) || errors.Is(err, replication.ErrChecksumChain) {
		return fmt.Errorf("%w: %v", errSequenceRace, err)
	}
	return err
}

// loadEvents returns every durable event for this Store's project, optionally
// filtered to the given Kinds, in journal order. Every domain projection in
// this package (effort.go, run.go, lease.go, worktree.go, message.go,
// activity.go, cursor.go) is a deterministic fold over this replay — there is
// no cached or incrementally-maintained state, matching the "thin,
// deterministic storage wiring" scope of Task 1: correctness over
// throughput, the same tradeoff pkg/state/gitstore/board.go documents for
// Task().Board().Rebuild().
func (s *Store) loadEvents(ctx context.Context, kinds ...string) ([]replication.Event, error) {
	events, err := s.journal.After(ctx, replication.Cursor{})
	if err != nil {
		return nil, fmt.Errorf("agentstore: load journal: %w", err)
	}
	if len(kinds) == 0 {
		return filterProject(events, s.options.ProjectID), nil
	}
	allowed := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		allowed[kind] = struct{}{}
	}
	filtered := make([]replication.Event, 0, len(events))
	for _, event := range events {
		if event.ProjectID != s.options.ProjectID {
			continue
		}
		if _, ok := allowed[event.Kind]; ok {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func filterProject(events []replication.Event, projectID string) []replication.Event {
	filtered := make([]replication.Event, 0, len(events))
	for _, event := range events {
		if event.ProjectID == projectID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func toStateCursor(c replication.Cursor) state.Cursor {
	return state.Cursor{Epoch: c.Epoch, Sequence: c.Sequence}
}

func toReplicationCursor(c state.Cursor) replication.Cursor {
	return replication.Cursor{Epoch: c.Epoch, Sequence: c.Sequence}
}
