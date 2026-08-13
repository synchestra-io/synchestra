package replication

// Features implemented: state-store/journal-batching

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// DefaultMaxBatchItems and DefaultMaxBatchDelayMS are the documented
// defaults for a journal constructed without explicit batching
// configuration (state-store/journal-batching#ac:defaults-are-100-items-1000ms).
const (
	DefaultMaxBatchItems   = 100
	DefaultMaxBatchDelayMS = 1000
)

// ErrJournalClosed is returned by Append once a journal's Close has run.
// Close's own flush of whatever was already pending still completes and
// acknowledges those callers normally; this error is only seen by an Append
// that starts after Close.
var ErrJournalClosed = errors.New("replication: journal is closed")

// BatchSettings is a journal's effective (resolved) group-commit
// configuration. MaxItems<=0 means the item-count dimension contributes no
// flush trigger; MaxDelayMS<=0 means the time dimension contributes none;
// both<=0 means batching is disabled outright and every Append commits
// immediately in its own transaction, byte-identical to the pre-batching
// journal (state-store/journal-batching#ac:zero-disables-each-dimension).
type BatchSettings struct {
	MaxItems   int
	MaxDelayMS int
}

func (s BatchSettings) disabled() bool { return s.MaxItems <= 0 && s.MaxDelayMS <= 0 }

func (s BatchSettings) delay() time.Duration { return time.Duration(s.MaxDelayMS) * time.Millisecond }

// resolveBatchSettings turns a journal constructor's optional *int knobs
// into an effective BatchSettings: nil means "use the documented default,"
// a non-nil pointer (including one pointing at 0) is used verbatim. This is
// what lets a caller distinguish "not configured" (defaults apply) from
// "explicitly disabled" (0), which a plain int field could not.
func resolveBatchSettings(maxItems, maxDelayMS *int) BatchSettings {
	settings := BatchSettings{MaxItems: DefaultMaxBatchItems, MaxDelayMS: DefaultMaxBatchDelayMS}
	if maxItems != nil {
		settings.MaxItems = *maxItems
	}
	if maxDelayMS != nil {
		settings.MaxDelayMS = *maxDelayMS
	}
	return settings
}

// pendingAppend is one caller's queued event plus the channel its outcome
// (nil on success) is delivered through. done is always buffered (size 1)
// so a caller that stops waiting (e.g. its ctx is cancelled) never blocks
// the batch that eventually resolves it.
type pendingAppend struct {
	event Event
	done  chan error
}

// batchCommitFunc durably commits every event in events, in order, inside
// exactly one physical transaction -- see appendBatcher's doc comment. It
// returns one error per input event (nil = that event committed) when the
// transaction itself succeeds; a non-nil second return means the whole
// transaction failed (an infrastructure failure, not a per-event validation
// failure) and no event in the batch is durable, in which case every
// per-event result the caller reports is that same error.
type batchCommitFunc func(ctx context.Context, events []Event) ([]error, error)

// appendBatcher implements group-commit batching shared by DALJournal and
// MemoryJournal (state-store/journal-batching). Concurrent callers Append
// into a shared pending slice; the batch commits -- once, in a single call
// to commit -- when the configured item count or time window is reached,
// whichever comes first, or when Close/an explicit flush (promotion's
// flush-then-fence, see promotion.go) runs.
//
// # Flush triggers and the one timer per window
//
// Exactly one of three things flushes a given window: (1) the enqueue call
// that makes pending reach MaxItems performs the flush itself, synchronously,
// after releasing b.mu; (2) a single time.AfterFunc timer, armed only once --
// when the first item enters an empty pending slice -- fires and flushes; (3)
// Close (or a caller-driven explicit flush) drains whatever is pending. A
// "generation" counter guards against a stale timer firing after another
// trigger already drained the window it was armed for: flush increments the
// generation, and the timer callback no-ops unless the generation it captured
// at arm-time still matches. This is what keeps the design to one monotonic
// timer per open window rather than polling or restarting a timer per item.
//
// # Durability (REQ group-commit-not-buffering)
//
// Append (the enqueue-and-wait sequence below) never returns before the
// event's batch has durably committed: enqueue only places the event and a
// result channel in pending; the caller then blocks on that channel, which
// commit's caller (flush) only signals after commit returns. There is no
// window in which a caller sees success before the transaction is durable.
//
// # Promotion interplay (flush-then-fence)
//
// FenceAsReplica (promotion.go, both DALJournal and MemoryJournal) calls
// flush while holding the journal's role/epoch lock, BEFORE building or
// appending its own checkpoint. Because that same role/epoch lock is also
// what Append's precondition check (and its subsequent enqueue) holds for
// its own brief critical section, every event that reached this batcher's
// pending slice before the fence acquired the lock is guaranteed to already
// be there by the time the fence's flush call runs -- RWMutex semantics mean
// the fence cannot acquire the exclusive lock until every such
// precondition-check+enqueue has completed and released it. So flush-then-
// fence commits everything that was legitimately in flight at the OLD epoch
// before the checkpoint (next epoch) is ever appended: a batched Append
// racing a fence resolves to exactly the same two outcomes the pre-batching,
// per-event contract promised -- durably committed before the fence, or
// *RoleFenceError from an Append that starts after it -- never a raw
// ErrEpochFenced/ErrChecksumChain that would hide the role transition (see
// RoleFenceError's doc comment in journal.go and this package's README
// "Batching" section).
type appendBatcher struct {
	settings BatchSettings
	commit   batchCommitFunc

	mu         sync.Mutex
	pending    []pendingAppend
	timer      *time.Timer
	generation uint64
	closed     bool
}

