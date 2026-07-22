package runner

// Features implemented: cli/runner/dispatch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	dispatchcontract "github.com/synchestra-io/synchestra-servers/pkg/dispatch-contract"
)

const testPlan = `---
id: PLAN-42
---
# Plan: Upgrade Dependencies

## Tasks

### Task 1: Update module dependencies

Update the module files.

### Task 2: Run validation

Run all validation.
`

func TestCreateAdHocRequestUsesFrozenContract(t *testing.T) {
	repo := newTestRepository(t, map[string]string{"README.md": "# Example\n"})
	var captured dispatchcontract.CreateDispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requirePath(t, request, http.MethodPost, "/v1/dispatches")
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		writeJSONResponse(t, writer, http.StatusCreated, dispatchcontract.CreateDispatchResponse{
			ProtocolVersion: dispatchcontract.ProtocolVersionV1,
			Dispatch:        queuedDispatch("dsp_adhoc", captured.Intent),
			Created:         true,
		})
	}))
	defer server.Close()

	deps := testDependencies(t, repo, server.URL, server.Client())
	output, err := executeRunner(t, deps, "dispatch", "--prompt", "Update dependencies", "--runner", "personal-vm", "--profile", "fast", "--agent", "claude-code", "--model", "sonnet", "--effort", "high", "--format", "json")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if captured.ProtocolVersion != dispatchcontract.ProtocolVersionV1 || captured.IdempotencyKey != "idem_test" || captured.CreatedBy != "test-actor" {
		t.Fatalf("request envelope = %+v", captured)
	}
	if captured.Intent.Source.Kind != dispatchcontract.SourceKindAdHoc || captured.Intent.Source.AdHoc == nil || captured.Intent.Source.AdHoc.Prompt != "Update dependencies" || captured.Intent.Source.SpecScore != nil {
		t.Fatalf("source = %+v", captured.Intent.Source)
	}
	requested := captured.Intent.Requested
	if requested.Profile != dispatchcontract.ProfileFast || requested.Agent != "claude-code" || requested.ModelSelector != "sonnet" || requested.Effort != "high" || requested.Fallback.Mode != dispatchcontract.FallbackReject {
		t.Fatalf("requested execution = %+v", requested)
	}
	if captured.Intent.Constraints.RunnerID != "personal-vm" {
		t.Fatalf("constraints = %+v", captured.Intent.Constraints)
	}
	repository := captured.Intent.Repository
	if repository.CanonicalID != "github.com/acme/example" || repository.CloneURL != "https://github.com/acme/example.git" {
		t.Fatalf("repository identity = %+v", repository)
	}
	if len(repository.BaseRevision) != 40 || repository.BaseRef != "main" {
		t.Fatalf("repository revision = %+v", repository)
	}
	if strings.Contains(repository.CloneURL, "placeholder-credential") || strings.Contains(output, "test-token") || strings.Contains(output, "placeholder-credential") {
		t.Fatalf("credential leaked: request=%+v output=%s", captured, output)
	}
	object := decodeSingleObject(t, output)
	if object["resolved"] == nil || object["dispatch"] == nil || object["error"] != nil {
		t.Fatalf("JSON shape = %#v", object)
	}
	if _, statErr := os.Stat(filepath.Join(repo, ".synchestra")); !os.IsNotExist(statErr) {
		t.Fatalf("ad-hoc dispatch created local task/state: %v", statErr)
	}
}

func TestCreateSpecScorePlanAndTaskShapes(t *testing.T) {
	repo := newTestRepository(t, map[string]string{
		"spec/plans/upgrade.md": testPlan,
		"tasks/remote.md":       "# Task: Remote Update\n\n**Task ID:** TASK-1024\n",
	})
	tests := []struct {
		name     string
		args     []string
		wantKind dispatchcontract.SpecScoreTargetKind
		wantID   string
		wantPath string
	}{
		{name: "plan path", args: []string{"dispatch", "--plan", "spec/plans/upgrade.md", "--format", "json"}, wantKind: dispatchcontract.SpecScoreTargetPlan, wantID: "PLAN-42", wantPath: "spec/plans/upgrade.md"},
		{name: "plan path positional", args: []string{"dispatch", "spec/plans/upgrade.md", "--format", "json"}, wantKind: dispatchcontract.SpecScoreTargetPlan, wantID: "PLAN-42", wantPath: "spec/plans/upgrade.md"},
		{name: "plan name positional", args: []string{"dispatch", "Upgrade Dependencies", "--format", "json"}, wantKind: dispatchcontract.SpecScoreTargetPlan, wantID: "PLAN-42", wantPath: "spec/plans/upgrade.md"},
		{name: "task name", args: []string{"dispatch", "--task", "Run validation", "--format", "json"}, wantKind: dispatchcontract.SpecScoreTargetTask, wantID: "PLAN-42#task-2", wantPath: "spec/plans/upgrade.md"},
		{name: "task path", args: []string{"dispatch", "--task", "tasks/remote.md", "--format", "json"}, wantKind: dispatchcontract.SpecScoreTargetTask, wantID: "TASK-1024", wantPath: "tasks/remote.md"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var captured dispatchcontract.CreateDispatchRequest
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
					t.Errorf("decode request: %v", err)
					return
				}
				writeJSONResponse(t, writer, http.StatusCreated, dispatchcontract.CreateDispatchResponse{
					ProtocolVersion: dispatchcontract.ProtocolVersionV1,
					Dispatch:        queuedDispatch("dsp_target", captured.Intent),
					Created:         true,
				})
			}))
			defer server.Close()
			deps := testDependencies(t, repo, server.URL, server.Client())
			if _, err := executeRunner(t, deps, test.args...); err != nil {
				t.Fatalf("dispatch: %v", err)
			}
			source := captured.Intent.Source
			if source.Kind != dispatchcontract.SourceKindSpecScore || source.SpecScore == nil || source.AdHoc != nil {
				t.Fatalf("source = %+v", source)
			}
			target := source.SpecScore
			if target.TargetKind != test.wantKind || target.TargetID != test.wantID || target.TargetPath != test.wantPath {
				t.Fatalf("target = %+v", target)
			}
			if target.TargetRevision != captured.Intent.Repository.BaseRevision || !strings.HasPrefix(target.SnapshotHash, "sha256:") || len(target.SnapshotHash) != len("sha256:")+64 {
				t.Fatalf("immutable target = %+v", target)
			}
		})
	}
}

