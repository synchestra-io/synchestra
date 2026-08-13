package replication

// Features implemented: state-store/backends/git, state-store/topology/offline-fallback
// Features depended on:  state-store/topology, state-store/journal-batching

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ingitdb/dalgo2ingitdb"
	"github.com/ingitdb/ingitdb-go/ingitdb/validator"
)

type gitDurabilityFixture struct {
	bare       string
	root       string
	journal    *DALJournal
	pushed     *GitPushJournal
	role       Role
	endpointID string
	replicaIDs []string
}

func newGitDurabilityFixture(t *testing.T) gitDurabilityFixture {
	return newGitDurabilityFixtureWithRole(t, RoleActive, "git-active", nil)
}

func newGitDurabilityFixtureWithRole(t *testing.T, role Role, endpointID string, replicaIDs []string) gitDurabilityFixture {
	t.Helper()
	ctx := context.Background()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitCommand(t, t.TempDir(), "init", "--bare", "--initial-branch=main", bare)
	root := filepath.Join(t.TempDir(), "writer")
	if out, err := exec.Command("git", "clone", bare, root).CombinedOutput(); err != nil {
		t.Fatalf("clone writer: %v: %s", err, out)
	}
	gitCommand(t, root, "config", "user.email", "test@example.com")
	gitCommand(t, root, "config", "user.name", "Test")
	database, err := dalgo2ingitdb.NewDatabase(root, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureDALJournalSchema(ctx, database); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, root, "add", "-A")
	gitCommand(t, root, "commit", "-m", "journal schema")
	gitCommand(t, root, "push", "origin", "HEAD:refs/heads/main")
	journal, err := NewDALJournal(database, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: endpointID, Role: role, AuthorityEpoch: 1, ReplicaIDs: replicaIDs, CommitMessage: "synchestra state", MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	pushed, err := NewGitPushJournal(journal, root, "origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	return gitDurabilityFixture{bare: bare, root: root, journal: journal, pushed: pushed, role: role, endpointID: endpointID, replicaIDs: append([]string(nil), replicaIDs...)}
}

func reopenGitPushJournal(t *testing.T, fixture gitDurabilityFixture) *GitPushJournal {
	t.Helper()
	database, err := dalgo2ingitdb.NewDatabase(fixture.root, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	journal, err := NewDALJournal(database, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: fixture.endpointID, Role: fixture.role, AuthorityEpoch: 1, ReplicaIDs: fixture.replicaIDs, CommitMessage: "synchestra state", MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	pushed, err := NewGitPushJournal(journal, fixture.root, "origin", "main")
	if err != nil {
		t.Fatal(err)
	}
	return pushed
}

func TestGitDeliveryRecoversEveryAppendReceiptBoundaryFromSerializedIntent(t *testing.T) {
	for _, stage := range []string{"intent-durable", "append-committed", "before-receipt-activate", "receipt-finalized"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newGitDurabilityFixture(t)
			fixture.pushed.remote.testFault = func(got string) error {
				if got == stage {
					return errors.New("injected crash at " + stage)
				}
				return nil
			}
			receipt, err := fixture.pushed.AppendAndPush(context.Background(), relayEvents(t)[0])
			if err == nil || receipt.OperationID == "" {
				t.Fatalf("injected %s failure = %+v, %v", stage, receipt, err)
			}
			commitsBeforeResume := gitCommand(t, fixture.root, "rev-list", "--count", "HEAD")
			restarted := reopenGitPushJournal(t, fixture)
			completed, err := restarted.ResumeOperation(context.Background(), receipt.OperationID)
			if err != nil || completed.AwaitingPush || completed.CommitSHA == "" {
				t.Fatalf("recover %s = %+v, %v", stage, completed, err)
			}
			commitsAfterResume := gitCommand(t, fixture.root, "rev-list", "--count", "HEAD")
			if stage != "intent-durable" && commitsAfterResume != commitsBeforeResume {
				t.Fatalf("recovery appended a second commit: before=%s after=%s", commitsBeforeResume, commitsAfterResume)
			}
			if got := gitCommand(t, fixture.root, "ls-remote", "origin", "refs/heads/main"); !strings.HasPrefix(got, completed.CommitSHA) {
				t.Fatalf("remote receipt = %s, want %s", got, completed.CommitSHA)
			}
		})
	}
}

func TestGitFallbackRecoversCommittedAppendFromSerializedIntent(t *testing.T) {
	for _, stage := range []string{"intent-durable", "append-committed", "before-receipt-activate", "receipt-finalized"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newGitDurabilityFixture(t)
			inbox, err := NewDALFallbackInbox(fixture.journal.db, "github.com/fair-split/relay", "synchestra fallback")
			if err != nil {
				t.Fatal(err)
			}
			fallback, err := NewGitFallbackInboxPush(inbox, fixture.pushed.remote)
			if err != nil {
				t.Fatal(err)
			}
			envelope, err := NewFallbackEnvelope(FallbackEnvelope{ProjectID: "github.com/fair-split/relay", EnvelopeID: "offline", OccurredAt: time.Date(2026, 8, 10, 14, 0, 0, 0, time.UTC), ActorID: "agent:codex", CommandID: "send", IdempotencyKey: "offline", Kind: "message.sent", Payload: json.RawMessage(`{"body":"offline"}`)})
			if err != nil {
				t.Fatal(err)
			}
			fixture.pushed.remote.testFault = func(got string) error {
				if got == stage {
					return errors.New("injected fallback crash at " + stage)
				}
				return nil
			}
			receipt, err := fallback.AppendFallbackAndPush(context.Background(), envelope)
			if err == nil || receipt.OperationID == "" {
				t.Fatalf("fallback %s crash = %+v, %v", stage, receipt, err)
			}
			reopened := reopenGitPushJournal(t, fixture)
			restartedInbox, err := NewDALFallbackInbox(reopened.Journal.(*DALJournal).db, "github.com/fair-split/relay", "synchestra fallback")
			if err != nil {
				t.Fatal(err)
			}
			restarted, err := NewGitFallbackInboxPush(restartedInbox, reopened.remote)
			if err != nil {
				t.Fatal(err)
			}
			completed, err := restarted.ResumeOperation(context.Background(), receipt.OperationID)
			if err != nil || completed.AwaitingPush {
				t.Fatalf("fallback %s resume = %+v, %v", stage, completed, err)
			}
		})
	}
}

