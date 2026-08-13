package replication

// Features implemented: state-store/backends/git
// Features depended on:  state-store/topology, state-store/journal-batching

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

const (
	gitReceiptSchemaV1       = "https://synchestra.io/schemas/git-delivery-receipt/v1"
	gitOperationJournalV1    = "journal.append.v1"
	gitOperationReplicaV1    = "journal.replica_ingest.v1"
	gitOperationFallbackV1   = "fallback.append.v1"
	gitOperationFenceV1      = "journal.fence.v1"
	gitOperationPromoteV1    = "journal.promote.v1"
	gitReceiptLockRetryDelay = 10 * time.Millisecond
	maxGitOperationPayload   = 4 << 20
)

var ErrPendingGitDelivery = errors.New("replication: another Git delivery needs recovery")

// GitCommitReceipt is both the durable pre-intent and the finalized pending
// delivery record. CommitSHA is empty only before the append commit is known.
// The operation payload lets a new process finish an interrupted append from
// serialized state, without relying on process memory.
type GitCommitReceipt struct {
	Schema           string          `json:"schema"`
	OperationID      string          `json:"operation_id"`
	OperationKind    string          `json:"operation_kind"`
	OperationPayload json.RawMessage `json:"operation_payload"`
	CommitSHA        string          `json:"commit_sha,omitempty"`
	RemoteRef        string          `json:"remote_ref"`
	ExpectedBase     string          `json:"expected_base,omitempty"`
	AwaitingPush     bool            `json:"awaiting_push"`
	Checksum         string          `json:"checksum"`
}

// GitPushJournal composes a Journal with durable CAS push evidence per call
// (see deliverEvent/GitRemoteDurability below): its own operation lock
// (acquireOperationLock) already serializes every mutating call across the
// whole endpoint, so wrapping a group-commit-batching-enabled DALJournal
// here defeats batching's purpose rather than merely being redundant with
// it. Concretely: GitPushJournal.Append (via deliverEvent -> runOperation ->
// withOperationLock) never lets a second caller even ENTER this type's
// Append while one call is in flight, so the wrapped DALJournal's pending
// batch can never accumulate more than the one event this call is
// delivering -- every call still pays up to the full configured time window
// waiting for a "batch" of exactly one item, with none of batching's
// throughput benefit. state-store/journal-batching's group-commit design is
// for a journal Append callers can reach directly and concurrently (e.g.
// pkg/state/gitstore's current *DALJournal wiring); a future feature that
// wants both durable per-endpoint CAS push AND batched local commits needs
// its own batching-aware redesign of this type (e.g. pushing a whole
// flushed batch's final commit in one CAS operation instead of one push per
// event), which is out of scope here. Construct the wrapped Journal with
// batching disabled (both knobs zero) when composing with GitPushJournal
// until that redesign exists -- NewGitPushJournal refuses to construct
// otherwise, returning an error naming the incompatibility, rather than
// silently degrading to one-item batches that pay the latency with none of
// the throughput.
type GitPushJournal struct {
	Journal
	remote *GitRemoteDurability
}

var (
	_ ReplicaIngestor = (*GitPushJournal)(nil)
	_ RemoteDurable   = (*GitPushJournal)(nil)
)

// Close passes through to the wrapped Journal's Close, if it implements
// BatchedJournal (state-store/journal-batching#ac:close-flushes-pending-batch).
// A Journal without batching support (or with it disabled) has nothing
// pending, so this is a safe no-op for those.
func (j *GitPushJournal) Close(ctx context.Context) error {
	if closer, ok := j.Journal.(interface{ Close(context.Context) error }); ok {
		return closer.Close(ctx)
	}
	return nil
}

// GitRemoteDurability owns cross-process operation serialization, crash-safe
// receipt files, atomic remote CAS, and fetched ancestry proof. testFault is an
// unexported deterministic seam used only by package tests.
type GitRemoteDurability struct {
	repoDir   string
	remote    string
	branch    string
	testFault func(stage string) error
}

// NewGitPushJournal refuses to wrap a journal with group-commit batching
// enabled (state-store/journal-batching): GitPushJournal's own operation
// lock already serializes every mutating call end-to-end, so a wrapped
// journal's pending batch can never accumulate more than the one event a
// given call is delivering -- every call would still pay up to the full
// configured delay window for a "batch" of exactly one item, with none of
// batching's throughput benefit (see the type doc comment above). Loudly
// refusing at construction, rather than silently composing into that
// latency-with-no-benefit shape, is the whole point of this check:
// construct the wrapped Journal with MaxBatchItems/MaxBatchDelayMS both
// explicitly zero instead.
func NewGitPushJournal(journal Journal, repoDir, remote, branch string) (*GitPushJournal, error) {
	if journal == nil {
		return nil, fmt.Errorf("replication: Git push journal needs a journal")
	}
	if batched, ok := journal.(BatchedJournal); ok {
		if settings := batched.BatchSettings(); !settings.disabled() {
			return nil, fmt.Errorf("replication: Git push journal cannot wrap a batching-enabled journal (MaxItems=%d MaxDelayMS=%d): GitPushJournal's own operation lock serializes every mutating call, so composing it with batching only adds latency for none of the throughput benefit -- construct the wrapped journal with MaxBatchItems/MaxBatchDelayMS both explicitly zero", settings.MaxItems, settings.MaxDelayMS)
		}
	}
	durability, err := NewGitRemoteDurability(repoDir, remote, branch)
	if err != nil {
		return nil, err
	}
	return &GitPushJournal{Journal: journal, remote: durability}, nil
}

