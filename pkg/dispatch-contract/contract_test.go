package dispatchcontract_test

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

const (
	baseRevision = "1111111111111111111111111111111111111111"
	resultCommit = "2222222222222222222222222222222222222222"
)

type mockHub struct {
	mu       sync.Mutex
	now      time.Time
	dispatch dispatchcontract.Dispatch
	attempt  dispatchcontract.Attempt
	logs     []dispatchcontract.LogEvent
}

func newMockHub() *mockHub {
	return &mockHub{now: time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)}
}

func (h *mockHub) create(req dispatchcontract.CreateDispatchRequest) dispatchcontract.CreateDispatchResponse {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.dispatch.ID != "" {
		return dispatchcontract.CreateDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, Dispatch: h.dispatch}
	}
	h.dispatch = dispatchcontract.Dispatch{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "dsp_01mock",
		OwnerID:         "usr_1",
		CreatedBy:       req.CreatedBy,
		IdempotencyKey:  req.IdempotencyKey,
		Intent:          req.Intent,
		Status:          dispatchcontract.DispatchStatusQueued,
		CreatedAt:       h.now,
		UpdatedAt:       h.now,
	}
	return dispatchcontract.CreateDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, Dispatch: h.dispatch, Created: true}
}

func (h *mockHub) claim(req dispatchcontract.ClaimDispatchRequest) dispatchcontract.ClaimDispatchResponse {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.dispatch.Status != dispatchcontract.DispatchStatusQueued {
		return dispatchcontract.ClaimDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, RetryAfterSeconds: 1}
	}
	lease := dispatchcontract.Lease{
		Owner:           req.Capabilities.Identity,
		Generation:      1,
		AcquiredAt:      h.now,
		ExpiresAt:       h.now.Add(time.Minute),
		LastHeartbeatAt: h.now,
	}
	resolved := dispatchcontract.ResolvedExecution{
		Profile:        dispatchcontract.ProfileBalanced,
		Agent:          "claude-code",
		Model:          "claude-sonnet",
		Effort:         "high",
		MappingVersion: "claude-2026-07-22",
		RoutingReason:  "explicit selector sonnet",
	}
	h.attempt = dispatchcontract.Attempt{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "att_01mock",
		DispatchID:      h.dispatch.ID,
		Number:          1,
		Status:          dispatchcontract.AttemptStatusLeased,
		Requested:       h.dispatch.Intent.Requested,
		Resolved:        &resolved,
		Worker:          &req.Capabilities,
		Lease:           &lease,
		CreatedAt:       h.now,
	}
	h.dispatch.Status = dispatchcontract.DispatchStatusLeased
	h.dispatch.ActiveAttemptID = h.attempt.ID
	h.dispatch.AttemptIDs = []string{h.attempt.ID}
	assignment := &dispatchcontract.ClaimAssignment{Dispatch: h.dispatch, Attempt: h.attempt}
	return dispatchcontract.ClaimDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, Assignment: assignment}
}

func (h *mockHub) owns(m dispatchcontract.AttemptMutation) bool {
	return h.attempt.ID == m.AttemptID && h.attempt.Lease != nil &&
		h.attempt.Lease.Owner.WorkerID == m.WorkerID && h.attempt.Lease.Generation == m.LeaseGeneration
}

func (h *mockHub) start(req dispatchcontract.StartAttemptRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.owns(req.AttemptMutation) {
		return errors.New(dispatchcontract.CodeOwnershipLost)
	}
	h.attempt.Status = dispatchcontract.AttemptStatusRunning
	h.attempt.Session = &req.Session
	h.attempt.Logs = req.Session.Logs
	h.attempt.StartedAt = &h.now
	h.dispatch.Status = dispatchcontract.DispatchStatusRunning
	return nil
}

func (h *mockHub) appendLogs(req dispatchcontract.AppendLogsRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.owns(req.AttemptMutation) {
		return errors.New(dispatchcontract.CodeOwnershipLost)
	}
	h.logs = append(h.logs, req.Events...)
	if h.attempt.Logs != nil && len(h.logs) > 0 {
		h.attempt.Logs.LastSequence = h.logs[len(h.logs)-1].Sequence
	}
	return nil
}

func (h *mockHub) complete(req dispatchcontract.CompleteAttemptRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.owns(req.AttemptMutation) {
		return errors.New(dispatchcontract.CodeOwnershipLost)
	}
	h.attempt.Status = dispatchcontract.AttemptStatusCompleted
	h.attempt.Result = &req.Result
	h.attempt.FinishedAt = &h.now
	h.dispatch.Status = dispatchcontract.DispatchStatusCompleted
	h.dispatch.ActiveAttemptID = ""
	return nil
}

func worker(id string) dispatchcontract.WorkerCapabilities {
	return dispatchcontract.WorkerCapabilities{
		Identity:         dispatchcontract.WorkerIdentity{WorkerID: id, HostID: "host_personal_vm", RunnerID: "runner_vm"},
		ProtocolVersions: []string{dispatchcontract.ProtocolVersionV1},
		Agents: []dispatchcontract.AgentCapability{{
			Agent:    "claude-code",
			Profiles: []dispatchcontract.ExecutionProfile{dispatchcontract.ProfileBalanced},
			Models:   []string{"claude-sonnet"},
			Efforts:  []string{"high"},
		}},
		Capabilities:  []string{"git", "worktree", "branch-publication"},
		MaxConcurrent: 1,
	}
}

