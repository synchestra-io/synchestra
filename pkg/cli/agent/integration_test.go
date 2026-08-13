package agent

// Features implemented: agent-coordination
// Features depended on:  state-store, state-store/backends/git

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// gitInit creates a minimal Synchestra-recognizable state repo: a Git
// repository with a synchestra-state-repo.yaml marker at its root (direct
// detection — see pkg/cli/resolve.StateRepoPath) and an "origin" remote so
// --project can be autodetected the way a real checkout would have one.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "--initial-branch=main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "synchestra-state-repo.yaml"), nil, 0o644); err != nil {
		t.Fatalf("write marker file: %v", err)
	}
	run("add", "synchestra-state-repo.yaml")
	run("commit", "-m", "init")
	run("remote", "add", "origin", "git@github.com:fair-split/relay.git")
}

// runIn executes the "agent" command tree with args from within dir,
// capturing stdout, and restoring the working directory afterward.
func runIn(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(prevWD); err != nil {
			t.Fatalf("restore Chdir: %v", err)
		}
	}()

	cmd := Command()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs(args)
	execErr := cmd.Execute()
	return out.String(), execErr
}

type claimJSON struct {
	ID         string `json:"id"`
	RunID      string `json:"run_id"`
	FenceEpoch int64  `json:"fence_epoch"`
	FenceToken string `json:"fence_token"`
	ReleasedAt string `json:"released_at"`
}