func TestGitPushJournalReplicaIngestResumesThroughRoleSafeSeam(t *testing.T) {
	fixture := newGitDurabilityFixtureWithRole(t, RoleReplica, "git-mirror", []string{"git-mirror"})
	event := relayEvents(t)[0]
	fixture.pushed.remote.testFault = func(stage string) error {
		if stage == "receipt-finalized" {
			return errors.New("injected replica delivery crash")
		}
		return nil
	}
	pending, err := fixture.pushed.IngestReplicaAndPush(context.Background(), event)
	if err == nil || !pending.AwaitingPush || pending.OperationKind != gitOperationReplicaV1 {
		t.Fatalf("replica ingest crash = %+v, %v", pending, err)
	}
	restarted := reopenGitPushJournal(t, fixture)
	completed, err := restarted.ResumeOperation(context.Background(), pending.OperationID)
	if err != nil || completed.AwaitingPush {
		t.Fatalf("replica ingest resume = %+v, %v", completed, err)
	}
	if head, hash, err := restarted.Head(context.Background()); err != nil || head != event.Cursor || hash != event.Checksum {
		t.Fatalf("replica head after resume = %+v %q, %v", head, hash, err)
	}
}

// TestPromoteFencesGitBackedActiveThroughDurablePush proves finding #6: when
// the active endpoint Promote fences is Git-backed and passed in as its
// *GitPushJournal wrapper (not the raw *DALJournal underneath), the fence
// checkpoint FenceAsReplica commits locally is durably pushed to the remote
// -- never silently stranded local-only the way calling the raw *DALJournal
// fencing seam directly (bypassing the push wrapper) used to leave it,
// permanently jamming the wrapper's ExpectedBase precondition for every
// future operation on that repo.
func TestPromoteFencesGitBackedActiveThroughDurablePush(t *testing.T) {
	ctx := context.Background()
	fixture := newGitDurabilityFixtureWithRole(t, RoleActive, "git-active", []string{"sqlite-mirror"})
	_, candidate := newSQLiteJournalWithRole(t, nil, RoleReplica, "sqlite-mirror")

	events := relayEvents(t)
	for _, event := range events {
		if _, err := fixture.pushed.AppendAndPush(ctx, event); err != nil {
			t.Fatalf("seed git-active append: %v", err)
		}
	}
	if _, err := Replicate(ctx, fixture.pushed, candidate, "sqlite-mirror"); err != nil {
		t.Fatalf("catch up candidate before promotion: %v", err)
	}

	// Promote is handed the *GitPushJournal wrapper as the active endpoint,
	// not fixture.journal (the raw *DALJournal) -- the whole point of this
	// test is that the fence commit goes through the durable-push layer.
	result, err := Promote(ctx, fixture.pushed, candidate, nil, PromotionRequest{ActorID: "operator", CommandID: "promote-git-active"})
	if err != nil {
		t.Fatalf("promote a Git-backed active through its push wrapper: %v", err)
	}
	if result.Checkpoint.Kind != PromotionCheckpointKind {
		t.Fatalf("checkpoint kind = %q", result.Checkpoint.Kind)
	}
	if role, _ := candidate.roleEpoch(); role != RoleActive {
		t.Fatalf("candidate role after promotion = %v, want active", role)
	}

	// The fence commit must be durably on the remote, not merely the local
	// working copy: the bare origin's ref must match the local repo's HEAD.
	localHead := gitCommand(t, fixture.root, "rev-parse", "HEAD")
	remoteHead := gitCommand(t, fixture.bare, "rev-parse", "refs/heads/main")
	if localHead != remoteHead {
		t.Fatalf("fence checkpoint not durably pushed: local HEAD %s, remote %s", localHead, remoteHead)
	}
	// A completely fresh clone of the remote (standing in for a different
	// process/host reconnecting with no shared local state) must already
	// see the fence checkpoint as the formerly-active endpoint's head.
	fresh := filepath.Join(t.TempDir(), "fresh-after-fence")
	if out, err := exec.Command("git", "clone", fixture.bare, fresh).CombinedOutput(); err != nil {
		t.Fatalf("clone after fence: %v: %s", err, out)
	}
	freshDB, err := dalgo2ingitdb.NewDatabase(fresh, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	freshJournal, err := NewDALJournal(freshDB, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "git-active", Role: RoleActive, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	freshHead, freshHash, err := freshJournal.Head(ctx)
	if err != nil || freshHead != result.Checkpoint.Cursor || freshHash != result.Checkpoint.Checksum {
		t.Fatalf("fresh clone head = %+v %q, %v; want the durably-pushed checkpoint %+v %q", freshHead, freshHash, err, result.Checkpoint.Cursor, result.Checkpoint.Checksum)
	}
	// And it self-fences the normal way, exactly like a naive restarted
	// process that has no idea promotion happened.
	stale := events[0]
	stale.EventID, stale.IdempotencyKey, stale.Checksum = "post-promotion-stale-git", "post-promotion-stale-git", ""
	stale, err = NewEvent(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := freshJournal.Append(ctx, stale); !errors.Is(err, ErrEpochFenced) {
		t.Fatalf("fresh clone append after fence = %v, want %v", err, ErrEpochFenced)
	}
}

func TestGitDeliverySerializesConcurrentExpectedBaseAppendAndReceipt(t *testing.T) {
	fixture := newGitDurabilityFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var intentCalls atomic.Int32
	fixture.pushed.remote.testFault = func(stage string) error {
		if stage == "intent-durable" && intentCalls.Add(1) == 1 {
			close(entered)
			<-release
		}
		return nil
	}
	events := relayEvents(t)
	firstDone := make(chan error, 1)
	go func() { _, err := fixture.pushed.AppendAndPush(context.Background(), events[0]); firstDone <- err }()
	<-entered
	secondDone := make(chan error, 1)
	go func() { _, err := fixture.pushed.AppendAndPush(context.Background(), events[1]); secondDone <- err }()
	select {
	case err := <-secondDone:
		t.Fatalf("second operation escaped serialization before first commit: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	if head, _, err := fixture.journal.Head(context.Background()); err != nil || head != (Cursor{Epoch: 1, Sequence: 2}) {
		t.Fatalf("serialized head = %+v, %v", head, err)
	}
}

func TestGitDeliveryResumeAcceptsFetchedRemoteDescendant(t *testing.T) {
	fixture := newGitDurabilityFixture(t)
	racer := filepath.Join(t.TempDir(), "racer")
	if out, err := exec.Command("git", "clone", fixture.bare, racer).CombinedOutput(); err != nil {
		t.Fatalf("clone racer: %v: %s", err, out)
	}
	gitCommand(t, racer, "config", "user.email", "racer@example.com")
	gitCommand(t, racer, "config", "user.name", "Racer")
	fixture.pushed.remote.testFault = func(stage string) error {
		if stage != "push-complete" {
			return nil
		}
		gitCommand(t, racer, "pull", "--ff-only")
		if err := os.WriteFile(filepath.Join(racer, "remote-descendant.txt"), []byte("descendant\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitCommand(t, racer, "add", "remote-descendant.txt")
		gitCommand(t, racer, "commit", "-m", "remote descendant")
		gitCommand(t, racer, "push", "origin", "HEAD:refs/heads/main")
		return errors.New("crash before readback")
	}
	pending, err := fixture.pushed.AppendAndPush(context.Background(), relayEvents(t)[0])
	if err == nil || !pending.AwaitingPush {
		t.Fatalf("readback crash = %+v, %v", pending, err)
	}
	stagedPath := filepath.Join(fixture.root, "local-staged.txt")
	if err := os.WriteFile(stagedPath, []byte("preserve staged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, fixture.root, "add", "local-staged.txt")
	untrackedPath := filepath.Join(fixture.root, "local-untracked.txt")
	if err := os.WriteFile(untrackedPath, []byte("preserve untracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	restarted := reopenGitPushJournal(t, fixture)
	completed, err := restarted.ResumeOperation(context.Background(), pending.OperationID)
	if err != nil || completed.AwaitingPush {
		t.Fatalf("descendant resume = %+v, %v", completed, err)
	}
	remoteHead := strings.Fields(gitCommand(t, fixture.root, "ls-remote", "origin", "refs/heads/main"))[0]
	if remoteHead == completed.CommitSHA {
		t.Fatal("test did not create a remote descendant")
	}
	gitCommand(t, fixture.root, "merge-base", "--is-ancestor", completed.CommitSHA, remoteHead)
	if localHead := gitCommand(t, fixture.root, "rev-parse", "HEAD"); localHead != remoteHead {
		t.Fatalf("local head after descendant receipt = %s, want %s", localHead, remoteHead)
	}
	status := gitCommand(t, fixture.root, "status", "--short")
	if !strings.Contains(status, "A  local-staged.txt") || !strings.Contains(status, "?? local-untracked.txt") {
		t.Fatalf("descendant synchronization lost dirty state:\n%s", status)
	}

	second, err := restarted.AppendAndPush(context.Background(), relayEvents(t)[1])
	if err != nil || second.AwaitingPush {
		t.Fatalf("append after descendant synchronization = %+v, %v", second, err)
	}
	status = gitCommand(t, fixture.root, "status", "--short")
	if !strings.Contains(status, "A  local-staged.txt") || !strings.Contains(status, "?? local-untracked.txt") {
		t.Fatalf("subsequent append lost dirty state:\n%s", status)
	}
	if names := gitCommand(t, fixture.root, "show", "--format=", "--name-only", second.CommitSHA); strings.Contains(names, "local-staged.txt") || strings.Contains(names, "local-untracked.txt") {
		t.Fatalf("operation commit captured unrelated dirty state:\n%s", names)
	}
	fresh := filepath.Join(t.TempDir(), "fresh-after-descendant")
	if out, err := exec.Command("git", "clone", fixture.bare, fresh).CombinedOutput(); err != nil {
		t.Fatalf("clone descendant result: %v: %s", err, out)
	}
	freshDB, err := dalgo2ingitdb.NewDatabase(fresh, validator.NewCollectionsReader())
	if err != nil {
		t.Fatal(err)
	}
	freshJournal, err := NewDALJournal(freshDB, DALJournalOptions{ProjectID: "github.com/fair-split/relay", EndpointID: "git-mirror", Role: RoleReplica, AuthorityEpoch: 1, MaxBatchItems: zeroBatch, MaxBatchDelayMS: zeroBatch})
	if err != nil {
		t.Fatal(err)
	}
	got, err := freshJournal.After(context.Background(), Cursor{})
	if err != nil || len(got) != 2 || got[0].Checksum != relayEvents(t)[0].Checksum || got[1].Checksum != relayEvents(t)[1].Checksum {
		t.Fatalf("fresh clone after descendant and next append = %#v, %v", got, err)
	}
}

func TestGitDeliveryRejectsRemoteSiblingWithPendingReceiptIntact(t *testing.T) {
	fixture := newGitDurabilityFixture(t)
	tracer := filepath.Join(t.TempDir(), "sibling")
	if out, err := exec.Command("git", "clone", fixture.bare, tracer).CombinedOutput(); err != nil {
		t.Fatalf("clone sibling writer: %v: %s", err, out)
	}
	gitCommand(t, tracer, "config", "user.email", "sibling@example.com")
	gitCommand(t, tracer, "config", "user.name", "Sibling")
	fixture.pushed.remote.testFault = func(stage string) error {
		if stage != "receipt-finalized" {
			return nil
		}
		if err := os.WriteFile(filepath.Join(tracer, "sibling.txt"), []byte("sibling\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitCommand(t, tracer, "add", "sibling.txt")
		gitCommand(t, tracer, "commit", "-m", "sibling commit")
		gitCommand(t, tracer, "push", "origin", "HEAD:refs/heads/main")
		return nil
	}
	pending, err := fixture.pushed.AppendAndPush(context.Background(), relayEvents(t)[0])
	if err == nil || !pending.AwaitingPush {
		t.Fatalf("sibling CAS = %+v, %v", pending, err)
	}
	restarted := reopenGitPushJournal(t, fixture)
	if _, err := restarted.ResumeOperation(context.Background(), pending.OperationID); err == nil {
		t.Fatal("resume overwrote sibling remote history")
	}
	if _, phase, err := restarted.remote.readOperation(context.Background(), pending.OperationID); err != nil || phase != "pending" {
		t.Fatalf("sibling rejection lost pending receipt: phase=%q err=%v", phase, err)
	}
}

func TestGitReceiptFilesAreValidatedAtomicAndPathSafe(t *testing.T) {
	fixture := newGitDurabilityFixture(t)
	ctx := context.Background()
	base, err := fixture.pushed.remote.ExpectedBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	receipt := GitCommitReceipt{OperationID: "../../outside", OperationKind: gitOperationJournalV1, OperationPayload: json.RawMessage(`{"event":"safe"}`), RemoteRef: "refs/heads/main", ExpectedBase: base, AwaitingPush: true}
	sealed, err := fixture.pushed.remote.activateReceipt(ctx, receipt, "intent")
	if err != nil {
		t.Fatal(err)
	}
	path, err := fixture.pushed.remote.receiptPath(ctx, receipt.OperationID, "intent")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := fixture.pushed.remote.receiptDir(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, dir+string(os.PathSeparator)) {
		t.Fatalf("unsafe receipt path %s escaped %s", path, dir)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("receipt mode = %v, %v", info, err)
	}
	if _, err := fixture.pushed.remote.activateReceipt(ctx, receipt, "intent"); err != nil {
		t.Fatalf("idempotent activation: %v", err)
	}
	receipt.OperationPayload = json.RawMessage(`{"event":"different"}`)
	if _, err := fixture.pushed.remote.activateReceipt(ctx, receipt, "intent"); err == nil {
		t.Fatal("non-clobber activation accepted different receipt content")
	}
	sealed.Checksum = "sha256:" + strings.Repeat("0", 64)
	tampered, err := json.Marshal(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fixture.pushed.remote.readOperation(ctx, sealed.OperationID); err == nil {
		t.Fatal("tampered receipt checksum was accepted")
	}
	badObject := receipt
	badObject.OperationID = "bad-object"
	badObject.CommitSHA = strings.Repeat("a", 40)
	if _, err := fixture.pushed.remote.activateReceipt(ctx, badObject, "pending"); err == nil {
		t.Fatal("lexical but nonexistent Git object ID was accepted")
	}
}