func TestTargetAmbiguityFailsWithoutHubMutation(t *testing.T) {
	planA := strings.Replace(testPlan, "id: PLAN-42", "id: PLAN-A", 1)
	planB := strings.Replace(testPlan, "id: PLAN-42", "id: PLAN-B", 1)
	repo := newTestRepository(t, map[string]string{
		"spec/plans/a.md": planA,
		"spec/plans/b.md": planB,
	})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJSONResponse(t, writer, http.StatusInternalServerError, dispatchcontract.APIError{Code: "UNEXPECTED", Message: "must not be called"})
	}))
	defer server.Close()
	deps := testDependencies(t, repo, server.URL, server.Client())
	output, err := executeRunner(t, deps, "dispatch", "--task", "Run validation", "--format", "json")
	if code := exitCode(t, err); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if requests.Load() != 0 {
		t.Fatalf("Hub received %d requests", requests.Load())
	}
	object := decodeSingleObject(t, output)
	errorObject, ok := object["error"].(map[string]any)
	if !ok || errorObject["code"] != dispatchcontract.CodeInvalidRequest || !strings.Contains(errorObject["message"].(string), "ambiguous") {
		t.Fatalf("error output = %#v", object)
	}
}

func TestCreateLeavesDirtyCheckoutUnchanged(t *testing.T) {
	repo := newTestRepository(t, map[string]string{
		"README.md":  "# Example\n",
		"staged.txt": "committed\n",
	})
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Dirty working tree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("staged change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestCommand(t, repo, "git", "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(repo, "untracked.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := checkoutSnapshot(t, repo)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var createRequest dispatchcontract.CreateDispatchRequest
		if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		writeJSONResponse(t, writer, http.StatusCreated, dispatchcontract.CreateDispatchResponse{
			ProtocolVersion: dispatchcontract.ProtocolVersionV1,
			Dispatch:        queuedDispatch("dsp_dirty", createRequest.Intent),
			Created:         true,
		})
	}))
	defer server.Close()
	deps := testDependencies(t, repo, server.URL, server.Client())
	if _, err := executeRunner(t, deps, "dispatch", "--prompt", "Inspect only"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	after := checkoutSnapshot(t, repo)
	if !bytes.Equal(before, after) {
		t.Fatalf("checkout changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func checkoutSnapshot(t *testing.T, repo string) []byte {
	t.Helper()
	var snapshot bytes.Buffer
	for _, command := range [][]string{
		{"status", "--porcelain=v2", "--branch", "--untracked-files=all"},
		{"rev-parse", "HEAD"},
		{"symbolic-ref", "HEAD"},
		{"ls-files", "--stage"},
	} {
		_, _ = snapshot.WriteString(runTestCommand(t, repo, "git", command...))
		_ = snapshot.WriteByte(0)
	}
	for _, name := range []string{"README.md", "staged.txt", "untracked.txt", ".git/index"} {
		data, err := os.ReadFile(filepath.Join(repo, name))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = snapshot.WriteString(name)
		_ = snapshot.WriteByte(0)
		_, _ = snapshot.Write(data)
		_ = snapshot.WriteByte(0)
	}
	return snapshot.Bytes()
}

func TestResolveTargetReadsCommittedSnapshotNotDirtyFile(t *testing.T) {
	repoPath := newTestRepository(t, map[string]string{"spec/plans/upgrade.md": testPlan})
	if err := os.WriteFile(filepath.Join(repoPath, "spec/plans/upgrade.md"), []byte("# Plan: Dirty Replacement\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := resolveRepository(context.Background(), repoPath)
	if err != nil {
		t.Fatal(err)
	}
	target, err := resolveTarget(context.Background(), repo, targetSelector{Kind: dispatchcontract.SpecScoreTargetPlan, Value: "Upgrade Dependencies"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(testPlan))
	if target.SnapshotHash != fmt.Sprintf("sha256:%x", digest) {
		t.Fatalf("snapshot hash = %q", target.SnapshotHash)
	}
}