func NewGitRemoteDurability(repoDir, remote, branch string) (*GitRemoteDurability, error) {
	if strings.TrimSpace(repoDir) == "" || strings.TrimSpace(remote) == "" || strings.TrimSpace(branch) == "" {
		return nil, fmt.Errorf("replication: Git remote durability needs repo, remote, and branch")
	}
	d := &GitRemoteDurability{repoDir: repoDir, remote: remote, branch: branch}
	if out, err := d.git(context.Background(), "check-ref-format", "refs/heads/"+branch); err != nil {
		return nil, fmt.Errorf("replication: invalid Git branch %q: %w: %s", branch, err, out)
	}
	return d, nil
}

func (j *GitPushJournal) Append(ctx context.Context, event Event) error {
	_, err := j.AppendAndPush(ctx, event)
	return err
}

func (j *GitPushJournal) AppendAndPush(ctx context.Context, event Event) (GitCommitReceipt, error) {
	return j.deliverEvent(ctx, event, gitOperationJournalV1, func() error {
		return j.Journal.Append(ctx, event)
	})
}

// IngestReplica is the only mirror mutation seam exposed by GitPushJournal.
// It retains the same durable intent, expected-base CAS, and fetched readback
// proof as an authority append without granting domain code an Append path.
func (j *GitPushJournal) IngestReplica(ctx context.Context, event Event) error {
	_, err := j.IngestReplicaAndPush(ctx, event)
	return err
}

func (j *GitPushJournal) IngestReplicaAndPush(ctx context.Context, event Event) (GitCommitReceipt, error) {
	ingestor, ok := j.Journal.(ReplicaIngestor)
	if !ok {
		return GitCommitReceipt{}, fmt.Errorf("replication: wrapped Git journal has no replica-ingest seam")
	}
	return j.deliverEvent(ctx, event, gitOperationReplicaV1, func() error {
		return ingestor.IngestReplica(ctx, event)
	})
}

func (j *GitPushJournal) deliverEvent(ctx context.Context, event Event, operationKind string, appendFn func() error) (GitCommitReceipt, error) {
	if err := event.Verify(); err != nil {
		return GitCommitReceipt{}, err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return GitCommitReceipt{}, fmt.Errorf("replication: encode journal delivery intent: %w", err)
	}
	operationID := operationKind + ":" + event.Checksum
	return j.remote.runOperation(ctx, operationID, operationKind, payload, appendFn, func() (bool, error) {
		events, err := j.After(ctx, Cursor{})
		if err != nil {
			return false, err
		}
		for _, existing := range events {
			if existing.EventID == event.EventID && existing.Checksum == event.Checksum {
				return true, nil
			}
		}
		return false, nil
	})
}

// ResumeOperation reconstructs an interrupted authority append, replica
// ingest, fence, or promote from the serialized intent. The operation kind
// selects the same role-fenced (or promotable) seam used before the restart.
// A fence/promote operation's local write is never re-attempted here: by the
// time pushLatestCommit ever creates a receipt for one, the local commit
// already happened durably as part of the promotable call itself (see
// FenceAsReplica/PromoteToActive below) — resumeOperation's own
// runOperationLocked short-circuits straight to pushPendingLocked for a
// "pending"-phase receipt, so the no-op appendFn below is never actually
// invoked in practice; it exists only to satisfy resumeOperation's uniform
// dispatch shape.
func (j *GitPushJournal) ResumeOperation(ctx context.Context, operationID string) (GitCommitReceipt, error) {
	return j.remote.resumeOperation(ctx, operationID, []string{gitOperationJournalV1, gitOperationReplicaV1, gitOperationFenceV1, gitOperationPromoteV1}, func(operationKind string, payload json.RawMessage) (func() error, func() (bool, error), error) {
		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, nil, fmt.Errorf("replication: decode pending journal event: %w", err)
		}
		if err := event.Verify(); err != nil {
			return nil, nil, err
		}
		if want := operationKind + ":" + event.Checksum; operationID != want {
			return nil, nil, fmt.Errorf("replication: journal receipt operation ID does not match serialized event")
		}
		verify := func() (bool, error) {
			events, err := j.After(ctx, Cursor{})
			if err != nil {
				return false, err
			}
			for _, existing := range events {
				if existing.EventID == event.EventID && existing.Checksum == event.Checksum {
					return true, nil
				}
			}
			return false, nil
		}
		switch operationKind {
		case gitOperationJournalV1:
			return func() error { return j.Journal.Append(ctx, event) }, verify, nil
		case gitOperationReplicaV1:
			ingestor, ok := j.Journal.(ReplicaIngestor)
			if !ok {
				return nil, nil, fmt.Errorf("replication: wrapped Git journal has no replica-ingest seam")
			}
			return func() error { return ingestor.IngestReplica(ctx, event) }, verify, nil
		case gitOperationFenceV1, gitOperationPromoteV1:
			return func() error { return nil }, verify, nil
		default:
			return nil, nil, fmt.Errorf("replication: unsupported journal operation kind %q", operationKind)
		}
	})
}

