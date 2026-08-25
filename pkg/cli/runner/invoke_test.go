package runner

// Features implemented: wb-session-transport, cli/runner/invoke
// Features depended on:  cli/runner/dispatch, dispatch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestInvokeCreatesTypedDispatchFromOpaqueJSONFile(t *testing.T) {
	repo := newTestRepository(t, map[string]string{"README.md": "# Example\n"})
	payload := []byte("{\n  \"handoff_id\": \"payload-field-must-stay-opaque\",\n  \"command\": \"printf secret-payload\"\n}\n")
	payloadPath := filepath.Join(repo, "handoff.json")
	if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Date(2026, 9, 1, 12, 0, 0, 0, time.FixedZone("offset", 2*60*60))

	var captured dispatchcontract.CreateDispatchRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requirePath(t, request, http.MethodPost, "/v1/dispatches")
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		writeJSONResponse(t, writer, http.StatusCreated, dispatchcontract.CreateDispatchResponse{
			ProtocolVersion: dispatchcontract.ProtocolVersionV1,
			Dispatch:        queuedDispatch("dsp_invoke", captured.Intent),
			Created:         true,
		})
	}))
	defer server.Close()

	deps := testDependencies(t, repo, server.URL, server.Client())
	output, err := executeRunner(t, deps,
		"invoke", "@handoff.json",
		"--runner", "personal-vm",
		"--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1),
		"--invocation-id", "handoff-42",
		"--deadline", deadline.Format(time.RFC3339),
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}

	invocation, ok, err := dispatchcontract.ParseHandlerInvocation(captured.Intent.Source)
	if err != nil || !ok {
		t.Fatalf("parse invocation: ok=%t err=%v", ok, err)
	}
	if invocation.ID != "handoff-42" || invocation.Handler != dispatchcontract.HandlerNameWBSessionAcceptV1 {
		t.Fatalf("invocation identity = %+v", invocation)
	}
	if string(invocation.Payload) != string(payload) {
		t.Fatalf("payload bytes changed: %q", invocation.Payload)
	}
	if invocation.PayloadDigest != dispatchcontract.HandlerPayloadDigest(payload) || invocation.PayloadSize != int64(len(payload)) {
		t.Fatalf("payload evidence = digest %q size %d", invocation.PayloadDigest, invocation.PayloadSize)
	}
	if invocation.Deadline == nil || !invocation.Deadline.Equal(deadline) || invocation.Deadline.Location() != time.UTC {
		t.Fatalf("deadline = %v", invocation.Deadline)
	}
	wantKey, err := dispatchcontract.WBHandoffIdempotencyKey("handoff-42")
	if err != nil {
		t.Fatal(err)
	}
	if captured.IdempotencyKey != wantKey {
		t.Fatalf("idempotency key = %q, want %q", captured.IdempotencyKey, wantKey)
	}
	wantRequested, err := dispatchcontract.HandlerRequestedExecution(dispatchcontract.HandlerNameWBSessionAcceptV1)
	if err != nil {
		t.Fatal(err)
	}
	wantCapability, err := dispatchcontract.HandlerRequiredCapability(dispatchcontract.HandlerNameWBSessionAcceptV1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(captured.Intent.Requested, wantRequested) {
		t.Fatalf("requested execution = %+v, want %+v", captured.Intent.Requested, wantRequested)
	}
	constraints := captured.Intent.Constraints
	if constraints.RunnerID != "personal-vm" || len(constraints.RequiredCapabilities) != 1 || constraints.RequiredCapabilities[0] != wantCapability {
		t.Fatalf("worker constraints = %+v", constraints)
	}
	if captured.Intent.Repository.CanonicalID != "github.com/acme/example" || len(captured.Intent.Repository.BaseRevision) != 40 {
		t.Fatalf("repository evidence = %+v", captured.Intent.Repository)
	}

	object := decodeSingleObject(t, output)
	resolved, ok := object["resolved"].(map[string]any)
	if !ok {
		t.Fatalf("resolved output = %#v", object)
	}
	publicInvocation, ok := resolved["invocation"].(map[string]any)
	if !ok || publicInvocation["id"] != "handoff-42" || publicInvocation["handler"] != string(dispatchcontract.HandlerNameWBSessionAcceptV1) {
		t.Fatalf("public invocation = %#v", publicInvocation)
	}
	if publicInvocation["payload_digest"] != dispatchcontract.HandlerPayloadDigest(payload) || publicInvocation["payload_size"] != float64(len(payload)) {
		t.Fatalf("public payload evidence = %#v", publicInvocation)
	}
	for _, forbidden := range []string{"secret-payload", "project_context", "synchestra.internal.handler_invocation", "synchestra-handler"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("typed output exposed %q: %s", forbidden, output)
		}
	}
	if object["created"] != true || object["dispatch"] == nil || object["error"] != nil {
		t.Fatalf("JSON shape = %#v", object)
	}
}