func mutation(workerID string) dispatchcontract.AttemptMutation {
	return dispatchcontract.AttemptMutation{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		DispatchID:      "dsp_01mock",
		AttemptID:       "att_01mock",
		WorkerID:        workerID,
		LeaseGeneration: 1,
		OperationID:     "op_1",
	}
}

func TestMockedVerticalDispatchHarness(t *testing.T) {
	hub := newMockHub()
	intent := dispatchcontract.DispatchIntent{
		Source: dispatchcontract.DispatchSource{Kind: dispatchcontract.SourceKindAdHoc, AdHoc: &dispatchcontract.AdHocSource{Prompt: "Update dependencies"}},
		Repository: dispatchcontract.RepositorySnapshot{
			CanonicalID:  "github.com/example/repo",
			CloneURL:     "https://github.com/example/repo.git",
			BaseRevision: baseRevision,
		},
		Requested: dispatchcontract.RequestedExecution{Profile: dispatchcontract.ProfileBalanced, ModelSelector: "sonnet"},
	}
	if err := intent.Validate(); err != nil {
		t.Fatalf("intent validation: %v", err)
	}
	created := hub.create(dispatchcontract.CreateDispatchRequest{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		IdempotencyKey:  "idem_1",
		CreatedBy:       "usr_1",
		Intent:          intent,
	})
	if !created.Created || created.Dispatch.Status != dispatchcontract.DispatchStatusQueued {
		t.Fatalf("create = %+v", created)
	}

	workers := []dispatchcontract.WorkerCapabilities{worker("worker_a"), worker("worker_b")}
	responses := make(chan dispatchcontract.ClaimDispatchResponse, len(workers))
	var claims sync.WaitGroup
	for i, capability := range workers {
		claims.Add(1)
		go func(i int, capability dispatchcontract.WorkerCapabilities) {
			defer claims.Done()
			responses <- hub.claim(dispatchcontract.ClaimDispatchRequest{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				RequestID:       fmt.Sprintf("claim_%d", i),
				Capabilities:    capability,
			})
		}(i, capability)
	}
	claims.Wait()
	close(responses)

	var owner string
	assignments := 0
	for response := range responses {
		if response.Assignment != nil {
			assignments++
			owner = response.Assignment.Attempt.Lease.Owner.WorkerID
		}
	}
	if assignments != 1 {
		t.Fatalf("assignments = %d, want exactly one", assignments)
	}

	logRef := &dispatchcontract.LogReference{SessionID: "ses_1", StreamID: "log_1"}
	if err := hub.attempt.Resolved.Validate(); err != nil {
		t.Fatalf("resolved validation: %v", err)
	}
	ownerMutation := mutation(owner)
	if err := hub.start(dispatchcontract.StartAttemptRequest{
		AttemptMutation: ownerMutation,
		Session: dispatchcontract.SessionReference{
			ID: "ses_1", Runtime: "synchestra-host", StartedAt: hub.now, Logs: logRef,
		},
	}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := hub.appendLogs(dispatchcontract.AppendLogsRequest{
		AttemptMutation: ownerMutation,
		Events: []dispatchcontract.LogEvent{{
			Sequence: 1, Timestamp: hub.now, Level: dispatchcontract.LogLevelInfo, Stage: "agent", Message: "completed safely",
		}},
	}); err != nil {
		t.Fatalf("append logs: %v", err)
	}
	result := dispatchcontract.BranchResult{
		RepositoryID: "github.com/example/repo",
		BaseRevision: baseRevision,
		Branch:       "synchestra/dsp_01mock",
		Commit:       resultCommit,
		Summary:      "Updated dependencies",
		Validation: []dispatchcontract.ValidationEvidence{{
			Name: "go test", Command: "go test ./...", Status: dispatchcontract.ValidationPassed,
		}},
		PublishedAt: hub.now,
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("result validation: %v", err)
	}
	if err := hub.complete(dispatchcontract.CompleteAttemptRequest{AttemptMutation: ownerMutation, Result: result}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if hub.dispatch.Status != dispatchcontract.DispatchStatusCompleted || hub.attempt.Result == nil || hub.attempt.Logs.LastSequence != 1 {
		t.Fatalf("terminal state dispatch=%s attempt=%+v", hub.dispatch.Status, hub.attempt)
	}

	stale := mutation("worker_stale")
	if err := hub.complete(dispatchcontract.CompleteAttemptRequest{AttemptMutation: stale, Result: result}); err == nil {
		t.Fatal("stale owner completion unexpectedly succeeded")
	}
}

func TestProtocolAndSafetyValidation(t *testing.T) {
	if dispatchcontract.AttemptStatusQueued != "queued" {
		t.Fatalf("queued attempt status = %q", dispatchcontract.AttemptStatusQueued)
	}
	if err := dispatchcontract.RequireCompatibleProtocol(dispatchcontract.ProtocolVersionV1); err != nil {
		t.Fatal(err)
	}
	if err := dispatchcontract.RequireCompatibleProtocol("synchestra.dispatch.v2"); err == nil {
		t.Fatal("incompatible protocol accepted")
	}
	repository := dispatchcontract.RepositorySnapshot{
		CanonicalID: "github.com/example/repo", CloneURL: "https://token@example.com/repo.git", BaseRevision: baseRevision,
	}
	if err := repository.Validate(); err == nil {
		t.Fatal("credential-bearing clone URL accepted")
	}
	repository.CloneURL = "https://github.com/example/repo.git"
	repository.BaseRevision = "main"
	if err := repository.Validate(); err == nil {
		t.Fatal("symbolic base revision accepted")
	}
}