// RemoteReceipt implements RemoteDurable (barrier.go): it proves -- via a
// live fetch and ancestry check against the configured Git remote, never a
// cached local receipt -- that at is durably present there. This is what
// lets Wait's mirror barrier satisfy
// state-store/topology#ac:mirror-barrier-proves-git-durability and
// state-store/backends/sqlite#ac:git-barrier-proves-portable-durability
// rather than merely observing a local commit.
//
// Because every IngestReplica/Append call on a GitPushJournal already blocks
// until its own push is durably proven (deliverEvent -> pushPendingLocked ->
// remoteContains), reaching cursor `at` locally on a GitPushJournal-backed
// replica normally already implies remote durability by the time the ingest
// call returned. RemoteReceipt does not trust that inference: it re-verifies
// live every time, so a replica that somehow reached `at` through a
// different path (a bypassed raw DALJournal write, a resumed operation whose
// receipt-vs-push bookkeeping is mid-flight, or simply a caller wanting
// fresh proof rather than an assumption) is never reported durable on faith.
func (j *GitPushJournal) RemoteReceipt(ctx context.Context, at Cursor) (string, bool, error) {
	head, _, err := j.Head(ctx)
	if err != nil {
		return "", false, err
	}
	if compareCursor(head, at) < 0 {
		return "", false, nil
	}
	return j.remote.VerifyLocalHeadDurable(ctx)
}

// promotable type-asserts the wrapped Journal to PromotableJournal, the seam
// FenceAsReplica/PromoteToActive/HeadEvent/RoleEpoch/EndpointID need.
func (j *GitPushJournal) promotable() (PromotableJournal, error) {
	p, ok := j.Journal.(PromotableJournal)
	if !ok {
		return nil, fmt.Errorf("replication: wrapped Git journal has no promotable seam")
	}
	return p, nil
}

func (j *GitPushJournal) EndpointID() string {
	p, err := j.promotable()
	if err != nil {
		return ""
	}
	return p.EndpointID()
}

func (j *GitPushJournal) RoleEpoch() (Role, int64) {
	p, err := j.promotable()
	if err != nil {
		return "", 0
	}
	return p.RoleEpoch()
}

func (j *GitPushJournal) HeadEvent(ctx context.Context) (Event, bool, error) {
	p, err := j.promotable()
	if err != nil {
		return Event{}, false, err
	}
	return p.HeadEvent(ctx)
}

// FenceAsReplica composes the inner journal's fence-first checkpoint write
// with this endpoint's durable CAS push, so a Git-backed active's fence
// commit is never left local-only the way calling the raw *DALJournal
// fencing seam directly (bypassing this wrapper entirely) used to: that
// silently committed only to the local inGitDB working copy and permanently
// jammed this wrapper's ExpectedBase precondition (local != remote with no
// receipt explaining why) for every future operation. It runs under the same
// cross-process operation lock as Append/IngestReplica, so it cannot
// interleave with an in-flight push. It is idempotent the way the inner call
// is: a resumed call whose local fence is already durable re-derives the
// same checkpoint (a no-op locally) and simply retries the push.
//
// Trade-off, disclosed rather than silently accepted: unlike Append/
// IngestReplica, this does not pre-write a durable "intent" receipt before
// the local commit, because the checkpoint's checksum can only be computed
// correctly INSIDE the promotable call's own locked/transactional scope
// (chained onto the endpoint's own fresh head) — exactly what closes the
// TOCTOU findings 1-4 exist to fix — so there is nothing to serialize as an
// intent before that call runs. A crash between the local commit succeeding
// and the receipt file landing (a narrow window: one local commit followed
// by one local file write) leaves a fail-closed but not self-describing
// state — ExpectedBase will correctly refuse every subsequent operation, but
// ResumeOperation has no receipt to key off and needs an operator to
// re-invoke FenceAsReplica/PromoteToActive directly (which durability-checks
// and no-ops the already-done local write, then completes the push) rather
// than resuming automatically via an operation ID.
func (j *GitPushJournal) FenceAsReplica(ctx context.Context, request PromotionRequest, candidateEndpointID string) (Event, error) {
	promotable, err := j.promotable()
	if err != nil {
		return Event{}, err
	}
	var checkpoint Event
	err = j.remote.withPromotionLock(ctx, func() error {
		if _, err := j.remote.ExpectedBase(ctx); err != nil {
			return fmt.Errorf("replication: Git fence needs a synced local/remote base: %w", err)
		}
		var fenceErr error
		checkpoint, fenceErr = promotable.FenceAsReplica(ctx, request, candidateEndpointID)
		if fenceErr != nil {
			return fenceErr
		}
		if _, pushErr := j.remote.pushLatestCommit(ctx, gitOperationFenceV1, checkpoint); pushErr != nil {
			return fmt.Errorf("replication: push fence checkpoint: %w", pushErr)
		}
		return nil
	})
	return checkpoint, err
}

