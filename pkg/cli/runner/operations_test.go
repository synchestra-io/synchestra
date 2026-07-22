package runner

// Features implemented: cli/runner/dispatch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestObservationOperationsUsePublicHubRoutes(t *testing.T) {
	dispatch := queuedDispatch("dsp_routes", dispatchcontract.DispatchIntent{})
	attempt := dispatchcontract.Attempt{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              "att_retry",
		DispatchID:      dispatch.ID,
		Number:          2,
		Status:          dispatchcontract.AttemptStatusQueued,
		CreatedAt:       fixedTime,
	}
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requirePath(t, request, request.Method, request.URL.Path)
		seen = append(seen, formatRequest(request))
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/v1/dispatches/dsp_routes":
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.GetDispatchResponse{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				Dispatch:        dispatch,
				Attempts:        []dispatchcontract.Attempt{attempt},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/v1/dispatches/dsp_routes/logs":
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.GetLogsResponse{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				Reference:       dispatchcontract.LogReference{SessionID: "ses_1", StreamID: "log_1", LastSequence: 4},
				Events:          []dispatchcontract.LogEvent{{Sequence: 4, Timestamp: fixedTime, Level: dispatchcontract.LogLevelInfo, Stage: "agent", Message: "working"}},
				NextCursor:      4,
			})
		case request.Method == http.MethodPost && request.URL.Path == "/v1/dispatches/dsp_routes/retry":
			var body dispatchcontract.RetryDispatchRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode retry: %v", err)
			}
			if body.ProtocolVersion != dispatchcontract.ProtocolVersionV1 || body.DispatchID != dispatch.ID || body.OperationID != "op_test" || body.RequestedBy != "test-actor" || body.Reason != "again" {
				t.Errorf("retry body = %+v", body)
			}
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.RetryDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, Dispatch: dispatch, Attempt: attempt})
		case request.Method == http.MethodPost && request.URL.Path == "/v1/dispatches/dsp_routes/cancel":
			var body dispatchcontract.CancelDispatchRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode cancel: %v", err)
			}
			if body.ProtocolVersion != dispatchcontract.ProtocolVersionV1 || body.DispatchID != dispatch.ID || body.OperationID != "op_test" || body.RequestedBy != "test-actor" || body.Reason != "stop" {
				t.Errorf("cancel body = %+v", body)
			}
			writeJSONResponse(t, writer, http.StatusOK, dispatchcontract.CancelDispatchResponse{ProtocolVersion: dispatchcontract.ProtocolVersionV1, Dispatch: dispatch})
		default:
			t.Errorf("unexpected route: %s", formatRequest(request))
			writeJSONResponse(t, writer, http.StatusNotFound, dispatchcontract.APIError{Code: dispatchcontract.CodeNotFound, Message: "not found"})
		}
	}))
	defer server.Close()
	deps := testDependencies(t, t.TempDir(), server.URL, server.Client())

	commands := [][]string{
		{"dispatch", "status", "dsp_routes", "--format", "json"},
		{"dispatch", "logs", "dsp_routes", "--cursor", "3", "--format", "json"},
		{"dispatch", "retry", "dsp_routes", "--reason", "again", "--format", "json"},
		{"dispatch", "cancel", "dsp_routes", "--reason", "stop", "--format", "json"},
	}
	for _, args := range commands {
		output, err := executeRunner(t, deps, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		object := decodeSingleObject(t, output)
		if object["resolved"] == nil || object["error"] != nil {
			t.Fatalf("%v output = %#v", args, object)
		}
	}
	want := []string{
		"GET /v1/dispatches/dsp_routes",
		"GET /v1/dispatches/dsp_routes/logs?cursor=3",
		"POST /v1/dispatches/dsp_routes/retry",
		"POST /v1/dispatches/dsp_routes/cancel",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("routes:\n%s\nwant:\n%s", strings.Join(seen, "\n"), strings.Join(want, "\n"))
	}
}

