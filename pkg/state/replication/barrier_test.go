package replication

// Features implemented: state-store/topology, state-store/backends/sqlite
// Features depended on:  state-store/topology, state-store/backends/git

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fastWaitOptions() WaitOptions {
	return WaitOptions{Timeout: 500 * time.Millisecond, PollInterval: 10 * time.Millisecond}
}

func TestWaitRejectsNilReplicaAndZeroCursor(t *testing.T) {
	ctx := context.Background()
	if _, err := Wait(ctx, nil, "mirror", Cursor{Epoch: 1, Sequence: 1}, WaitOptions{}); err == nil {
		t.Fatal("expected an error for a nil replica")
	}
	source, replica := newDrainPair(t, "mirror")
	appendAll(t, source, relayEvents(t))
	if _, err := Wait(ctx, replica, "mirror", Cursor{}, WaitOptions{}); err == nil {
		t.Fatal("expected an error for a zero target cursor")
	}
}

// TestWaitSatisfiedImmediatelyWhenAlreadyAtCursor covers the non-RemoteDurable
// path (a plain replica whose local Head IS its own durability boundary,
// e.g. a future SQLite mirror): reaching the cursor locally is already
// sufficient, RemoteProven stays false, and no polling delay is needed.
func TestWaitSatisfiedImmediatelyWhenAlreadyAtCursor(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	if _, err := DrainOutbox(ctx, source, replica, "mirror"); err != nil {
		t.Fatalf("drain: %v", err)
	}
	target := events[len(events)-1].Cursor

	start := time.Now()
	result, err := Wait(ctx, replica, "mirror", target, fastWaitOptions())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !result.Satisfied || result.RemoteProven || result.Observed != target {
		t.Fatalf("result = %+v, want satisfied/non-remote-proven at target", result)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Wait took %s for an already-satisfied cursor; want near-immediate", elapsed)
	}
}