// PromoteToActive composes the inner journal's promotion role-flip with a
// durable CAS push in the same way FenceAsReplica does — see its doc
// comment, including the disclosed trade-off, for the full contract.
func (j *GitPushJournal) PromoteToActive(ctx context.Context, checkpoint Event, replicaIDs []string) error {
	promotable, err := j.promotable()
	if err != nil {
		return err
	}
	return j.remote.withPromotionLock(ctx, func() error {
		if _, err := j.remote.ExpectedBase(ctx); err != nil {
			return fmt.Errorf("replication: Git promote needs a synced local/remote base: %w", err)
		}
		if err := promotable.PromoteToActive(ctx, checkpoint, replicaIDs); err != nil {
			return err
		}
		if _, err := j.remote.pushLatestCommit(ctx, gitOperationPromoteV1, checkpoint); err != nil {
			return fmt.Errorf("replication: push promotion checkpoint: %w", err)
		}
		return nil
	})
}

func (d *GitRemoteDurability) runOperation(ctx context.Context, operationID, kind string, payload json.RawMessage, appendFn func() error, verifyFn func() (bool, error)) (GitCommitReceipt, error) {
	return d.withOperationLock(ctx, func() (GitCommitReceipt, error) {
		return d.runOperationLocked(ctx, operationID, kind, payload, appendFn, verifyFn)
	})
}

func (d *GitRemoteDurability) resumeOperation(ctx context.Context, operationID string, expectedKinds []string, buildAppend func(string, json.RawMessage) (func() error, func() (bool, error), error)) (GitCommitReceipt, error) {
	return d.withOperationLock(ctx, func() (GitCommitReceipt, error) {
		receipt, _, err := d.readOperation(ctx, operationID)
		if err != nil {
			return GitCommitReceipt{}, err
		}
		if !containsString(expectedKinds, receipt.OperationKind) {
			return GitCommitReceipt{}, fmt.Errorf("replication: pending operation %q has kind %q, want one of %v", operationID, receipt.OperationKind, expectedKinds)
		}
		appendFn, verifyFn, err := buildAppend(receipt.OperationKind, receipt.OperationPayload)
		if err != nil {
			return GitCommitReceipt{}, err
		}
		return d.runOperationLocked(ctx, receipt.OperationID, receipt.OperationKind, receipt.OperationPayload, appendFn, verifyFn)
	})
}

func (d *GitRemoteDurability) runOperationLocked(ctx context.Context, operationID, kind string, payload json.RawMessage, appendFn func() error, verifyFn func() (bool, error)) (GitCommitReceipt, error) {
	receipt, phase, err := d.readOperation(ctx, operationID)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return GitCommitReceipt{}, err
	}
	if errors.Is(err, fs.ErrNotExist) {
		if other, listErr := d.otherPendingOperation(ctx, operationID); listErr != nil {
			return GitCommitReceipt{}, listErr
		} else if other != "" {
			return GitCommitReceipt{}, fmt.Errorf("%w: %s", ErrPendingGitDelivery, other)
		}
		expected, baseErr := d.ExpectedBase(ctx)
		if baseErr != nil {
			return GitCommitReceipt{}, baseErr
		}
		receipt = GitCommitReceipt{OperationID: operationID, OperationKind: kind, OperationPayload: append(json.RawMessage(nil), payload...), RemoteRef: "refs/heads/" + d.branch, ExpectedBase: expected, AwaitingPush: true}
		receipt, err = d.activateReceipt(ctx, receipt, "intent")
		if err != nil {
			return GitCommitReceipt{}, err
		}
		phase = "intent"
		if err := d.fault("intent-durable"); err != nil {
			return receipt, err
		}
	} else if receipt.OperationKind != kind || !jsonBytesEqual(receipt.OperationPayload, payload) {
		return GitCommitReceipt{}, fmt.Errorf("replication: operation %q payload does not match durable intent", operationID)
	}
	if phase == "pending" {
		return d.pushPendingLocked(ctx, receipt)
	}

	local, err := d.localHead(ctx)
	if err != nil {
		return receipt, err
	}
	if local == receipt.ExpectedBase {
		if err := appendFn(); err != nil {
			after, headErr := d.localHead(ctx)
			if headErr == nil && after == receipt.ExpectedBase {
				_ = d.removeOperation(ctx, receipt.OperationID)
			}
			return receipt, err
		}
		if err := d.fault("append-committed"); err != nil {
			return receipt, err
		}
		local, err = d.localHead(ctx)
		if err != nil {
			return receipt, err
		}
	}
	if local == "" {
		return receipt, fmt.Errorf("replication: operation %q produced no Git commit", operationID)
	}
	committed, err := verifyFn()
	if err != nil {
		return receipt, err
	}
	if !committed {
		return receipt, fmt.Errorf("replication: operation %q is absent from the discovered Git commit", operationID)
	}
	if receipt.ExpectedBase != "" {
		if err := d.proveLocalAncestor(ctx, receipt.ExpectedBase, local); err != nil {
			return receipt, err
		}
		count, countErr := d.commitCount(ctx, receipt.ExpectedBase, local)
		if countErr != nil {
			return receipt, countErr
		}
		if count > 1 {
			return receipt, fmt.Errorf("replication: operation %q cannot bind %d commits after expected base", operationID, count)
		}
	}
	receipt.CommitSHA = local
	if err := d.fault("before-receipt-activate"); err != nil {
		return receipt, err
	}
	receipt, err = d.activateReceipt(ctx, receipt, "pending")
	if err != nil {
		return receipt, err
	}
	if err := d.fault("receipt-finalized"); err != nil {
		return receipt, err
	}
	return d.pushPendingLocked(ctx, receipt)
}

