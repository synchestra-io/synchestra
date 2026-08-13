package replication

// Features implemented: state-store/topology, state-store/backends/sqlite
// Features depended on:  state-store/topology, state-store/backends/git

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultWaitTimeout and DefaultWaitPollInterval are Wait's documented
// defaults, used whenever a caller leaves WaitOptions' corresponding field
// at its zero value -- mirroring how DALJournalOptions/MemoryJournalOptions
// document a nonzero default for a left-zero batching knob rather than
// silently treating zero as "no timeout"/"busy-poll".
const (
	DefaultWaitTimeout      = 30 * time.Second
	DefaultWaitPollInterval = 200 * time.Millisecond
)

// ErrBarrierTimeout is the sentinel behind BarrierTimeoutError; a caller that
// only needs the coarse "did the barrier resolve" answer can errors.Is
// against this without importing the typed struct.
var ErrBarrierTimeout = errors.New("replication: mirror barrier timed out before the replica satisfied the requested cursor")

// RemoteDurable is implemented by journals whose local Head cursor is not by
// itself proof of durability -- today, *GitPushJournal, whose durability
// requires a live remote receipt, not merely a local Git commit. Wait
// type-asserts a replica Journal against this interface: when present, Wait
// requires RemoteReceipt to report the target cursor proven before treating
// the barrier as satisfied; when absent (e.g. a bare *DALJournal or a future
// SQLite journal, where the local transaction commit already IS that
// backend's durability boundary — there is no separate remote hop to prove),
// reaching the target cursor locally is already sufficient. Honest
// disclosure of the current contract difference: a Git-backed RemoteDurable
// replica's barrier proves CONTENT ancestry (a live fetch and ancestry
// check), while a non-RemoteDurable replica's barrier proves only that its
// own cursor NUMBER has reached target, with no independent content check.
type RemoteDurable interface {
	// RemoteReceipt reports whether at is durably present on this journal's
	// configured remote right now, and the commit SHA that recorded it when
	// so. It is always a LIVE check -- re-fetching and re-proving ancestry,
	// never trusting a cached receipt file -- so a caller can rely on ok
	// meaning "verified this instant," not "was true at some earlier point."
	// ok is false both when the local journal has not yet reached at
	// (nothing to prove yet) and when it has reached at locally but the
	// remote has not (yet) confirmed it.
	//
	// Caveat, disclosed rather than silently accepted: on *GitPushJournal,
	// this call (via GitRemoteDurability.VerifyLocalHeadDurable) acquires
	// the SAME cross-process operation lock that every mutating call on the
	// endpoint (Append/IngestReplica/fence/promote) serializes through.
	// Wait's polling therefore contends with ordinary replication traffic
	// for that lock -- on a busy mirror, a single RemoteReceipt call can
	// block for as long as an in-flight write holds the lock, which can
	// itself consume up to whatever Timeout budget remains, not merely one
	// PollInterval tick.
	RemoteReceipt(ctx context.Context, at Cursor) (commitSHA string, ok bool, err error)
}

// WaitOptions configures one Wait call. Both fields default when left at
// their zero value -- see resolved().
type WaitOptions struct {
	Timeout      time.Duration
	PollInterval time.Duration
}

func (o WaitOptions) resolved() WaitOptions {
	if o.Timeout <= 0 {
		o.Timeout = DefaultWaitTimeout
	}
	if o.PollInterval <= 0 {
		o.PollInterval = DefaultWaitPollInterval
	}
	if o.PollInterval > o.Timeout {
		o.PollInterval = o.Timeout
	}
	return o
}

// BarrierResult reports what Wait actually observed, independent of whether
// the requested cursor was reached before the deadline -- so a caller (in
// particular the `synchestra state wait` CLI command) can report "the active
// write already succeeded; the mirror just isn't durable yet" rather than
// implying the underlying operation failed. See
// state-store/topology#ac:mirror-barrier-proves-git-durability and
// state-store/backends/sqlite#ac:git-barrier-proves-portable-durability:
// "timeout returns the active success plus an explicit unsatisfied barrier,
// never a false rollback."
type BarrierResult struct {
	EndpointID string
	Requested  Cursor
	// Observed is the replica's own Head cursor at the moment Wait returned
	// -- which may be at, ahead of (a later promotion advanced it further),
	// or behind Requested (a timeout).
	Observed Cursor
	// RemoteProven is true only when the replica implements RemoteDurable
	// AND its RemoteReceipt call confirmed Requested durable on the
	// configured remote. It is always false for a non-RemoteDurable replica,
	// even when Satisfied is true -- local durability there needed no
	// separate remote hop to prove.
	RemoteProven bool
	// CommitSHA is the Git commit that recorded Requested, populated only
	// when RemoteProven is true.
	CommitSHA string
	Satisfied bool
	Elapsed   time.Duration
}

// BarrierTimeoutError carries the same diagnostic fields as RoleFenceError's
// doc comment describes for that type: enough for a caller to explain
// exactly what was requested versus observed, not just that a barrier timed
// out. Cause, when non-nil, is the LAST poll's own probe error -- either one
// that was still in flight when the deadline landed, or one that Wait
// retried (see Wait's doc comment: a probe error with timeout budget
// remaining is treated as transient and retried, never surfaced on its own)
// and never saw a subsequent successful probe before the deadline expired.
type BarrierTimeoutError struct {
	EndpointID string
	Requested  Cursor
	Observed   Cursor
	Elapsed    time.Duration
	Cause      error
}

