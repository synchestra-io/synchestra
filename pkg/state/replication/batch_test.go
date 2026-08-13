package replication

// Features implemented: state-store/journal-batching

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

// buildChain constructs n chained, valid events for projectID starting at
// cursor 1/1, each's PreviousHash pointing at the previous event's checksum
// -- a batching test's raw material when it wants full control over which
// events land in the same pending window (as opposed to relayEvents(t)'s
// fixed four-event fair-split fixture).
func buildChain(t *testing.T, projectID string, n int) []Event {
	t.Helper()
	events := make([]Event, n)
	previous := ""
	for i := 0; i < n; i++ {
		events[i] = chainedEvent(t, projectID, fmt.Sprintf("chain-%d", i+1), Cursor{Epoch: 1, Sequence: int64(i + 1)}, previous)
		previous = events[i].Checksum
	}
	return events
}

func intPtr(v int) *int { return &v }

func newMemoryJournalWithBatching(t *testing.T, maxItems, maxDelayMS *int) *MemoryJournal {
	t.Helper()
	journal, err := NewMemoryJournal(MemoryJournalOptions{
		ProjectID: "p", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1,
		MaxBatchItems: maxItems, MaxBatchDelayMS: maxDelayMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	return journal
}

// appendConcurrently launches one goroutine per event, all released
// together, and returns their Append results aligned by index. It is the
// shape every "these N appends land in one batch" test in this file needs:
// launching sequentially would each individually enqueue into (and often
// flush) its own single-item window before the next one even starts.
func appendConcurrently(t *testing.T, journal Journal, ctx context.Context, events []Event) []error {
	t.Helper()
	errs := make([]error, len(events))
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, event := range events {
		wg.Add(1)
		go func(i int, event Event) {
			defer wg.Done()
			<-start
			errs[i] = journal.Append(ctx, event)
		}(i, event)
	}
	close(start)
	wg.Wait()
	return errs
}

// TestBatchFlushesAtItemThreshold is state-store/journal-batching#ac:batch-flushes-at-item-threshold:
// N concurrent Appends against a journal configured with max_items=N (time
// trigger disabled) commit in exactly one storage transaction -- one Git
// commit on the real inGitDB backend -- containing all N events, their
// outbox rows, and a single head advance.
func TestBatchFlushesAtItemThreshold(t *testing.T) {
	ctx := context.Background()
	const maxItems = 5
	root, journal := newGitJournalWithOptions(t, DALJournalOptions{
		ReplicaIDs: []string{"mirror"}, MaxBatchItems: intPtr(maxItems), MaxBatchDelayMS: intPtr(0),
	})
	before := commitCountForTest(t, root)
	events := buildChain(t, "github.com/fair-split/relay", maxItems)

	errs := appendConcurrently(t, journal, ctx, events)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	after := commitCountForTest(t, root)
	if after != before+1 {
		t.Fatalf("commit count = %d, want %d (exactly one Git commit for the whole %d-event batch)", after, before+1, maxItems)
	}
	got, err := journal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != maxItems {
		t.Fatalf("journal has %d events, want %d", len(got), maxItems)
	}
	for _, replicaID := range []string{"mirror"} {
		pending, err := journal.PendingOutbox(ctx, replicaID)
		if err != nil {
			t.Fatal(err)
		}
		if len(pending) != maxItems {
			t.Fatalf("outbox for %q has %d rows, want %d (the same one-transaction batch must fan out every event)", replicaID, len(pending), maxItems)
		}
	}
}

func commitCountForTest(t *testing.T, root string) int {
	t.Helper()
	n, err := strconv.Atoi(gitCommand(t, root, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestBatchFlushesAtTimeThreshold is state-store/journal-batching#ac:batch-flushes-at-time-threshold:
// with the item-count dimension disabled, a lone pending event flushes once
// max_delay_ms has elapsed since it entered the batch, and a second event
// that joins partway through that same window shares its deadline (the "one
// monotonic timer per open window" design -- see batch.go's appendBatcher
// doc comment) rather than restarting its own full window.
func TestBatchFlushesAtTimeThreshold(t *testing.T) {
	ctx := context.Background()
	const delayMS = 120
	journal := newMemoryJournalWithBatching(t, intPtr(0), intPtr(delayMS))
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")

	start := time.Now()
	firstDone := make(chan error, 1)
	go func() { firstDone <- journal.Append(ctx, seed) }()

	time.Sleep(delayMS / 2 * time.Millisecond)
	second := chainedEvent(t, "p", "second", Cursor{1, 2}, seed.Checksum)
	secondDone := make(chan error, 1)
	go func() { secondDone <- journal.Append(ctx, second) }()

	firstErr := <-firstDone
	firstElapsed := time.Since(start)
	secondErr := <-secondDone
	secondElapsed := time.Since(start)
	if firstErr != nil {
		t.Fatalf("first append: %v", firstErr)
	}
	if secondErr != nil {
		t.Fatalf("second append: %v", secondErr)
	}
	if firstElapsed < delayMS*time.Millisecond {
		t.Fatalf("first append returned after %v, want at least the %dms window (no event waits longer than K, but it must not resolve before K either since nothing else can trigger a flush)", firstElapsed, delayMS)
	}
	if firstElapsed > 5*delayMS*time.Millisecond {
		t.Fatalf("first append took %v, suspiciously longer than the %dms window", firstElapsed, delayMS)
	}
	if secondElapsed-firstElapsed > delayMS*time.Millisecond {
		t.Fatalf("second (later) append resolved %v after the first; want it to share the first's single anchored window, not restart its own", secondElapsed-firstElapsed)
	}
	got, err := journal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("journal has %d events, want 2 (both in the one flushed batch)", len(got))
	}
}

// TestZeroDisablesEachBatchDimension is state-store/journal-batching#ac:zero-disables-each-dimension.
func TestZeroDisablesEachBatchDimension(t *testing.T) {
	t.Run("both_zero_is_immediate_and_constructs_no_batcher", func(t *testing.T) {
		journal := newMemoryJournalWithBatching(t, intPtr(0), intPtr(0))
		if journal.batcher != nil {
			t.Fatal("batcher constructed despite both knobs disabled -- the disabled path must bypass batching entirely, not merely behave like it")
		}
		if got := journal.BatchSettings(); got != (BatchSettings{MaxItems: 0, MaxDelayMS: 0}) {
			t.Fatalf("BatchSettings = %+v, want zero", got)
		}
		seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
		start := time.Now()
		if err := journal.Append(context.Background(), seed); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
			t.Fatalf("disabled-batching append took %v, want an immediate per-event commit", elapsed)
		}
		head, _, err := journal.Head(context.Background())
		if err != nil || head != seed.Cursor {
			t.Fatalf("head after immediate append = %+v, %v", head, err)
		}
	})

	t.Run("zero_items_disables_only_the_count_trigger", func(t *testing.T) {
		const delayMS = 60
		journal := newMemoryJournalWithBatching(t, intPtr(0), intPtr(delayMS))
		seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
		start := time.Now()
		done := make(chan error, 1)
		go func() { done <- journal.Append(context.Background(), seed) }()
		select {
		case err := <-done:
			t.Fatalf("append with the item-count dimension disabled returned immediately (err=%v) after %v; a zeroed knob must contribute no flush trigger, so only the time window may resolve this", err, time.Since(start))
		case <-time.After(20 * time.Millisecond):
			// Good: still pending well before the time window.
		}
		if err := <-done; err != nil {
			t.Fatalf("append: %v", err)
		}
		if elapsed := time.Since(start); elapsed < delayMS*time.Millisecond {
			t.Fatalf("append resolved after %v, want at least the %dms time window", elapsed, delayMS)
		}
	})

	t.Run("zero_delay_disables_only_the_time_trigger", func(t *testing.T) {
		const items = 3
		journal := newMemoryJournalWithBatching(t, intPtr(items), intPtr(0))
		seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
		second := chainedEvent(t, "p", "second", Cursor{1, 2}, seed.Checksum)
		third := chainedEvent(t, "p", "third", Cursor{1, 3}, second.Checksum)

		seedDone := make(chan error, 1)
		go func() { seedDone <- journal.Append(context.Background(), seed) }()
		select {
		case err := <-seedDone:
			t.Fatalf("append with the time window disabled returned before reaching the item threshold (err=%v); a zeroed knob must contribute no flush trigger", err)
		case <-time.After(150 * time.Millisecond):
			// Good: still pending well past what any nonzero delay would
			// have flushed at, proving the time dimension is truly inert.
		}
		secondDone := make(chan error, 1)
		thirdDone := make(chan error, 1)
		go func() { secondDone <- journal.Append(context.Background(), second) }()
		go func() { thirdDone <- journal.Append(context.Background(), third) }()
		if err := <-seedDone; err != nil {
			t.Fatalf("first append: %v", err)
		}
		if err := <-secondDone; err != nil {
			t.Fatalf("second append: %v", err)
		}
		if err := <-thirdDone; err != nil {
			t.Fatalf("third append: %v", err)
		}
	})
}

// TestBatchDefaultsAre100ItemsAnd1000ms is state-store/journal-batching#ac:defaults-are-100-items-1000ms.
func TestBatchDefaultsAre100ItemsAnd1000ms(t *testing.T) {
	if DefaultMaxBatchItems != 100 || DefaultMaxBatchDelayMS != 1000 {
		t.Fatalf("documented defaults changed: items=%d delay=%d, want 100/1000", DefaultMaxBatchItems, DefaultMaxBatchDelayMS)
	}
	memJournal, err := NewMemoryJournal(MemoryJournalOptions{ProjectID: "p", EndpointID: "active", Role: RoleActive, AuthorityEpoch: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := memJournal.BatchSettings(); got != (BatchSettings{MaxItems: 100, MaxDelayMS: 1000}) {
		t.Fatalf("MemoryJournal BatchSettings without explicit configuration = %+v, want {100 1000}", got)
	}
	if memJournal.batcher == nil {
		t.Fatal("a journal constructed without explicit batching configuration must batch by default (not bypass the batcher)")
	}

	_, dalJournal := newGitJournalWithOptions(t, DALJournalOptions{})
	if got := dalJournal.BatchSettings(); got != (BatchSettings{MaxItems: 100, MaxDelayMS: 1000}) {
		t.Fatalf("DALJournal BatchSettings without explicit configuration = %+v, want {100 1000}", got)
	}
	if dalJournal.batcher == nil {
		t.Fatal("a DALJournal constructed without explicit batching configuration must batch by default")
	}
}

// TestBatchCommitIsAllOrNothingOnInfrastructureFailure is
// state-store/journal-batching#ac:append-acknowledges-only-durable-commits'
// crash-safety proof: a fault injected partway through a physical batch
// transaction (simulating a crash/storage failure before the batch's one
// commit lands) must leave nothing durable -- not even the events that had
// already been individually validated earlier in the loop -- and every
// event in the batch must report that same failure, never a silent
// success. This is the exact regression the pre-fix commitBatch had: a
// per-event validation failure and an infrastructure write failure were
// treated identically (skip this event, keep going), which let a real
// storage failure be silently absorbed as "one event was invalid" while the
// transaction still committed whatever had written so far.
func TestBatchCommitIsAllOrNothingOnInfrastructureFailure(t *testing.T) {
	ctx := context.Background()
	_, sqliteJournal := newSQLiteJournal(t, nil)
	// failOn=4 lands inside the SECOND event's writes (event1: sets #1-#2,
	// event2: sets #3-#4), after the first event already "succeeded"
	// in-loop -- exactly the scenario that must NOT leave event1 durable.
	failing := failAfterSet{DB: sqliteJournal.db, failOn: 4}
	journal, err := NewDALJournal(&failing, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "sqlite-active", Role: RoleActive, AuthorityEpoch: 1})
	if err != nil {
		t.Fatal(err)
	}
	events := buildChain(t, "github.com/fair-split/relay", 3)

	results, txErr := journal.commitBatch(ctx, events)
	if txErr == nil {
		t.Fatal("injected infrastructure failure unexpectedly left the batch committing successfully")
	}
	for i, r := range results {
		if !errors.Is(r, txErr) {
			t.Fatalf("result[%d] = %v, want the shared infrastructure failure %v -- no event may be silently acknowledged when the physical transaction itself failed", i, r, txErr)
		}
	}
	// Re-read through the ORIGINAL, unwrapped journal over the same
	// underlying database: nothing from the failed batch is durable, not
	// even the first event that validated fine before the injected failure.
	got, err := sqliteJournal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("journal has %d events after an all-or-nothing batch failure, want 0: %#v", len(got), got)
	}
	if head, _, err := sqliteJournal.Head(ctx); err != nil || !head.IsZero() {
		t.Fatalf("journal head after an all-or-nothing batch failure = %+v, %v; want zero (no head advance without a durable commit)", head, err)
	}
}

// TestAppendDoesNotAcknowledgeBeforeItsBatchCommits is the ordering half of
// AC append-acknowledges-only-durable-commits: Append must not return
// (success OR failure) until commit has actually run, proven with a gated
// fake commit function rather than timing heuristics.
func TestAppendDoesNotAcknowledgeBeforeItsBatchCommits(t *testing.T) {
	gate := make(chan struct{})
	commit := func(_ context.Context, events []Event) ([]error, error) {
		<-gate
		return make([]error, len(events)), nil
	}
	batcher := newAppendBatcher(BatchSettings{MaxItems: 1, MaxDelayMS: 0}, commit)
	event := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")

	done, flushNow, err := batcher.enqueue(event)
	if err != nil {
		t.Fatal(err)
	}
	if !flushNow {
		t.Fatal("enqueue crossing MaxItems=1 must report flushNow")
	}
	appendDone := make(chan error, 1)
	go func() {
		batcher.flush(context.Background())
		appendDone <- batcher.wait(context.Background(), done)
	}()
	select {
	case err := <-appendDone:
		t.Fatalf("Append-equivalent wait resolved (err=%v) before the gated commit function returned", err)
	case <-time.After(30 * time.Millisecond):
		// Good: still blocked on the ungated commit.
	}
	close(gate)
	if err := <-appendDone; err != nil {
		t.Fatalf("append: %v", err)
	}
}

// TestBatchCloseFlushesPendingBatch is state-store/journal-batching#ac:close-flushes-pending-batch.
func TestBatchCloseFlushesPendingBatch(t *testing.T) {
	ctx := context.Background()
	// A delay window far longer than any sane test timeout: if Close did
	// not proactively flush, this test would hang instead of merely being
	// slow, making the omission impossible to miss.
	const delayMS = 60_000
	journal := newMemoryJournalWithBatching(t, intPtr(0), intPtr(delayMS))
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- journal.Append(ctx, seed) }()
	// Give the append a moment to actually enqueue before closing.
	time.Sleep(10 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() { closeDone <- journal.Close(ctx) }()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return promptly -- it must flush the pending batch itself rather than waiting out the window")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Close took %v, want well under the %dms configured window (a one-shot short-lived process must not be delayed)", elapsed, delayMS)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("pending append after Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending Append never resolved after Close flushed")
	}
	got, err := journal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != seed.EventID {
		t.Fatalf("journal after Close = %#v, want the one pending event durably committed", got)
	}

	// A second Close is a safe no-op, and an Append after Close is refused
	// immediately rather than silently accepted into a batch nothing will
	// ever flush again.
	if err := journal.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	next := chainedEvent(t, "p", "after-close", Cursor{1, 2}, seed.Checksum)
	if err := journal.Append(ctx, next); !errors.Is(err, ErrJournalClosed) {
		t.Fatalf("append after Close error = %v, want %v", err, ErrJournalClosed)
	}
}

// TestBatchedEventsPreserveFencingAndOrder is
// state-store/journal-batching#ac:batched-events-preserve-fencing-and-order.
// It exercises the realistic race that produces a "one rejected event"
// batch in practice: two events independently computed the SAME next
// cursor (exactly what pkg/state/agentstore's snapshot-then-append CAS
// callers do when they race), landing in one physical batch together with a
// third, properly chained event. Only the loser's slot may fail; the winner
// and the trailing valid event must commit, in cursor order, sharing the
// same outbox fan-out an unbatched sequence of appends would have produced.
//
// This calls journal.commitBatch directly with an explicit, deterministic
// input order rather than racing real goroutines through the public Append
// API: which of two equal-cursor events actually lands first through
// concurrent enqueue is scheduler-dependent (both are equally "valid" from
// the caller's perspective, exactly like two racing agentstore CAS
// callers), so asserting a FIXED winner identity through real concurrency
// would make the test itself flaky. commitBatch's own cursor-stable-sort
// ordering (see appendBatcher.commitAndNotify in batch.go, which every real
// flush -- concurrent or not -- goes through) is what determines the
// winner, and calling it directly exercises exactly that same decision
// deterministically.
func TestBatchedEventsPreserveFencingAndOrder(t *testing.T) {
	ctx := context.Background()
	journal := newMemoryJournalWithBatching(t, intPtr(4), intPtr(0))
	seed := chainedEvent(t, "p", "seed", Cursor{1, 1}, "")
	winner := chainedEvent(t, "p", "winner", Cursor{1, 2}, seed.Checksum)
	loser := chainedEvent(t, "p", "loser", Cursor{1, 2}, seed.Checksum) // same cursor, loses the race
	third := chainedEvent(t, "p", "third", Cursor{1, 3}, winner.Checksum)

	// Input order IS the tie-break for two equal cursors (commitAndNotify's
	// sort.SliceStable), so "winner" (listed before "loser") is guaranteed
	// to be the one that actually claims cursor 1/2 here.
	results, txErr := journal.commitBatch(ctx, []Event{seed, winner, loser, third})
	if txErr != nil {
		t.Fatalf("commitBatch: %v", txErr)
	}
	if results[0] != nil {
		t.Fatalf("seed result: %v", results[0])
	}
	if results[1] != nil {
		t.Fatalf("winner result: %v", results[1])
	}
	if results[2] == nil {
		t.Fatal("loser result unexpectedly nil -- two events claiming the same cursor cannot both commit")
	}
	if !errors.Is(results[2], ErrSequenceGap) {
		t.Fatalf("loser result = %v, want %v (sequence validation, not a raw transaction failure)", results[2], ErrSequenceGap)
	}
	if results[3] != nil {
		t.Fatalf("third result: %v", results[3])
	}

	got, err := journal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("journal has %d events, want 3 (seed, winner, third -- the loser must not appear)", len(got))
	}
	wantOrder := []string{"seed", "winner", "third"}
	for i, want := range wantOrder {
		if got[i].EventID != want {
			t.Fatalf("event %d = %q, want %q (cursor order)", i, got[i].EventID, want)
		}
	}
	for _, e := range got {
		if e.EventID == loser.EventID {
			t.Fatalf("loser event %q is durable; it must have been excluded from the batch's commit", loser.EventID)
		}
	}
}