func (d *GitRemoteDurability) pushPendingLocked(ctx context.Context, receipt GitCommitReceipt) (GitCommitReceipt, error) {
	if err := d.validateReceipt(ctx, receipt, true); err != nil {
		return receipt, err
	}
	if durable, remoteCommit, err := d.remoteContains(ctx, receipt.CommitSHA); err == nil && durable {
		if err := d.synchronizeLocalToRemote(ctx, remoteCommit); err != nil {
			return receipt, err
		}
		return d.completeReceipt(ctx, receipt)
	}
	if receipt.ExpectedBase != "" {
		if err := d.proveLocalAncestor(ctx, receipt.ExpectedBase, receipt.CommitSHA); err != nil {
			return receipt, err
		}
	}
	lease := "--force-with-lease=" + receipt.RemoteRef + ":" + receipt.ExpectedBase
	if out, err := d.git(ctx, "push", lease, d.remote, receipt.CommitSHA+":"+receipt.RemoteRef); err != nil {
		return receipt, fmt.Errorf("replication: CAS push Git commit from expected base %s: %w: %s", receipt.ExpectedBase, err, out)
	}
	if err := d.fault("push-complete"); err != nil {
		return receipt, err
	}
	durable, remoteCommit, err := d.remoteContains(ctx, receipt.CommitSHA)
	if err != nil {
		return receipt, err
	}
	if !durable {
		return receipt, fmt.Errorf("replication: remote ref does not contain pushed commit %s", receipt.CommitSHA)
	}
	if err := d.synchronizeLocalToRemote(ctx, remoteCommit); err != nil {
		return receipt, err
	}
	return d.completeReceipt(ctx, receipt)
}

func (d *GitRemoteDurability) completeReceipt(ctx context.Context, receipt GitCommitReceipt) (GitCommitReceipt, error) {
	if err := d.removeOperation(ctx, receipt.OperationID); err != nil {
		return receipt, err
	}
	receipt.AwaitingPush = false
	receipt.Checksum = ""
	return sealGitReceipt(receipt)
}

func (d *GitRemoteDurability) ExpectedBase(ctx context.Context) (string, error) {
	remote, err := d.remoteHead(ctx)
	if err != nil {
		return "", err
	}
	local, err := d.localHead(ctx)
	if err != nil {
		return "", err
	}
	if remote != local {
		return "", fmt.Errorf("replication: Git remote %s is not expected base %s", remote, local)
	}
	return remote, nil
}