// TestWaitPollsUntilReplicaCatchesUp proves the barrier actually blocks a
// lagging replica, then unblocks the instant a concurrent drain delivers the
// requested cursor -- not merely on some later poll tick well after.
func TestWaitPollsUntilReplicaCatchesUp(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	target := events[len(events)-1].Cursor

	type outcome struct {
		result BarrierResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := Wait(ctx, replica, "mirror", target, fastWaitOptions())
		done <- outcome{result, err}
	}()

	select {
	case o := <-done:
		t.Fatalf("Wait returned before the replica caught up: %+v, %v", o.result, o.err)
	case <-time.After(60 * time.Millisecond):
	}

	if _, err := DrainOutbox(ctx, source, replica, "mirror"); err != nil {
		t.Fatalf("drain: %v", err)
	}

	select {
	case o := <-done:
		if o.err != nil || !o.result.Satisfied || o.result.Observed != target {
			t.Fatalf("Wait after catch-up = %+v, %v", o.result, o.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not unblock after the replica caught up")
	}
}

// TestWaitTimesOutWithoutFalseSuccess proves the never-a-false-rollback
// contract from ac:mirror-barrier-proves-git-durability /
// ac:git-barrier-proves-portable-durability: a replica that never catches up
// gets a typed timeout, an unsatisfied result reporting exactly what WAS
// observed, and never a false "durable."
func TestWaitTimesOutWithoutFalseSuccess(t *testing.T) {
	ctx := context.Background()
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	target := events[len(events)-1].Cursor

	result, err := Wait(ctx, replica, "mirror", target, WaitOptions{Timeout: 80 * time.Millisecond, PollInterval: 10 * time.Millisecond})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	var timeoutErr *BarrierTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("err = %v (%T), want *BarrierTimeoutError", err, err)
	}
	if !errors.Is(err, ErrBarrierTimeout) {
		t.Fatalf("errors.Is(err, ErrBarrierTimeout) = false for %v", err)
	}
	if timeoutErr.EndpointID != "mirror" || timeoutErr.Requested != target {
		t.Fatalf("timeout error = %+v, want endpoint=mirror requested=%v", timeoutErr, target)
	}
	if result.Satisfied {
		t.Fatalf("result.Satisfied = true on a timeout, want false: %+v", result)
	}
	if !result.Observed.IsZero() {
		t.Fatalf("result.Observed = %v, want zero (replica never received anything)", result.Observed)
	}
}

// TestWaitContextCancellationPropagatesVerbatim proves Wait distinguishes
// the CALLER's own context ending from its internally configured Timeout
// elapsing: the former must surface as the caller's own ctx.Err(), not get
// repackaged as a *BarrierTimeoutError.
func TestWaitContextCancellationPropagatesVerbatim(t *testing.T) {
	source, replica := newDrainPair(t, "mirror")
	events := relayEvents(t)
	appendAll(t, source, events)
	target := events[len(events)-1].Cursor

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	_, err := Wait(ctx, replica, "mirror", target, WaitOptions{Timeout: 5 * time.Second, PollInterval: 10 * time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	var timeoutErr *BarrierTimeoutError
	if errors.As(err, &timeoutErr) {
		t.Fatalf("caller cancellation was mislabeled as a barrier timeout: %v", err)
	}
}

// TestWaitGitMirrorProvesRemoteDurability is the physical proof behind
// state-store/topology#ac:mirror-barrier-proves-git-durability and
// state-store/backends/sqlite#ac:git-barrier-proves-portable-durability: once
// events are delivered through a real *GitPushJournal (whose IngestReplica/
// Append already durably push-and-verify per event), Wait reports
// RemoteProven with the exact commit SHA the bare origin actually holds.
func TestWaitGitMirrorProvesRemoteDurability(t *testing.T) {
	ctx := context.Background()
	fixture := newGitDurabilityFixtureWithRole(t, RoleActive, "git-active", nil)
	events := relayEvents(t)
	for _, event := range events {
		if _, err := fixture.pushed.AppendAndPush(ctx, event); err != nil {
			t.Fatalf("seed git-active append: %v", err)
		}
	}
	target := events[len(events)-1].Cursor

	result, err := Wait(ctx, fixture.pushed, "git-active", target, fastWaitOptions())
	if err != nil {
		t.Fatalf("Wait against a Git-backed journal: %v", err)
	}
	if !result.Satisfied || !result.RemoteProven {
		t.Fatalf("result = %+v, want satisfied and remote-proven", result)
	}
	remoteHead := gitCommand(t, fixture.bare, "rev-parse", "refs/heads/main")
	if result.CommitSHA != remoteHead {
		t.Fatalf("result.CommitSHA = %q, want the bare origin's actual head %q", result.CommitSHA, remoteHead)
	}
}

// TestWaitGitMirrorRejectsLocalOnlyCommitAsUnproven is the negative case that
// makes the positive test above meaningful: a commit that only exists LOCALLY
// (never pushed) must never be reported as a satisfied barrier, even though
// the local journal's own Head already reflects the target cursor. This is
// the concrete difference between "applied" and "durably recorded on the
// required Git mirror" the AC insists on.
func TestWaitGitMirrorRejectsLocalOnlyCommitAsUnproven(t *testing.T) {
	ctx := context.Background()
	fixture := newGitDurabilityFixtureWithRole(t, RoleActive, "git-active", nil)
	baseline := relayEvents(t)[:2]
	for _, event := range baseline {
		if _, err := fixture.pushed.AppendAndPush(ctx, event); err != nil {
			t.Fatalf("seed git-active append: %v", err)
		}
	}
	remoteHeadBefore := gitCommand(t, fixture.bare, "rev-parse", "refs/heads/main")

	// Commit the remaining events through the RAW journal, bypassing
	// fixture.pushed entirely -- exactly like a bug or a partially-resumed
	// operation could leave local state ahead of what was ever pushed.
	unpushed := relayEvents(t)[2:]
	for _, event := range unpushed {
		if err := fixture.journal.Append(ctx, event); err != nil {
			t.Fatalf("raw local append (bypassing push): %v", err)
		}
	}
	target := unpushed[len(unpushed)-1].Cursor
	if head, _, err := fixture.pushed.Head(ctx); err != nil || head != target {
		t.Fatalf("test setup: local head = %v, %v; want it to already reflect the unpushed target %v", head, err, target)
	}

	// Timeout/PollInterval are generous relative to the other tests in this
	// file: each poll here performs several real `git` subprocess round
	// trips (fetch, rev-parse, merge-base) through RemoteReceipt, which is
	// meaningfully slower than the pure in-memory MemoryJournal cases above.
	result, err := Wait(ctx, fixture.pushed, "git-active", target, WaitOptions{Timeout: 3 * time.Second, PollInterval: 250 * time.Millisecond})
	var timeoutErr *BarrierTimeoutError
	if err == nil || !errors.As(err, &timeoutErr) {
		t.Fatalf("err = %v, want *BarrierTimeoutError (local-only commit must never satisfy the barrier)", err)
	}
	if result.Satisfied || result.RemoteProven {
		t.Fatalf("result = %+v, want unsatisfied/unproven for a commit that was never pushed", result)
	}
	if result.Observed != target {
		t.Fatalf("result.Observed = %v, want %v (Wait must still report exactly what it saw locally)", result.Observed, target)
	}
	remoteHeadAfter := gitCommand(t, fixture.bare, "rev-parse", "refs/heads/main")
	if remoteHeadAfter != remoteHeadBefore {
		t.Fatalf("Wait must never push as a side effect: remote head changed from %s to %s", remoteHeadBefore, remoteHeadAfter)
	}
}