// TestAgentCommandsEndToEndThroughRealGit exercises the full CLI surface
// task-3 adds — claim, renew, handoff, release, list, message send/list/ack
// — against a real Git-backed state repository (not a mock), proving the
// wiring from cobra command through resolveStore/gitstore/agentstore is
// correct end to end, matching pkg/state/gitstore/agent_test.go's "real
// Git" proof style one layer up the stack.
func TestAgentCommandsEndToEndThroughRealGit(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)

	claimOut, err := runIn(t, repo, "claim", "--repository", "relay", "--run", "run-a", "--branch", "agent/run-a", "--target-ref", "main", "--format", "json")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	var claim claimJSON
	if err := json.Unmarshal([]byte(claimOut), &claim); err != nil {
		t.Fatalf("decode claim output %q: %v", claimOut, err)
	}
	if claim.ID == "" || claim.RunID != "run-a" {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	// A competing claim on the same branch conflicts.
	if _, err := runIn(t, repo, "claim", "--repository", "relay", "--run", "run-b", "--branch", "agent/run-a"); err == nil {
		t.Fatal("competing claim unexpectedly succeeded")
	}

	// list shows the active claim.
	listOut, err := runIn(t, repo, "list", "--repository", "relay", "--active-only", "--format", "json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var claims []claimJSON
	if err := json.Unmarshal([]byte(listOut), &claims); err != nil {
		t.Fatalf("decode list output %q: %v", listOut, err)
	}
	if len(claims) != 1 || claims[0].ID != claim.ID {
		t.Fatalf("list = %+v, want exactly the one active claim", claims)
	}

	// renew with the wrong fence is refused.
	if _, err := runIn(t, repo, "renew", "--claim", claim.ID, "--fence-epoch", "1", "--fence-token", "wrong"); err == nil {
		t.Fatal("renew with wrong fence unexpectedly succeeded")
	}
	renewOut, err := runIn(t, repo, "renew", "--claim", claim.ID, "--fence-epoch", intStr(claim.FenceEpoch), "--fence-token", claim.FenceToken, "--format", "json")
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	var renewed claimJSON
	if err := json.Unmarshal([]byte(renewOut), &renewed); err != nil {
		t.Fatalf("decode renew output %q: %v", renewOut, err)
	}

	// handoff moves ownership to run-b under a new fence.
	handoffOut, err := runIn(t, repo, "handoff", "--claim", claim.ID, "--fence-epoch", intStr(renewed.FenceEpoch), "--fence-token", renewed.FenceToken, "--to-run", "run-b", "--reason", "checkpointed", "--format", "json")
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	var handedOff claimJSON
	if err := json.Unmarshal([]byte(handoffOut), &handedOff); err != nil {
		t.Fatalf("decode handoff output %q: %v", handoffOut, err)
	}
	if handedOff.ID != claim.ID || handedOff.RunID != "run-b" {
		t.Fatalf("unexpected handoff result: %+v", handedOff)
	}
	// The outgoing run's old fence no longer works.
	if _, err := runIn(t, repo, "renew", "--claim", claim.ID, "--fence-epoch", intStr(renewed.FenceEpoch), "--fence-token", renewed.FenceToken); err == nil {
		t.Fatal("renew with the pre-handoff fence unexpectedly succeeded")
	}

	// release by the new owner.
	if _, err := runIn(t, repo, "release", "--claim", claim.ID, "--fence-epoch", intStr(handedOff.FenceEpoch), "--fence-token", handedOff.FenceToken); err != nil {
		t.Fatalf("release: %v", err)
	}
	listOut, err = runIn(t, repo, "list", "--repository", "relay", "--active-only", "--format", "json")
	if err != nil {
		t.Fatalf("list after release: %v", err)
	}
	if err := json.Unmarshal([]byte(listOut), &claims); err != nil {
		t.Fatalf("decode list output %q: %v", listOut, err)
	}
	if len(claims) != 0 {
		t.Fatalf("active claims after release = %+v, want none", claims)
	}
}

// TestAgentMessageCommandsEndToEndThroughRealGit exercises the typed
// negotiation message verbs (send/list/ack) against a real Git-backed
// state repository, including the decision.accepted-needs-evidence rule.
func TestAgentMessageCommandsEndToEndThroughRealGit(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)

	if _, err := runIn(t, repo, "message", "send", "--thread", "fair-split", "--run", "run-cli-owner", "--to", "run-lib-owner", "--kind", "coordination.request", "--body", "need stable ordering"); err != nil {
		t.Fatalf("send request: %v", err)
	}
	if _, err := runIn(t, repo, "message", "send", "--thread", "fair-split", "--run", "run-lib-owner", "--to", "run-cli-owner", "--kind", "coordination.proposal", "--body", "lexical order breaks ties"); err != nil {
		t.Fatalf("send proposal: %v", err)
	}
	if _, err := runIn(t, repo, "message", "send", "--thread", "fair-split", "--run", "run-lib-owner", "--kind", "coordination.decision.accepted", "--body", "accepted"); err == nil {
		t.Fatal("decision.accepted without --evidence unexpectedly succeeded")
	}
	decisionOut, err := runIn(t, repo, "message", "send", "--thread", "fair-split", "--run", "run-lib-owner", "--to", "run-cli-owner",
		"--kind", "coordination.decision.accepted", "--body", "accepted",
		"--evidence", "counterexample:tie at Bob/Carol:msg-2", "--format", "json")
	if err != nil {
		t.Fatalf("send decision: %v", err)
	}
	var decision struct {
		ID       string `json:"id"`
		Evidence []struct {
			Kind      string `json:"kind"`
			Reference string `json:"reference"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(decisionOut), &decision); err != nil {
		t.Fatalf("decode decision output %q: %v", decisionOut, err)
	}
	if len(decision.Evidence) != 1 || decision.Evidence[0].Reference != "msg-2" {
		t.Fatalf("decision evidence = %+v, want one reference to msg-2", decision.Evidence)
	}

	if _, err := runIn(t, repo, "message", "ack", "--message", decision.ID, "--run", "run-cli-owner"); err != nil {
		t.Fatalf("ack: %v", err)
	}

	listOut, err := runIn(t, repo, "message", "list", "--thread", "fair-split", "--format", "json")
	if err != nil {
		t.Fatalf("message list: %v", err)
	}
	var thread []struct {
		Kind        string `json:"kind"`
		SenderRunID string `json:"sender_run_id"`
	}
	if err := json.Unmarshal([]byte(listOut), &thread); err != nil {
		t.Fatalf("decode message list output %q: %v", listOut, err)
	}
	if len(thread) != 3 {
		t.Fatalf("thread length = %d, want 3 (got %+v)", len(thread), thread)
	}
	if thread[0].Kind != "coordination.request" || thread[1].Kind != "coordination.proposal" || thread[2].Kind != "coordination.decision.accepted" {
		t.Fatalf("unexpected thread kind order: %+v", thread)
	}
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

// TestAgentRunCommandsEndToEndThroughRealGit is the CLI-level proof for
// agent-coordination#ac:optional-model-provenance-is-correctable, exercised
// against a real Git-backed state repository with enough prior history
// (several unrelated messages) that "run correct"'s pre-read (Run().Correct
// -> Get -> a full journal scan, unlike Send's O(1) Head()-only precondition
// — see run.go's comment) is genuinely exercised, not accidentally trivial.
func TestAgentRunCommandsEndToEndThroughRealGit(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo)

	// Build up some unrelated history first.
	for i := 0; i < 3; i++ {
		if _, err := runIn(t, repo, "message", "send", "--thread", "warmup", "--run", "run-a", "--body", "warmup"); err != nil {
			t.Fatalf("warmup send %d: %v", i, err)
		}
	}

	startOut, err := runIn(t, repo, "run", "start", "--effort", "e1", "--family", "claude", "--model", "claude-opus-4-6", "--model-provenance", "caller_declared", "--format", "json")
	if err != nil {
		t.Fatalf("run start: %v", err)
	}
	var run struct {
		ID              string `json:"id"`
		Model           string `json:"model"`
		ModelProvenance string `json:"model_provenance"`
	}
	if err := json.Unmarshal([]byte(startOut), &run); err != nil {
		t.Fatalf("decode run start output %q: %v", startOut, err)
	}
	if run.Model != "claude-opus-4-6" || run.ModelProvenance != "caller_declared" {
		t.Fatalf("unexpected started run: %+v", run)
	}

	// A caller later proves the declared model was wrong: correct it to a
	// runtime-observed value without rewriting the original run.started
	// event.
	correctOut, err := runIn(t, repo, "run", "correct", "--run", run.ID, "--model", "claude-sonnet-5", "--model-provenance", "runtime_observed", "--reason", "declared value was wrong", "--format", "json")
	if err != nil {
		t.Fatalf("run correct: %v", err)
	}
	var corrected struct {
		Model           string `json:"model"`
		ModelProvenance string `json:"model_provenance"`
	}
	if err := json.Unmarshal([]byte(correctOut), &corrected); err != nil {
		t.Fatalf("decode run correct output %q: %v", correctOut, err)
	}
	if corrected.Model != "claude-sonnet-5" || corrected.ModelProvenance != "runtime_observed" {
		t.Fatalf("unexpected corrected run: %+v", corrected)
	}

	// A second correction clears the model back to null/unknown rather than
	// guessing.
	clearedOut, err := runIn(t, repo, "run", "correct", "--run", run.ID, "--reason", "runtime exposes no reliable id", "--format", "json")
	if err != nil {
		t.Fatalf("run correct (clear): %v", err)
	}
	var cleared struct {
		Model           string `json:"model,omitempty"`
		ModelProvenance string `json:"model_provenance,omitempty"`
	}
	if err := json.Unmarshal([]byte(clearedOut), &cleared); err != nil {
		t.Fatalf("decode cleared run output %q: %v", clearedOut, err)
	}
	if cleared.Model != "" || cleared.ModelProvenance != "" {
		t.Fatalf("cleared run still reports a model: %+v", cleared)
	}
}