// acquireOperationLock is the cross-process flock every mutating operation on
// one Git-backed endpoint serializes through — Append, IngestReplica, and now
// FenceAsReplica/PromoteToActive (via withPromotionLock) all share it, so a
// fence/promote can never interleave with an in-flight ordinary append push
// or vice versa.
func (d *GitRemoteDurability) acquireOperationLock(ctx context.Context) (*flock.Flock, error) {
	dir, err := d.receiptDir(ctx)
	if err != nil {
		return nil, err
	}
	if err := ensureDurableDirectory(dir); err != nil {
		return nil, fmt.Errorf("replication: create Git receipt directory: %w", err)
	}
	lock := flock.New(filepath.Join(dir, "operation.lock"))
	locked, err := lock.TryLockContext(ctx, gitReceiptLockRetryDelay)
	if err != nil {
		return nil, fmt.Errorf("replication: acquire Git delivery lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("replication: Git delivery lock not acquired")
	}
	return lock, nil
}

func (d *GitRemoteDurability) withOperationLock(ctx context.Context, fn func() (GitCommitReceipt, error)) (GitCommitReceipt, error) {
	lock, err := d.acquireOperationLock(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	defer func() { _ = lock.Unlock() }()
	return fn()
}

// withPromotionLock is withOperationLock's shape for FenceAsReplica/
// PromoteToActive/VerifyLocalHeadDurable, whose result (an Event, nothing, or
// a commit SHA/bool pair -- not a GitCommitReceipt) differs from the
// ordinary append/replica-ingest operations. Despite the name, it is the
// general "hold the exclusive cross-process operation lock for a simple
// func() error" shape; VerifyLocalHeadDurable reuses it for a read-only
// check for the same reason every mutating call does -- it must never race
// an in-flight push's fetch/update-ref bookkeeping.
func (d *GitRemoteDurability) withPromotionLock(ctx context.Context, fn func() error) error {
	lock, err := d.acquireOperationLock(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()
	return fn()
}

// VerifyLocalHeadDurable performs a live remote check -- fetch, then an
// ancestry proof -- that the current local HEAD commit is present on the
// configured remote, under the same cross-process operation lock every other
// mutating/verifying call on this endpoint uses (so it can never race an
// in-flight push). Because a Git-backed journal's cursor at or before HEAD is
// always recorded by an ancestor-or-self of HEAD (commits are appended
// linearly), proving HEAD itself is an ancestor of the remote's ref is
// sufficient evidence that any earlier cursor is durably present remotely
// too; RemoteReceipt (git_push_journal.go) is what establishes "the caller's
// target cursor is at or before HEAD" before ever calling this. It never
// trusts a cached receipt file: every call re-fetches. ok is false (with no
// error) for an unborn local branch, matching remoteContains' own "nothing
// to prove yet" behavior.
func (d *GitRemoteDurability) VerifyLocalHeadDurable(ctx context.Context) (string, bool, error) {
	var commitSHA string
	var durable bool
	err := d.withPromotionLock(ctx, func() error {
		local, err := d.localHead(ctx)
		if err != nil {
			return err
		}
		if local == "" {
			return nil
		}
		ok, remoteCommit, err := d.remoteContains(ctx, local)
		if err != nil {
			return err
		}
		durable, commitSHA = ok, remoteCommit
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return commitSHA, durable, nil
}

// pushLatestCommit durably pushes the current local HEAD commit — which the
// caller (FenceAsReplica/PromoteToActive) has just produced via a promotable
// fence/promote write, already inside acquireOperationLock/withPromotionLock
// — using the same CAS-lease and receipt evidence AppendAndPush uses, so a
// Git-backed promotion checkpoint is never left committed locally without
// either being pushed or leaving a resumable pending receipt behind. The
// receipt is created directly in "pending" phase (no separate "intent"
// pre-image, since the local commit already happened before this is called
// — see FenceAsReplica's doc comment for the disclosed trade-off that follows
// from that).
func (d *GitRemoteDurability) pushLatestCommit(ctx context.Context, kind string, event Event) (GitCommitReceipt, error) {
	local, err := d.localHead(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if local == "" {
		return GitCommitReceipt{}, fmt.Errorf("replication: operation %q produced no Git commit", kind)
	}
	parent := ""
	if out, err := d.git(ctx, "rev-parse", local+"^"); err == nil {
		parent = strings.TrimSpace(string(out))
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return GitCommitReceipt{}, fmt.Errorf("replication: encode promotion checkpoint: %w", err)
	}
	receipt := GitCommitReceipt{OperationID: kind + ":" + event.Checksum, OperationKind: kind, OperationPayload: payload, CommitSHA: local, RemoteRef: "refs/heads/" + d.branch, ExpectedBase: parent, AwaitingPush: true}
	receipt, err = d.activateReceipt(ctx, receipt, "pending")
	if err != nil {
		return receipt, err
	}
	return d.pushPendingLocked(ctx, receipt)
}

func (d *GitRemoteDurability) fault(stage string) error {
	if d.testFault != nil {
		return d.testFault(stage)
	}
	return nil
}

func sealGitReceipt(receipt GitCommitReceipt) (GitCommitReceipt, error) {
	receipt.Schema = gitReceiptSchemaV1
	receipt.Checksum = ""
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return GitCommitReceipt{}, fmt.Errorf("replication: encode Git receipt checksum: %w", err)
	}
	hash := sha256.Sum256(encoded)
	receipt.Checksum = "sha256:" + hex.EncodeToString(hash[:])
	return receipt, nil
}

func verifyGitReceipt(receipt GitCommitReceipt) error {
	if receipt.Schema != gitReceiptSchemaV1 {
		return fmt.Errorf("replication: unsupported Git receipt schema %q", receipt.Schema)
	}
	if strings.TrimSpace(receipt.OperationID) == "" || len(receipt.OperationID) > 512 {
		return fmt.Errorf("replication: invalid Git receipt operation ID")
	}
	switch receipt.OperationKind {
	case gitOperationJournalV1, gitOperationReplicaV1, gitOperationFallbackV1, gitOperationFenceV1, gitOperationPromoteV1:
	default:
		return fmt.Errorf("replication: invalid Git receipt operation kind %q", receipt.OperationKind)
	}
	if len(receipt.OperationPayload) == 0 || len(receipt.OperationPayload) > maxGitOperationPayload || !json.Valid(receipt.OperationPayload) {
		return fmt.Errorf("replication: invalid Git receipt operation payload")
	}
	if !strings.HasPrefix(receipt.RemoteRef, "refs/heads/") {
		return fmt.Errorf("replication: invalid Git receipt remote ref %q", receipt.RemoteRef)
	}
	if !strings.HasPrefix(receipt.Checksum, "sha256:") || len(receipt.Checksum) != len("sha256:")+64 {
		return fmt.Errorf("replication: invalid Git receipt checksum")
	}
	if receipt.ExpectedBase != "" && !isLexicalObjectID(receipt.ExpectedBase) {
		return fmt.Errorf("replication: invalid expected-base object ID")
	}
	if receipt.CommitSHA != "" && !isLexicalObjectID(receipt.CommitSHA) {
		return fmt.Errorf("replication: invalid commit object ID")
	}
	want := receipt.Checksum
	receipt.Checksum = ""
	sealed, err := sealGitReceipt(receipt)
	if err != nil {
		return err
	}
	if sealed.Checksum != want {
		return fmt.Errorf("replication: Git delivery receipt checksum mismatch")
	}
	return nil
}

func isLexicalObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func (d *GitRemoteDurability) validateReceipt(ctx context.Context, receipt GitCommitReceipt, requireCommit bool) error {
	if err := verifyGitReceipt(receipt); err != nil {
		return err
	}
	if receipt.RemoteRef != "refs/heads/"+d.branch || !receipt.AwaitingPush {
		return fmt.Errorf("replication: receipt remote ref or pending state does not match transport")
	}
	if requireCommit && receipt.CommitSHA == "" {
		return fmt.Errorf("replication: finalized receipt has no commit")
	}
	for _, objectID := range []string{receipt.ExpectedBase, receipt.CommitSHA} {
		if objectID == "" {
			continue
		}
		out, err := d.git(ctx, "cat-file", "-e", objectID+"^{commit}")
		if err != nil {
			return fmt.Errorf("replication: receipt object %s is not a local commit: %w: %s", objectID, err, out)
		}
	}
	return nil
}

func (d *GitRemoteDurability) activateReceipt(ctx context.Context, receipt GitCommitReceipt, phase string) (GitCommitReceipt, error) {
	sealed, err := sealGitReceipt(receipt)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if err := d.validateReceipt(ctx, sealed, phase == "pending"); err != nil {
		return GitCommitReceipt{}, err
	}
	path, err := d.receiptPath(ctx, sealed.OperationID, phase)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	encoded, err := json.Marshal(sealed)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if err := activateDurableFile(path, encoded); err != nil {
		return GitCommitReceipt{}, err
	}
	return sealed, nil
}

func activateDurableFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := ensureDurableDirectory(dir); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(data) {
			return nil
		}
		return fmt.Errorf("replication: durable receipt already exists with different content")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".receipt-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			existing, readErr := os.ReadFile(path)
			if readErr == nil && string(existing) == string(data) {
				return nil
			}
		}
		return err
	}
	return syncDirectory(dir)
}

func ensureDurableDirectory(path string) error {
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("replication: durable receipt path %s is not a directory", path)
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return fmt.Errorf("replication: cannot create durable receipt directory %s", path)
	}
	if err := ensureDurableDirectory(parent); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if err := syncDirectory(path); err != nil {
		return err
	}
	return syncDirectory(parent)
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}

func (d *GitRemoteDurability) readOperation(ctx context.Context, operationID string) (GitCommitReceipt, string, error) {
	for _, phase := range []string{"pending", "intent"} {
		path, err := d.receiptPath(ctx, operationID, phase)
		if err != nil {
			return GitCommitReceipt{}, "", err
		}
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return GitCommitReceipt{}, "", err
		}
		var receipt GitCommitReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return GitCommitReceipt{}, "", fmt.Errorf("replication: decode %s receipt: %w", phase, err)
		}
		if err := d.validateReceipt(ctx, receipt, phase == "pending"); err != nil {
			return GitCommitReceipt{}, "", err
		}
		if receipt.OperationID != operationID {
			return GitCommitReceipt{}, "", fmt.Errorf("replication: receipt path does not match operation ID")
		}
		return receipt, phase, nil
	}
	return GitCommitReceipt{}, "", fs.ErrNotExist
}

