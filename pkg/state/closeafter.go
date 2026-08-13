package state

// Features implemented: state-store, state-store/journal-batching

import (
	"context"
	"time"
)

// DefaultCloseAfterGrace is CloseAfter's default grace period. It only needs
// to exceed the in-memory time an Append takes to enqueue into a pending
// group-commit batch (a microsecond-scale mutex operation, independent of
// how long the eventual physical commit — a Git commit, a SQL transaction —
// takes), not the operation's own end-to-end latency, so it can stay short.
const DefaultCloseAfterGrace = 20 * time.Millisecond

// CloseAfter runs fn to completion and returns its result, but does not let
// a SOLE pending Append wait out the underlying journal's full group-commit
// window (state-store/journal-batching) to do it: it waits at most grace for
// fn to finish on its own, and if fn is still running when grace elapses, it
// calls store.Close to force whatever fn has already enqueued to flush now.
//
// fn MUST issue at most one logical Append, AND that Append must be the
// first journal operation fn performs — no precondition read first. Close
// is a one-way, terminal operation (BatchedJournal.Close's doc comment:
// "then marks the journal closed to further Append calls" — there is no
// public "flush but stay open" primitive), so grace only has one chance to
// get this right:
//
//   - A second, sequential Append after the first one returns (e.g.
//     state.WorktreeStore.Claim, which internally does a lease Acquire
//     followed by a worktree.claimed follow-up event) is unsafe: a Close
//     forced while the first Append is still enqueued correctly flushes it,
//     but then permanently fails the SECOND Append with ErrJournalClosed
//     instead of merely delaying it.
//   - A precondition READ before the (only) Append is unsafe too, and more
//     subtly so: a call like Get/List/Correct/Finish/Acknowledge/Transition
//     reads the journal's After() (a full scan of every event, unlike
//     Append's own O(1) Head() precondition) BEFORE it ever reaches its
//     Append — and that scan's duration is unbounded, growing with journal
//     history, not just a microsecond mutex operation. On a real (non-
//     in-memory) journal with enough prior history, this scan can routinely
//     exceed grace even without concurrency or -race involved, so Close
//     fires and closes the journal before fn's Append has even been
//     attempted, permanently failing it the exact same way. This was
//     observed in practice against a real Git-backed journal during task-3's
//     own testing (state-store/journal-batching) — see
//     pkg/cli/agent/run.go's runRunCorrect and message.go's runMessageAck
//     for two call sites this ruled OUT of using CloseAfter, versus
//     runMessageSend/runRunStart (Message().Send/Run().Start, Head()-only
//     preconditions) which it is safe for.
//
// In short: only wrap a domain call in CloseAfter if you have traced it
// down to confirm its ENTIRE execution is exactly one Head()-preconditioned
// Append with no other journal read — e.g. Message().Send, Effort().Create,
// Run().Start, Activity().Record. When in doubt, don't: call the operation
// plainly and pay the batching window, or (for a mutating call) at least
// call Close afterward for shutdown hygiene, matching pkg/cli/agent's
// claim/renew/release/handoff commands.
//
// This is the "a caller that owns a Store's lifecycle — a one-shot CLI
// invocation, in particular — must call Close before process exit for its
// own writes to return promptly" contract (Store.Close's doc comment) made
// safe to use around a SINGLE in-flight operation: calling Close
// concurrently with fn without this grace/select structure risks a race
// where Close marks the journal closed before fn's Append has even
// enqueued, which would fail fn with ErrJournalClosed instead of speeding it
// up. Waiting a short grace first — the same synchronization
// pkg/state/replication/batch_test.go's TestBatchCloseFlushesPendingBatch
// uses to make a Close-unblocks-a-pending-Append proof deterministic — makes
// that race exponentially unlikely in practice while keeping the worst case
// bounded to roughly grace plus one physical commit, instead of the full
// configured window.
//
// fn's own return value/error is always what CloseAfter returns; a failure
// from the forced Close call itself is not silently discarded, but is only
// surfaced by wrapping fn's error when fn also failed — a successful fn
// result is returned as-is even if the best-effort forced Close reported an
// error, since fn's own outcome (delivered through the journal's normal
// per-event acknowledgement) is the authoritative signal for whether the
// write is durable.
//
// CloseAfter calls store.Agent() once, synchronously, before starting fn's
// goroutine or the grace timer. This matters for a backend whose Agent()
// lazily constructs its journal on first use (e.g. gitstore.GitStateStore
// opening the Git-backed database and ensuring its schema) — without this
// warm-up, that one-time construction can itself take longer than grace, so
// the grace timer could fire and force Close before fn ever reaches its
// first Append, permanently fencing fn's write with ErrJournalClosed
// instead of speeding it up. Agent() is cheap to call more than once on an
// already-constructed backend (gitstore caches it), so this adds no
// meaningful cost once warm.
func CloseAfter[T any](ctx context.Context, store Store, grace time.Duration, fn func(context.Context) (T, error)) (T, error) {
	if grace <= 0 {
		grace = DefaultCloseAfterGrace
	}
	_ = store.Agent() // force lazy journal construction before the race below
	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn(ctx)
		done <- result{value: v, err: err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-time.After(grace):
		_ = store.Close(ctx)
		r := <-done
		return r.value, r.err
	}
}