// newAppendBatcher constructs a batcher. Callers must not construct one for
// a disabled BatchSettings (settings.disabled()) -- DALJournal/MemoryJournal
// bypass the batcher entirely in that case so the unbatched path stays
// byte-identical to the pre-batching journal.
func newAppendBatcher(settings BatchSettings, commit batchCommitFunc) *appendBatcher {
	return &appendBatcher{settings: settings, commit: commit}
}

// enqueue places event in the pending batch and reports whether this call
// crossed the configured item-count threshold. It never blocks on I/O --
// only on b.mu, held briefly -- so it is safe (and required, see promotion
// interplay above) for a caller to hold an outer role/epoch lock across this
// call. The caller must release that outer lock before calling flush/wait
// (both of which may block on the actual commit), and must call flush
// itself when flushNow is true -- enqueue only detects the threshold, it
// does not act on it, since acting means a slow commit this fast/lock-holding
// method must not perform.
func (b *appendBatcher) enqueue(event Event) (done chan error, flushNow bool, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, false, ErrJournalClosed
	}
	item := pendingAppend{event: event, done: make(chan error, 1)}
	b.pending = append(b.pending, item)
	if len(b.pending) == 1 && b.settings.MaxDelayMS > 0 {
		generation := b.generation
		b.timer = time.AfterFunc(b.settings.delay(), func() { b.onTimerFired(generation) })
	}
	flushNow = b.settings.MaxItems > 0 && len(b.pending) >= b.settings.MaxItems
	return item.done, flushNow, nil
}

// wait blocks until done resolves or ctx is cancelled. A cancelled wait does
// not remove the event from the batch -- it is still committed (or failed)
// normally by whichever trigger flushes the window; the caller here simply
// stops observing the outcome.
func (b *appendBatcher) wait(ctx context.Context, done chan error) error {
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// onTimerFired is the sole callback of the one timer armed per open window
// (see the type doc comment). It no-ops if another trigger already drained
// this window (generation mismatch or empty pending) or the batcher closed.
func (b *appendBatcher) onTimerFired(generation uint64) {
	b.mu.Lock()
	if b.closed || generation != b.generation || len(b.pending) == 0 {
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	b.flush(context.Background())
}

// flush drains whatever is currently pending and commits it. It is safe to
// call from multiple goroutines/triggers concurrently (an item-threshold
// enqueue, a fired timer, Close, and a promotion's flush-then-fence step can
// all reach here); only the first to observe a non-empty pending slice
// performs a commit, every other concurrent caller finds it already drained
// and no-ops. It never returns an error itself -- each event's own result
// (including a shared infrastructure failure, see batchCommitFunc) is
// delivered through its own done channel, which is what every caller
// (Append via wait, or a trigger that does not own a specific event) relies
// on instead.
func (b *appendBatcher) flush(ctx context.Context) {
	b.mu.Lock()
	if len(b.pending) == 0 {
		b.mu.Unlock()
		return
	}
	items := b.pending
	b.pending = nil
	b.generation++
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.mu.Unlock()
	b.commitAndNotify(ctx, items)
}

// commitAndNotify sorts items into cursor order (state-store/journal-batching#ac:batched-events-preserve-fencing-and-order),
// commits them as one call to b.commit, and delivers each item's result.
func (b *appendBatcher) commitAndNotify(ctx context.Context, items []pendingAppend) {
	sort.SliceStable(items, func(i, k int) bool {
		return compareCursor(items[i].event.Cursor, items[k].event.Cursor) < 0
	})
	events := make([]Event, len(items))
	for i, item := range items {
		events[i] = item.event
	}
	results, err := b.commit(ctx, events)
	if err != nil {
		for _, item := range items {
			item.done <- err
		}
		return
	}
	for i, item := range items {
		item.done <- results[i]
	}
}

// Close flushes any pending batch and durably commits it before returning
// (state-store/journal-batching#ac:close-flushes-pending-batch), then marks
// the batcher closed: every Append after Close returns ErrJournalClosed
// without waiting out any window. Safe to call more than once.
func (b *appendBatcher) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.mu.Unlock()
	b.flush(ctx)
	return nil
}