func (d *GitRemoteDurability) otherPendingOperation(ctx context.Context, operationID string) (string, error) {
	dir, err := d.receiptDir(ctx)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".intent.json") && !strings.HasSuffix(entry.Name(), ".pending.json")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}
		var receipt GitCommitReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return "", err
		}
		if err := verifyGitReceipt(receipt); err != nil {
			return "", err
		}
		if receipt.OperationID != operationID {
			return receipt.OperationID, nil
		}
	}
	return "", nil
}

func (d *GitRemoteDurability) removeOperation(ctx context.Context, operationID string) error {
	dir, err := d.receiptDir(ctx)
	if err != nil {
		return err
	}
	for _, phase := range []string{"pending", "intent"} {
		path, err := d.receiptPath(ctx, operationID, phase)
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return syncDirectory(dir)
}

func (d *GitRemoteDurability) receiptDir(ctx context.Context) (string, error) {
	out, err := d.git(ctx, "rev-parse", "--git-path", "synchestra/replication-pending")
	if err != nil {
		return "", fmt.Errorf("replication: resolve receipt directory: %w: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.repoDir, path)
	}
	return filepath.Clean(path), nil
}

func (d *GitRemoteDurability) receiptPath(ctx context.Context, operationID, phase string) (string, error) {
	if phase != "intent" && phase != "pending" {
		return "", fmt.Errorf("replication: invalid receipt phase %q", phase)
	}
	dir, err := d.receiptDir(ctx)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(operationID))
	return filepath.Join(dir, hex.EncodeToString(hash[:])+"."+phase+".json"), nil
}

func (d *GitRemoteDurability) remoteContains(ctx context.Context, commit string) (bool, string, error) {
	observed, err := d.remoteHead(ctx)
	if err != nil {
		return false, "", err
	}
	if observed == "" {
		return false, "", nil
	}
	refHash := sha256.Sum256([]byte(d.remote + "\x00" + d.branch))
	readbackRef := "refs/synchestra/readback/" + hex.EncodeToString(refHash[:])
	_, _ = d.git(ctx, "update-ref", "-d", readbackRef)
	if out, err := d.git(ctx, "fetch", "--no-tags", d.remote, "+refs/heads/"+d.branch+":"+readbackRef); err != nil {
		return false, "", fmt.Errorf("replication: fetch remote receipt: %w: %s", err, out)
	}
	fetchedOut, err := d.git(ctx, "rev-parse", "--verify", readbackRef+"^{commit}")
	if err != nil {
		return false, "", fmt.Errorf("replication: resolve fetched remote receipt: %w: %s", err, fetchedOut)
	}
	fetched := strings.TrimSpace(string(fetchedOut))
	if fetched != observed {
		return false, "", fmt.Errorf("replication: remote changed during receipt readback: observed %s fetched %s", observed, fetched)
	}
	if out, err := d.git(ctx, "merge-base", "--is-ancestor", commit, fetched); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, fetched, nil
		}
		return false, "", fmt.Errorf("replication: prove remote receipt ancestry: %w: %s", err, out)
	}
	return true, fetched, nil
}

