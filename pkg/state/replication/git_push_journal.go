package replication

// Features implemented: state-store/backends/git
// Features depended on:  state-store/topology

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitCommitReceipt distinguishes a local durable commit from a remote mirror
// receipt. A non-empty CommitSHA with AwaitingPush=true is deliberately not a
// durability acknowledgement from the Git replica.
type GitCommitReceipt struct {
	CommitSHA    string
	RemoteRef    string
	ExpectedBase string
	AwaitingPush bool
}

// GitPushJournal adds expected-base, non-forcing push semantics to a journal
// whose Git adapter already commits each Append locally. It is the adapter
// boundary responsible for declaring remote durability evidence.
type GitPushJournal struct {
	Journal
	remote *GitRemoteDurability
}

// GitRemoteDurability is the provider-neutral Git transport primitive shared
// by authority journal commits and the separate fallback inbox. It treats a
// local commit as awaiting_push until the expected-base push and remote ref
// readback both succeed.
type GitRemoteDurability struct{ repoDir, remote, branch string }

func NewGitPushJournal(journal Journal, repoDir, remote, branch string) (*GitPushJournal, error) {
	if journal == nil || strings.TrimSpace(repoDir) == "" || strings.TrimSpace(remote) == "" || strings.TrimSpace(branch) == "" {
		return nil, fmt.Errorf("replication: Git push journal needs journal, repo, remote, and branch")
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
	return &GitRemoteDurability{repoDir: repoDir, remote: remote, branch: branch}, nil
}

func (j *GitPushJournal) Append(ctx context.Context, event Event) error {
	_, err := j.AppendAndPush(ctx, event)
	return err
}

func (j *GitPushJournal) AppendAndPush(ctx context.Context, event Event) (GitCommitReceipt, error) {
	expected, err := j.remote.ExpectedBase(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if err := j.Journal.Append(ctx, event); err != nil {
		return GitCommitReceipt{}, err
	}
	pending, err := j.remote.RecordPending(ctx, expected)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	return j.remote.PushPending(ctx, pending)
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

func (d *GitRemoteDurability) RecordPending(ctx context.Context, expected string) (GitCommitReceipt, error) {
	commit, err := d.localHead(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	receipt := GitCommitReceipt{CommitSHA: commit, ExpectedBase: expected, RemoteRef: "refs/heads/" + d.branch, AwaitingPush: true}
	if err := d.writePending(ctx, receipt); err != nil {
		return GitCommitReceipt{}, err
	}
	return receipt, nil
}

func (d *GitRemoteDurability) PushPending(ctx context.Context, receipt GitCommitReceipt) (GitCommitReceipt, error) {
	if receipt.CommitSHA == "" || receipt.RemoteRef != "refs/heads/"+d.branch {
		return receipt, fmt.Errorf("replication: invalid pending Git receipt")
	}
	if receipt.ExpectedBase != "" {
		if out, err := d.git(ctx, "merge-base", "--is-ancestor", receipt.ExpectedBase, receipt.CommitSHA); err != nil {
			return receipt, fmt.Errorf("replication: pending commit is not descendant of expected base: %w: %s", err, out)
		}
	}
	lease := "--force-with-lease=" + receipt.RemoteRef + ":" + receipt.ExpectedBase
	if out, err := d.git(ctx, "push", lease, d.remote, receipt.CommitSHA+":"+receipt.RemoteRef); err != nil {
		return receipt, fmt.Errorf("replication: CAS push Git commit from expected base %s: %w: %s", receipt.ExpectedBase, err, out)
	}
	remote, err := d.remoteHead(ctx)
	if err != nil {
		return receipt, err
	}
	if remote != receipt.CommitSHA {
		return receipt, fmt.Errorf("replication: Git remote receipt %s does not match pushed commit %s", remote, receipt.CommitSHA)
	}
	if err := d.removePending(ctx, receipt.CommitSHA); err != nil {
		return receipt, err
	}
	receipt.AwaitingPush = false
	return receipt, nil
}

// ResumePending retries the exact local commit already recorded after a push
// failure. It performs no new append and succeeds idempotently after a remote
// receipt has already been observed.
func (d *GitRemoteDurability) ResumePending(ctx context.Context, commitSHA string) (GitCommitReceipt, error) {
	receipt, err := d.readPending(ctx, commitSHA)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	if remote, err := d.remoteHead(ctx); err == nil && remote == receipt.CommitSHA {
		if err := d.removePending(ctx, receipt.CommitSHA); err != nil {
			return receipt, err
		}
		receipt.AwaitingPush = false
		return receipt, nil
	}
	return d.PushPending(ctx, receipt)
}

func (d *GitRemoteDurability) pendingPath(ctx context.Context, commit string) (string, error) {
	out, err := d.git(ctx, "rev-parse", "--git-path", "synchestra/replication-pending/"+commit+".json")
	if err != nil {
		return "", fmt.Errorf("replication: resolve pending receipt path: %w: %s", err, out)
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.repoDir, path)
	}
	return path, nil
}
func (d *GitRemoteDurability) writePending(ctx context.Context, receipt GitCommitReceipt) error {
	path, err := d.pendingPath(ctx, receipt.CommitSHA)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
func (d *GitRemoteDurability) readPending(ctx context.Context, commit string) (GitCommitReceipt, error) {
	path, err := d.pendingPath(ctx, commit)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return GitCommitReceipt{}, fmt.Errorf("replication: read pending receipt: %w", err)
	}
	var receipt GitCommitReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return GitCommitReceipt{}, err
	}
	return receipt, nil
}
func (d *GitRemoteDurability) removePending(ctx context.Context, commit string) error {
	path, err := d.pendingPath(ctx, commit)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
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
	return fields[0], nil
}

func (d *GitRemoteDurability) git(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "git", append([]string{"-C", d.repoDir}, args...)...).CombinedOutput()
}
