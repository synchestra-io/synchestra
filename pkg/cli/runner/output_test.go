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
