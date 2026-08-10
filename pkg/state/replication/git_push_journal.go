package replication

// Features implemented: state-store/backends/git
// Features depended on:  state-store/topology

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitCommitReceipt distinguishes a local durable commit from a remote mirror
// receipt. A non-empty CommitSHA with AwaitingPush=true is deliberately not a
// durability acknowledgement from the Git replica.
type GitCommitReceipt struct {
	CommitSHA    string
	RemoteRef    string
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
	return j.remote.PushExpected(ctx, expected)
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

func (d *GitRemoteDurability) PushExpected(ctx context.Context, expected string) (GitCommitReceipt, error) {
	commit, err := d.localHead(ctx)
	if err != nil {
		return GitCommitReceipt{}, err
	}
	receipt := GitCommitReceipt{CommitSHA: commit, AwaitingPush: true}
	ref := "refs/heads/" + d.branch
	if out, err := d.git(ctx, "push", d.remote, "HEAD:"+ref); err != nil {
		return receipt, fmt.Errorf("replication: push Git commit from expected base %s: %w: %s", expected, err, out)
	}
	remote, err := d.remoteHead(ctx)
	if err != nil {
		return receipt, err
	}
	if remote != commit {
		return receipt, fmt.Errorf("replication: Git remote receipt %s does not match pushed commit %s", remote, commit)
	}
	receipt.RemoteRef, receipt.AwaitingPush = ref, false
	return receipt, nil
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