func TestInvokeReplayReturnsExistingTerminalAttemptAndDigestConflict(t *testing.T) {
	repo := newTestRepository(t, map[string]string{"README.md": "# Example\n"})
	payloadPath := filepath.Join(repo, "handoff.json")
	firstPayload := []byte(`{"handoff_id":"handoff-replay","secret":"payload-one"}`)
	if err := os.WriteFile(payloadPath, firstPayload, 0o600); err != nil {
		t.Fatal(err)
	}

	var storedRequest dispatchcontract.CreateDispatchRequest
	var postCount int
	var statusCount int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/dispatches":
			postCount++
			var createRequest dispatchcontract.CreateDispatchRequest
			if err := json.NewDecoder(request.Body).Decode(&createRequest); err != nil {
				t.Errorf("decode create request: %v", err)
				return
			}
			if storedRequest.IdempotencyKey == "" {
				storedRequest = createRequest
			} else {
				storedPayload, err := json.Marshal(struct {
					CreatedBy string
					Intent    dispatchcontract.DispatchIntent
				}{storedRequest.CreatedBy, storedRequest.Intent})
				if err != nil {
					t.Errorf("marshal stored payload: %v", err)
					return
				}
				incomingPayload, err := json.Marshal(struct {
					CreatedBy string
					Intent    dispatchcontract.DispatchIntent
				}{createRequest.CreatedBy, createRequest.Intent})
				if err != nil {
					t.Errorf("marshal incoming payload: %v", err)
					return
				}
				if createRequest.IdempotencyKey != storedRequest.IdempotencyKey || !bytes.Equal(incomingPayload, storedPayload) {
					writeJSONResponse(t, writer, http.StatusConflict, dispatchcontract.APIError{Code: dispatchcontract.CodeConflict, Message: "idempotency_key was already used with a different payload"})
					return
				}
			}
			dispatch := queuedDispatch("dsp_replay", storedRequest.Intent)
			dispatch.Status = dispatchcontract.DispatchStatusCompleted
			dispatch.AttemptIDs = []string{"att_replay"}
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.CreateDispatchResponse{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				Dispatch:        dispatch,
				Created:         false,
			})
		case request.Method == http.MethodGet && request.URL.Path == "/v1/dispatches/dsp_replay":
			statusCount++
			dispatch := queuedDispatch("dsp_replay", storedRequest.Intent)
			dispatch.Status = dispatchcontract.DispatchStatusCompleted
			dispatch.AttemptIDs = []string{"att_replay"}
			finishedAt := fixedTime.Add(time.Minute)
			attempt := dispatchcontract.Attempt{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				ID:              "att_replay",
				DispatchID:      dispatch.ID,
				Number:          1,
				Status:          dispatchcontract.AttemptStatusCompleted,
				Result: &dispatchcontract.BranchResult{
					Branch:      "synchestra/internal-compatibility-branch",
					Commit:      strings.Repeat("2", 40),
					Summary:     "payload-one",
					Validation:  []dispatchcontract.ValidationEvidence{{Name: "wb-receipt", ArtifactRef: "artifact://wb/receipt-42"}},
					PublishedAt: finishedAt,
				},
				CreatedAt:  fixedTime,
				StartedAt:  &fixedTime,
				FinishedAt: &finishedAt,
			}
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.GetDispatchResponse{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				Dispatch:        dispatch,
				Attempts:        []dispatchcontract.Attempt{attempt},
			})
		default:
			t.Errorf("unexpected request: %s", formatRequest(request))
			writeJSONResponse(t, writer, http.StatusNotFound, dispatchcontract.APIError{Code: dispatchcontract.CodeNotFound, Message: "not found"})
		}
	}))
	defer server.Close()
	deps := testDependencies(t, repo, server.URL, server.Client())
	args := []string{
		"invoke", "@handoff.json",
		"--runner", "personal-vm",
		"--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1),
		"--invocation-id", "handoff-replay",
		"--format", "json",
	}
	for attemptNumber := 1; attemptNumber <= 2; attemptNumber++ {
		output, err := executeRunner(t, deps, args...)
		if err != nil {
			t.Fatalf("replay %d: %v", attemptNumber, err)
		}
		object := decodeSingleObject(t, output)
		if object["created"] != false {
			t.Fatalf("replay %d created = %#v", attemptNumber, object["created"])
		}
		attempts, ok := object["attempts"].([]any)
		if !ok || len(attempts) != 1 {
			t.Fatalf("replay %d attempts = %#v", attemptNumber, object["attempts"])
		}
		result, ok := attempts[0].(map[string]any)["result"].(map[string]any)
		if !ok {
			t.Fatalf("replay %d result = %#v", attemptNumber, attempts[0])
		}
		artifacts, ok := result["artifact_references"].([]any)
		if !ok || len(artifacts) != 1 || artifacts[0] != "artifact://wb/receipt-42" {
			t.Fatalf("replay %d artifacts = %#v", attemptNumber, result)
		}
		for _, forbidden := range []string{"payload-one", "project_context", "synchestra-handler", "internal-compatibility-branch"} {
			if strings.Contains(output, forbidden) {
				t.Fatalf("replay output exposed %q: %s", forbidden, output)
			}
		}
	}
	if postCount != 2 || statusCount != 2 {
		t.Fatalf("requests post=%d status=%d, want 2 each", postCount, statusCount)
	}

	if err := os.WriteFile(payloadPath, []byte(`{"handoff_id":"handoff-replay","secret":"payload-two"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := executeRunner(t, deps, args...)
	if code := exitCode(t, err); code != 1 {
		t.Fatalf("digest conflict exit = %d, want 1 (%v)", code, err)
	}
	if strings.Contains(output, "payload-two") || strings.Contains(output, "payload-one") {
		t.Fatalf("conflict output exposed payload: %s", output)
	}
}

func TestInvokeRejectsInvalidInputsBeforeHubMutation(t *testing.T) {
	repo := newTestRepository(t, map[string]string{"README.md": "# Example\n"})
	if err := os.WriteFile(filepath.Join(repo, "valid.json"), []byte(`{"secret":"do-not-echo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "invalid.json"), []byte(`{"secret":"do-not-echo"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "empty.json"), []byte(" \n\t"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "large.json"), bytes.Repeat([]byte("x"), dispatchcontract.MaxHandlerPayloadBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJSONResponse(t, writer, http.StatusInternalServerError, dispatchcontract.APIError{Code: "UNEXPECTED", Message: "must not be called"})
	}))
	defer server.Close()
	deps := testDependencies(t, repo, server.URL, server.Client())
	base := []string{"--runner", "personal-vm", "--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1), "--invocation-id", "handoff-invalid", "--format", "json"}
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing payload", args: append([]string{"invoke"}, base...)},
		{name: "payload without at sign", args: append([]string{"invoke", "valid.json"}, base...)},
		{name: "missing runner", args: []string{"invoke", "@valid.json", "--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1), "--invocation-id", "handoff-invalid", "--format", "json"}},
		{name: "missing handler", args: []string{"invoke", "@valid.json", "--runner", "personal-vm", "--invocation-id", "handoff-invalid", "--format", "json"}},
		{name: "missing invocation id", args: []string{"invoke", "@valid.json", "--runner", "personal-vm", "--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1), "--format", "json"}},
		{name: "unknown handler", args: []string{"invoke", "@valid.json", "--runner", "personal-vm", "--handler", "sh -c printenv", "--invocation-id", "handoff-invalid", "--format", "json"}},
		{name: "invalid deadline", args: append([]string{"invoke", "@valid.json", "--deadline", "tomorrow"}, base...)},
		{name: "missing file", args: append([]string{"invoke", "@missing.json"}, base...)},
		{name: "invalid JSON", args: append([]string{"invoke", "@invalid.json"}, base...)},
		{name: "empty JSON", args: append([]string{"invoke", "@empty.json"}, base...)},
		{name: "oversized JSON", args: append([]string{"invoke", "@large.json"}, base...)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := executeRunner(t, deps, test.args...)
			if code := exitCode(t, err); code != 2 {
				t.Fatalf("exit = %d, want 2 (%v)", code, err)
			}
			object := decodeSingleObject(t, output)
			if object["resolved"] == nil || object["error"] == nil || object["dispatch"] != nil {
				t.Fatalf("error shape = %#v", object)
			}
			if strings.Contains(output, "do-not-echo") {
				t.Fatalf("error exposed payload: %s", output)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("Hub received %d invalid invocation requests", requests.Load())
	}
}

func TestInvokeLeavesDirtyCheckoutUnchanged(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(repo, "handoff.json"), []byte(`{"handoff_id":"handoff-dirty"}`), 0o600); err != nil {
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
			Dispatch:        queuedDispatch("dsp_dirty_invoke", createRequest.Intent),
			Created:         true,
		})
	}))
	defer server.Close()
	deps := testDependencies(t, repo, server.URL, server.Client())
	if _, err := executeRunner(t, deps,
		"invoke", "@handoff.json",
		"--runner", "personal-vm",
		"--handler", string(dispatchcontract.HandlerNameWBSessionAcceptV1),
		"--invocation-id", "handoff-dirty",
	); err != nil {
		t.Fatalf("invoke: %v", err)
	}
	after := checkoutSnapshot(t, repo)
	if !bytes.Equal(before, after) {
		t.Fatalf("checkout changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