func TestStableExitCodesAndJSONErrors(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		apiError    dispatchcontract.APIError
		response    any
		wantExit    int
		wantCode    string
		wantMessage string
	}{
		{name: "runner not found", status: http.StatusNotFound, apiError: dispatchcontract.APIError{Code: dispatchcontract.CodeNotFound, Message: "runner missing", Details: map[string]string{"resource": "runner"}}, wantExit: exitRunnerNotFound, wantCode: dispatchcontract.CodeNotFound},
		{name: "no eligible worker", status: http.StatusConflict, apiError: dispatchcontract.APIError{Code: dispatchcontract.CodeNoEligibleWorker, Message: "no worker"}, wantExit: exitNoEligibleWorker, wantCode: dispatchcontract.CodeNoEligibleWorker},
		{name: "unsupported selector", status: http.StatusUnprocessableEntity, apiError: dispatchcontract.APIError{Code: dispatchcontract.CodeUnsupportedSelector, Message: "model unsupported"}, wantExit: 2, wantCode: dispatchcontract.CodeUnsupportedSelector},
		{name: "HTTP authentication wins", status: http.StatusUnauthorized, apiError: dispatchcontract.APIError{Code: dispatchcontract.CodeNotFound, Message: "masked"}, wantExit: exitUnauthenticated, wantCode: dispatchcontract.CodeUnauthenticated},
		{name: "protocol mismatch", status: http.StatusOK, response: dispatchcontract.GetDispatchResponse{ProtocolVersion: "synchestra.dispatch.v2", Dispatch: dispatchcontract.Dispatch{ProtocolVersion: "synchestra.dispatch.v2"}}, wantExit: exitIncompatibleProtocol, wantCode: dispatchcontract.CodeIncompatibleProtocol, wantMessage: "upgrade the older component"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if test.response != nil {
					writeJSONResponse(t, writer, test.status, test.response)
					return
				}
				writeJSONResponse(t, writer, test.status, test.apiError)
			}))
			defer server.Close()
			deps := testDependencies(t, t.TempDir(), server.URL, server.Client())
			output, err := executeRunner(t, deps, "dispatch", "status", "dsp_test", "--format", "json")
			if code := exitCode(t, err); code != test.wantExit {
				t.Fatalf("exit code = %d, want %d (%v)", code, test.wantExit, err)
			}
			object := decodeSingleObject(t, output)
			errorObject, ok := object["error"].(map[string]any)
			if !ok || errorObject["code"] != test.wantCode {
				t.Fatalf("error shape = %#v", object)
			}
			if test.wantMessage != "" && !strings.Contains(errorObject["message"].(string), test.wantMessage) {
				t.Fatalf("message = %q", errorObject["message"])
			}
			if object["dispatch"] != nil {
				t.Fatalf("error output also contains dispatch: %#v", object)
			}
		})
	}
}

func TestHubUnreachableExitCode(t *testing.T) {
	deps := testDependencies(t, t.TempDir(), "https://hub.example.test", httpDoerFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}))
	output, err := executeRunner(t, deps, "dispatch", "status", "dsp_test", "--format", "json")
	if code := exitCode(t, err); code != exitHubUnreachable {
		t.Fatalf("exit code = %d, want %d", code, exitHubUnreachable)
	}
	object := decodeSingleObject(t, output)
	if object["error"].(map[string]any)["code"] != "HUB_UNREACHABLE" {
		t.Fatalf("output = %#v", object)
	}
}

func TestUnauthenticatedAndInvalidArgumentsHaveStableExitCodes(t *testing.T) {
	deps := normalizeDependencies(Dependencies{
		Getwd:       func() (string, error) { return t.TempDir(), nil },
		UserHomeDir: func() (string, error) { return t.TempDir(), nil },
		LookupEnv:   func(string) (string, bool) { return "", false },
	})
	output, err := executeRunner(t, deps, "dispatch", "status", "dsp_test", "--format", "json")
	if code := exitCode(t, err); code != exitUnauthenticated {
		t.Fatalf("unauthenticated exit = %d", code)
	}
	object := decodeSingleObject(t, output)
	if object["error"].(map[string]any)["code"] != dispatchcontract.CodeUnauthenticated {
		t.Fatalf("output = %#v", object)
	}

	invalidCases := [][]string{
		{"dispatch", "--prompt", "one", "--plan", "two"},
		{"dispatch", "--prompt", "one", "--profile", "medium"},
		{"dispatch", "--prompt", "one", "two"},
		{"dispatch", "status", "bad/id"},
		{"dispatch", "logs", "dsp_test", "--cursor", "-1"},
		{"dispatch", "status", "dsp_test", "--format", "yaml"},
		{"dispatch", "status", "dsp_test", "--unknown"},
	}
	for _, args := range invalidCases {
		_, err := executeRunner(t, deps, args...)
		if code := exitCode(t, err); code != 2 {
			t.Errorf("%v exit = %d, want 2 (%v)", args, code, err)
		}
	}
	jsonFlagError, err := executeRunner(t, deps, "dispatch", "status", "dsp_test", "--format", "json", "--unknown")
	if code := exitCode(t, err); code != 2 {
		t.Fatalf("JSON flag error exit = %d", code)
	}
	object = decodeSingleObject(t, jsonFlagError)
	if object["resolved"] == nil || object["error"] == nil {
		t.Fatalf("JSON flag error output = %#v", object)
	}
}
