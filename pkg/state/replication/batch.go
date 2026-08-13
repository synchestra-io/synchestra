package replication

// Features implemented: state-store/journal-batching

import (
	"context"
	"errors"
	"fmt"
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
// Close (or a caller-driven explicit flush, e.g. a promotion's flush-then-
// fence) drains whatever is pending unconditionally. A "generation" counter
// guards (1) and (2) against draining a LATER window than the one they were
// triggered for: both capture the window's generation at the moment their
// event joined it (enqueue's return value; the timer closure's argument) and
// pass it into flush, which re-checks that capture against the current
// generation atomically, under the same lock that would otherwise let it
// drain -- not merely before calling flush, since time.Timer.Stop cannot
// retroactively cancel a callback that has already started running, so a
// stale timer's callback WILL still execute after another trigger has
// drained its window and even started a new one. Close and an explicit
// flush-then-fence flush pass a nil generation to force-drain regardless,
// since they must flush whatever is legitimately pending right now, no
// matter which window it belongs to. This is what keeps the design to one
// monotonic timer per open window rather than polling or restarting a timer
// per item.
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
// crossed the configured item-count threshold, plus the window's generation
// at the moment this event joined it. It never blocks on I/O -- only on
// b.mu, held briefly -- so it is safe (and required, see promotion interplay
// above) for a caller to hold an outer role/epoch lock across this call. The
// caller must release that outer lock before calling flush/wait (both of
// which may block on the actual commit), and must call flush(ctx,
// &generation) itself when flushNow is true -- enqueue only detects the
// threshold, it does not act on it, since acting means a slow commit this
// fast/lock-holding method must not perform. Passing the returned generation
// back into flush (rather than forcing) is what keeps this trigger, like the
// timer, from ever draining a window later than the one this event actually
// joined -- see flush's doc comment.
func (b *appendBatcher) enqueue(event Event) (done chan error, flushNow bool, generation uint64, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, false, 0, ErrJournalClosed
	}
	item := pendingAppend{event: event, done: make(chan error, 1)}
	b.pending = append(b.pending, item)
	generation = b.generation
	if len(b.pending) == 1 && b.settings.MaxDelayMS > 0 {
		armedGeneration := generation
		b.timer = time.AfterFunc(b.settings.delay(), func() { b.onTimerFired(armedGeneration) })
	}
	flushNow = b.settings.MaxItems > 0 && len(b.pending) >= b.settings.MaxItems
	return item.done, flushNow, generation, nil
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
// (see the type doc comment). It delegates straight to flush, passing the
// generation it was armed with: flush is what re-checks that generation
// atomically at drain time, which is the only place this can safely happen
// (see flush's doc comment for why a pre-check here, before calling flush,
// would not be enough). Its own commit outcome, if any, is already fully
// handled by commitAndNotify (delivered to every waiter's done channel), so
// there is nothing further for this fire-and-forget callback to do with
// flush's returned error.
func (b *appendBatcher) onTimerFired(generation uint64) {
	_ = b.flush(context.Background(), &generation)
}

// flush drains whatever is currently pending and commits it, returning the
// shared commit-level error, if the underlying physical transaction itself
// failed (nil on every no-op, and on a transaction that ran to completion --
// even one containing per-event validation failures, which are delivered
// individually through each item's own done channel exactly as always,
// never through this return value). It is safe to call from multiple
// goroutines/triggers concurrently (an item-threshold enqueue, a fired
// timer, Close, and a promotion's flush-then-fence step can all reach
// here).
//
// generation, when non-nil, restricts this call to the window it was
// captured for (by enqueue or the timer closure): if b.generation has
// already moved past it -- another trigger drained that window and
// possibly started a new one -- this call no-ops rather than draining
// whatever happens to be pending in that NEXT window, which it was never
// triggered for. The check runs under b.mu, atomically with the drain
// itself, so there is no gap between "check" and "act" for another trigger
// to race into. generation nil forces an unconditional drain of whatever is
// currently pending, regardless of which window it belongs to -- Close and
// an explicit flush-then-fence flush use this, since both need every
// legitimately pending event flushed right now.
//
// Either way, only the first caller to observe a non-empty, generation-
// matching pending slice performs a commit; every other concurrent caller
// finds it already drained (or not-yet-this-generation's) and no-ops
// (returning nil).
func (b *appendBatcher) flush(ctx context.Context, generation *uint64) error {
	b.mu.Lock()
	if len(b.pending) == 0 {
		b.mu.Unlock()
		return nil
	}
	if generation != nil && *generation != b.generation {
		b.mu.Unlock()
		return nil
	}
	items := b.pending
	b.pending = nil
	b.generation++
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.mu.Unlock()
	return b.commitAndNotify(ctx, items)
}

// commitAndNotify sorts items into cursor order (state-store/journal-batching#ac:batched-events-preserve-fencing-and-order),
// commits them as one call to b.commit (via callCommit, which contains a
// storage-layer panic rather than letting it strand every sibling waiter --
// see callCommit's doc comment), and delivers each item's result. It
// returns the same shared commit-level error it delivers to every item's
// done channel when the whole transaction fails or panics (nil otherwise)
// -- flush propagates this up to Close, so a shutdown caller can tell a
// failed commit apart from a durably-flushed one, exactly like every
// individual waiter already can via its own done channel.
func (b *appendBatcher) commitAndNotify(ctx context.Context, items []pendingAppend) error {
	sort.SliceStable(items, func(i, k int) bool {
		return compareCursor(items[i].event.Cursor, items[k].event.Cursor) < 0
	})
	events := make([]Event, len(items))
	for i, item := range items {
		events[i] = item.event
	}
	results, err := b.callCommit(ctx, events)
	if err != nil {
		for _, item := range items {
			item.done <- err
		}
		return err
	}
	for i, item := range items {
		item.done <- results[i]
	}
	return nil
}

// callCommit invokes b.commit with a recover() around exactly that call --
// before either notification loop in commitAndNotify has sent anything, so
// recovering here can never race a partial delivery -- converting a panic
// from the underlying storage layer into a plain error return instead of
// letting it unwind past this batch's flush. Without this, a panicking
// commit (e.g. a driver bug or a nil-pointer bug several layers down) would
// strand every sibling waiter in the same batch mid-commit: their done
// channels would simply never receive anything, hanging every blocked
// Append/Close forever instead of surfacing a definitive error. The
// recovered panic is treated as an ordinary whole-transaction failure by
// its only caller (commitAndNotify): delivered to every item's done
// channel and returned up through flush, never re-panicked. The batcher's
// own internal state (pending/generation/timer) is already fully reset by
// flush before this call ever runs, so a contained panic leaves the
// batcher itself perfectly usable for the next window.
func (b *appendBatcher) callCommit(ctx context.Context, events []Event) (results []error, err error) {
	defer func() {
		if r := recover(); r != nil {
			results = nil
			err = fmt.Errorf("replication: batch commit panicked: %v", r)
		}
	}()
	return b.commit(ctx, events)
}

// Close flushes any pending batch and durably commits it before returning
// (state-store/journal-batching#ac:close-flushes-pending-batch), then marks
// the batcher closed: every Append after Close returns ErrJournalClosed
// without waiting out any window. Safe to call more than once.
//
// Close returns its flush's commit-level error, if the pending batch's
// transaction itself failed: a shutdown caller reading Close()==nil as
// "everything pending is now durable" would otherwise be deceived by a
// commit that actually failed. Every blocked Append still gets its own
// per-event outcome through its own done channel exactly as before --
// Close's return is the batch-wide signal on top of that, not a
// replacement for it.
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
	return b.flush(ctx, nil)
}
