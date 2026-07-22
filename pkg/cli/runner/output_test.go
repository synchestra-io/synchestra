package runner

// Features implemented: cli/runner/dispatch

import (
	"bytes"
	"strings"
	"testing"

	dispatchcontract "github.com/synchestra-io/synchestra-servers/pkg/dispatch-contract"
)

func TestCreateTextOutputGolden(t *testing.T) {
	repository := dispatchcontract.RepositorySnapshot{
		CanonicalID:  "github.com/acme/example",
		CloneURL:     "https://github.com/acme/example.git",
		BaseRevision: "1111111111111111111111111111111111111111",
		BaseRef:      "main",
		Subdirectory: "services/api",
	}
	requested := dispatchcontract.RequestedExecution{
		Profile:       dispatchcontract.ProfileBalanced,
		Agent:         "claude-code",
		ModelSelector: "sonnet",
		Effort:        "high",
	}
	source := dispatchcontract.DispatchSource{Kind: dispatchcontract.SourceKindAdHoc, AdHoc: &dispatchcontract.AdHocSource{Prompt: "hidden from text"}}
	resolved := resolvedOutput{Operation: "create", Source: &source, Repository: &repository, RequestedExecution: &requested}
	dispatch := queuedDispatch("dsp_golden", dispatchcontract.DispatchIntent{})
	var output bytes.Buffer
	if err := writeCreateText(&output, resolved, dispatch); err != nil {
		t.Fatal(err)
	}
	want := `Resolved:
  source:       ad_hoc
  repository:   github.com/acme/example
  revision:     1111111111111111111111111111111111111111
  base-ref:     main
  subdirectory: services/api
  profile:      balanced
  agent:        claude-code
  model:        sonnet
  effort:       high
  runner:       any

Dispatch:
  dispatch-id: dsp_golden
  status:      queued
  created-at:  2026-07-22T12:34:56Z
`
	if output.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestStatusAndLogsTextOutputGolden(t *testing.T) {
	dispatch := queuedDispatch("dsp_golden", dispatchcontract.DispatchIntent{})
	dispatch.ActiveAttemptID = "att_1"
	attempt := dispatchcontract.Attempt{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "att_1",
		DispatchID:      dispatch.ID,
		Number:          1,
		Status:          dispatchcontract.AttemptStatusRunning,
	}
	var statusOutput bytes.Buffer
	if err := writeStatusText(&statusOutput, dispatchcontract.GetDispatchResponse{Dispatch: dispatch, Attempts: []dispatchcontract.Attempt{attempt}}); err != nil {
		t.Fatal(err)
	}
	wantStatus := `Dispatch:
  dispatch-id: dsp_golden
  status:      queued
  updated-at:  2026-07-22T12:34:56Z
  attempt-id:  att_1

Attempts:
  1  att_1  running
`
	if statusOutput.String() != wantStatus {
		t.Fatalf("status output:\n%s\nwant:\n%s", statusOutput.String(), wantStatus)
	}

	var logsOutput bytes.Buffer
	logs := dispatchcontract.GetLogsResponse{
		Reference:  dispatchcontract.LogReference{StreamID: "log_1"},
		Events:     []dispatchcontract.LogEvent{{Timestamp: fixedTime, Level: dispatchcontract.LogLevelInfo, Stage: "agent", Message: "line one\nline two"}},
		NextCursor: 9,
	}
	if err := writeLogsText(&logsOutput, logs); err != nil {
		t.Fatal(err)
	}
	wantLogs := "Logs:\n  stream-id:   log_1\n  next-cursor: 9\n2026-07-22T12:34:56Z\tinfo\tagent\tline one\\nline two\n"
	if logsOutput.String() != wantLogs {
		t.Fatalf("logs output:\n%s\nwant:\n%s", logsOutput.String(), wantLogs)
	}
	if strings.Contains(logsOutput.String(), "\nline two\n") {
		t.Fatal("multiline log message destabilized record framing")
	}
}

func TestStatusTextTerminalDetailsGolden(t *testing.T) {
	dispatch := queuedDispatch("dsp_terminal", dispatchcontract.DispatchIntent{})
	dispatch.Status = dispatchcontract.DispatchStatusFailed
	success := dispatchcontract.Attempt{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "att_success",
		DispatchID:      dispatch.ID,
		Number:          1,
		Status:          dispatchcontract.AttemptStatusCompleted,
		Resolved: &dispatchcontract.ResolvedExecution{
			Profile:        dispatchcontract.ProfileBalanced,
			Agent:          "claude-code",
			Model:          "claude-sonnet-4-5",
			Effort:         "high",
			MappingVersion: "routing-2026-07",
			RoutingReason:  "balanced profile for repository policy",
		},
		Result: &dispatchcontract.BranchResult{
			Branch:  "synchestra/dsp_terminal",
			Commit:  "2222222222222222222222222222222222222222",
			Summary: "Implemented change",
			Validation: []dispatchcontract.ValidationEvidence{
				{Name: "go-test", Command: "go test ./...", Status: dispatchcontract.ValidationPassed, ExitCode: 0, Summary: "all packages passed", ArtifactRef: "artifact://validation/1"},
			},
		},
	}
	failure := dispatchcontract.Attempt{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "att_failure",
		DispatchID:      dispatch.ID,
		Number:          2,
		Status:          dispatchcontract.AttemptStatusFailed,
		Failure: &dispatchcontract.TerminalFailure{
			Stage:     "validation",
			Code:      "TEST_FAILED",
			Message:   "go test failed\ninspect the log",
			Retryable: true,
			Logs: &dispatchcontract.LogReference{
				SessionID:    "sess_2",
				StreamID:     "log_2",
				Href:         "https://hub.example.test/v1/dispatches/dsp_terminal/logs",
				LastSequence: 42,
			},
		},
	}

	var output bytes.Buffer
	if err := writeStatusText(&output, dispatchcontract.GetDispatchResponse{Dispatch: dispatch, Attempts: []dispatchcontract.Attempt{success, failure}}); err != nil {
		t.Fatal(err)
	}
	want := `Dispatch:
  dispatch-id: dsp_terminal
  status:      failed
  updated-at:  2026-07-22T12:34:56Z
  attempt-id:  -

Attempts:
  1  att_success  completed
    Resolved execution:
      profile:         balanced
      agent:           claude-code
      model:           claude-sonnet-4-5
      effort:          high
      mapping-version: routing-2026-07
      mapping-reason:  balanced profile for repository policy
    Result:
      branch:          synchestra/dsp_terminal
      commit:          2222222222222222222222222222222222222222
      summary:         Implemented change
      validation:
        - go-test: passed (exit 0)
          command:  go test ./...
          summary:  all packages passed
          artifact: artifact://validation/1
  2  att_failure  failed
    Failure:
      stage:           validation
      code:            TEST_FAILED
      message:         go test failed\ninspect the log
      retryable:       true
      logs:
        href:          https://hub.example.test/v1/dispatches/dsp_terminal/logs
        stream-id:     log_2
        session-id:    sess_2
        last-sequence: 42
`
	if output.String() != want {
		t.Fatalf("status output:\n%s\nwant:\n%s", output.String(), want)
	}
}