func (e *BarrierTimeoutError) Error() string {
	msg := fmt.Sprintf("%v: endpoint=%q requested=%v observed=%v elapsed=%s", ErrBarrierTimeout, e.EndpointID, e.Requested, e.Observed, e.Elapsed)
	if e.Cause != nil {
		msg += fmt.Sprintf(" (last probe error: %v)", e.Cause)
	}
	return msg
}

func (e *BarrierTimeoutError) Unwrap() error { return ErrBarrierTimeout }

// Wait implements the mirror barrier from state-store/topology's Vocabulary:
// "a request to wait until selected replicas acknowledge a specified state
// change." It blocks until replica's Head cursor reaches or passes target
// AND (for a RemoteDurable replica, i.e. a Git-backed mirror) a live remote
// check proves target durable there too -- never merely a local commit
// dressed up as durability. On timeout it returns the caller's ctx error (if
// the CALLER's own context was what expired or was canceled) or a typed
// *BarrierTimeoutError (if Wait's own configured Timeout elapsed first)
// alongside the last BarrierResult observed; it never returns a "false
// rollback" — the caller's own write already durably succeeded on the
// active before Wait was ever invoked, and Wait's only job is reporting
// whether replication has since caught up, not re-litigating that write.
//
// A probe error (checkBarrier's Head read or RemoteReceipt's live Git fetch
// returning an error, e.g. a momentary network blip) does NOT abort Wait
// immediately as long as timeout budget remains: Wait treats every probe
// error the same way -- transient -- records it, and keeps polling at the
// configured PollInterval exactly as it would after an ordinary
// not-yet-satisfied result. The configured Timeout, not a single failed
// probe, is what decides the barrier is unsatisfied. This deliberately does
// not attempt to classify probe errors as transient vs. permanent: a
// genuinely broken probe (e.g. a misconfigured remote) simply keeps failing
// until Timeout, which remains the one bound on how long it can spin.
//
// A context deadline landing mid-probe (e.g. during RemoteReceipt's live Git
// fetch, which can take longer than a single poll interval), or expiring
// after one or more retried probe errors with no successful probe in
// between, is normalized into the same *BarrierTimeoutError rather than
// surfaced as that probe's own raw, unclassified error -- the LAST probe
// error is preserved on BarrierTimeoutError.Cause for diagnostics.
func Wait(ctx context.Context, replica Journal, endpointID string, target Cursor, opts WaitOptions) (BarrierResult, error) {
	if replica == nil {
		return BarrierResult{EndpointID: endpointID, Requested: target}, fmt.Errorf("replication: mirror barrier needs a replica journal")
	}
	if target.IsZero() {
		return BarrierResult{EndpointID: endpointID}, fmt.Errorf("replication: mirror barrier needs a non-zero target cursor")
	}
	options := opts.resolved()
	waitCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	start := time.Now()
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	for {
		result, satisfied, err := checkBarrier(waitCtx, replica, endpointID, target, start)
		if err != nil {
			if waitCtx.Err() != nil {
				// waitCtx's deadline (or the caller's own ctx) landed WHILE
				// this probe was in flight, so the probe's own error is just
				// that deadline surfacing through a git subprocess call or
				// similar -- classify it exactly like the select branch
				// below would, instead of returning an unclassified error
				// whose shape depends on which goroutine happened to notice
				// the deadline first.
				return classifyDeadline(ctx, endpointID, target, result, err)
			}
			// Budget remains: treat this probe error as transient rather
			// than fatal. Record it (it resurfaces as
			// BarrierTimeoutError.Cause only if Timeout is later reached
			// without an intervening successful probe) and retry at the
			// next configured poll tick, exactly like an ordinary
			// not-yet-satisfied result would.
			select {
			case <-waitCtx.Done():
				return classifyDeadline(ctx, endpointID, target, result, err)
			case <-ticker.C:
				continue
			}
		}
		if satisfied {
			return result, nil
		}
		select {
		case <-waitCtx.Done():
			return classifyDeadline(ctx, endpointID, target, result, nil)
		case <-ticker.C:
		}
	}
}

// classifyDeadline distinguishes the caller's own ctx ending (propagated
// verbatim) from Wait's internally configured Timeout elapsing (a typed
// *BarrierTimeoutError, optionally carrying the in-flight probe's own error
// as Cause).
func classifyDeadline(ctx context.Context, endpointID string, target Cursor, result BarrierResult, cause error) (BarrierResult, error) {
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, &BarrierTimeoutError{EndpointID: endpointID, Requested: target, Observed: result.Observed, Elapsed: result.Elapsed, Cause: cause}
}

// checkBarrier performs one poll iteration: read the replica's current head,
// and if it has reached target, additionally require a live RemoteDurable
// proof when the replica implements that interface.
func checkBarrier(ctx context.Context, replica Journal, endpointID string, target Cursor, start time.Time) (BarrierResult, bool, error) {
	head, _, err := replica.Head(ctx)
	if err != nil {
		return BarrierResult{EndpointID: endpointID, Requested: target, Elapsed: time.Since(start)}, false, err
	}
	result := BarrierResult{EndpointID: endpointID, Requested: target, Observed: head, Elapsed: time.Since(start)}
	if compareCursor(head, target) < 0 {
		return result, false, nil
	}
	prover, ok := replica.(RemoteDurable)
	if !ok {
		result.Satisfied = true
		return result, true, nil
	}
	commitSHA, proven, err := prover.RemoteReceipt(ctx, target)
	if err != nil {
		return result, false, err
	}
	result.RemoteProven = proven
	result.CommitSHA = commitSHA
	result.Satisfied = proven
	return result, proven, nil
}