// synchronizeLocalToRemote makes a fetched, proven remote descendant the next
// expected base. A normal fast-forward merge is deliberately used instead of
// reset/update-ref: Git preserves non-conflicting staged and unstaged work and
// refuses the update if it would overwrite local state.
func (d *GitRemoteDurability) synchronizeLocalToRemote(ctx context.Context, remoteCommit string) error {
	if !isLexicalObjectID(remoteCommit) {
		return fmt.Errorf("replication: cannot synchronize invalid remote commit %q", remoteCommit)
	}
	local, err := d.localHead(ctx)
	if err != nil {
		return err
	}
	if local == remoteCommit {
		return nil
	}
	if local == "" {
		return fmt.Errorf("replication: cannot synchronize an unborn local branch")
	}
	if err := d.proveLocalAncestor(ctx, local, remoteCommit); err != nil {
		return err
	}
	if out, err := d.git(ctx, "merge", "--ff-only", remoteCommit); err != nil {
		return fmt.Errorf("replication: fast-forward local Git state to durable remote descendant: %w: %s", err, out)
	}
	after, err := d.localHead(ctx)
	if err != nil {
		return err
	}
	if after != remoteCommit {
		return fmt.Errorf("replication: local Git head %s does not match durable remote %s after fast-forward", after, remoteCommit)
	}
	return nil
}

func (d *GitRemoteDurability) proveLocalAncestor(ctx context.Context, ancestor, descendant string) error {
	if out, err := d.git(ctx, "merge-base", "--is-ancestor", ancestor, descendant); err != nil {
		return fmt.Errorf("replication: commit %s is not descendant of expected base %s: %w: %s", descendant, ancestor, err, out)
	}
	return nil
}

func (d *GitRemoteDurability) commitCount(ctx context.Context, base, head string) (int64, error) {
	out, err := d.git(ctx, "rev-list", "--count", base+".."+head)
	if err != nil {
		return 0, fmt.Errorf("replication: count operation commits: %w: %s", err, out)
	}
	var count int64
	if _, err := fmt.Sscan(strings.TrimSpace(string(out)), &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (d *GitRemoteDurability) localHead(ctx context.Context) (string, error) {
	out, err := d.git(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		if strings.Contains(string(out), "Needed a single revision") || strings.Contains(string(out), "unknown revision") {
			return "", nil
		}
		return "", fmt.Errorf("replication: read local Git head: %w: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *GitRemoteDurability) remoteHead(ctx context.Context) (string, error) {
	out, err := d.git(ctx, "ls-remote", "--heads", d.remote, "refs/heads/"+d.branch)
	if err != nil {
		return "", fmt.Errorf("replication: read Git remote ref: %w: %s", err, out)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", nil
	}
	if !isLexicalObjectID(fields[0]) {
		return "", fmt.Errorf("replication: remote returned invalid object ID")
	}
	return fields[0], nil
}

func (d *GitRemoteDurability) git(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "git", append([]string{"-C", d.repoDir}, args...)...).CombinedOutput()
}

func jsonBytesEqual(a, b []byte) bool {
	var left, right bytes.Buffer
	if json.Compact(&left, a) != nil || json.Compact(&right, b) != nil {
		return false
	}
	return bytes.Equal(left.Bytes(), right.Bytes())
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