// TestBatchedAppendRoleEpochFencedAtEnqueueStillLetsOthersCommit covers
// AC batched-events-preserve-fencing-and-order's "...or role/epoch fencing"
// clause: an Append whose event targets a stale epoch is rejected
// synchronously with *RoleFenceError before it ever reaches the pending
// batch, while concurrent, correctly-epoched Appends still land in and
// complete via the same batch.
func TestBatchedAppendRoleEpochFencedAtEnqueueStillLetsOthersCommit(t *testing.T) {
	ctx := context.Background()
	journal := newMemoryJournalWithBatching(t, intPtr(2), intPtr(0))
	valid1 := chainedEvent(t, "p", "valid-1", Cursor{1, 1}, "")
	valid2 := chainedEvent(t, "p", "valid-2", Cursor{1, 2}, valid1.Checksum)
	staleEpoch := chainedEvent(t, "p", "stale-epoch", Cursor{2, 1}, "")

	var wg sync.WaitGroup
	var validErrs [2]error
	var staleErr error
	wg.Add(3)
	go func() { defer wg.Done(); validErrs[0] = journal.Append(ctx, valid1) }()
	go func() { defer wg.Done(); validErrs[1] = journal.Append(ctx, valid2) }()
	go func() { defer wg.Done(); staleErr = journal.Append(ctx, staleEpoch) }()
	wg.Wait()

	var fence *RoleFenceError
	if !errors.As(staleErr, &fence) {
		t.Fatalf("stale-epoch append error = %v, want *RoleFenceError", staleErr)
	}
	if fence.AuthorityEpoch != 1 || fence.EventEpoch != 2 {
		t.Fatalf("fence evidence = %+v, want authority=1 event=2", fence)
	}
	for i, err := range validErrs {
		if err != nil {
			t.Fatalf("valid append %d: %v", i, err)
		}
	}
	got, err := journal.After(ctx, Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("journal has %d events, want 2 (the stale-epoch event must never have entered the batch)", len(got))
	}
}
