package runner

// Features implemented: cli/runner/dispatch, cli/runner/invoke, wb-session-transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

var fixedTime = time.Date(2026, 7, 22, 12, 34, 56, 0, time.UTC)

type httpDoerFunc func(*http.Request) (*http.Response, error)

func (function httpDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func newTestRepository(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	runTestCommand(t, dir, "git", "init", "-b", "main")
	runTestCommand(t, dir, "git", "config", "user.name", "Dispatch Test")
	runTestCommand(t, dir, "git", "config", "user.email", "dispatch@example.com")
	runTestCommand(t, dir, "git", "config", "commit.gpgsign", "false")
	runTestCommand(t, dir, "git", "remote", "add", "origin", "https://github.com/acme/example.git")
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runTestCommand(t, dir, "git", "add", ".")
	runTestCommand(t, dir, "git", "commit", "-m", "initial")
	return dir
}

func runTestCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}

func testDependencies(t *testing.T, cwd, hubURL string, client httpDoer) Dependencies {
	t.Helper()
	home := t.TempDir()
	environment := map[string]string{
		"SYNCHESTRA_URL":   hubURL,
		"SYNCHESTRA_TOKEN": "test-token",
		"SYNCHESTRA_ACTOR": "test-actor",
	}
	return Dependencies{
		Getwd:       func() (string, error) { return cwd, nil },
		UserHomeDir: func() (string, error) { return home, nil },
		LookupEnv: func(key string) (string, bool) {
			value, ok := environment[key]
			return value, ok
		},
		HTTPClient: client,
		NewID: func(prefix string) (string, error) {
			return prefix + "test", nil
		},
	}
}

func executeRunner(t *testing.T, deps Dependencies, args ...string) (string, error) {
	t.Helper()
	cmd := Command(deps)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func queuedDispatch(id string, intent dispatchcontract.DispatchIntent) dispatchcontract.Dispatch {
	return dispatchcontract.Dispatch{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		ID:              id,
		OwnerID:         "usr_test",
		CreatedBy:       "test-actor",
		IdempotencyKey:  "idem_test",
		Intent:          intent,
		Status:          dispatchcontract.DispatchStatusQueued,
		CreatedAt:       fixedTime,
		UpdatedAt:       fixedTime,
	}
}

func writeJSONResponse(t *testing.T, writer http.ResponseWriter, status int, value any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		t.Fatal("expected command error")
	}
	type exitCoder interface{ ExitCode() int }
	coder, ok := err.(exitCoder)
	if !ok {
		t.Fatalf("error %T does not expose ExitCode: %v", err, err)
	}
	return coder.ExitCode()
}

func decodeSingleObject(t *testing.T, output string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(output))
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		t.Fatalf("output contains more than one JSON value: %s", output)
	}
	return value
}

func requirePath(t *testing.T, request *http.Request, method, path string) {
	t.Helper()
	if request.Method != method || request.URL.Path != path {
		t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, method, path)
	}
	if request.Header.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Authorization header = %q", request.Header.Get("Authorization"))
	}
	if request.Header.Get("X-Synchestra-Dispatch-Protocol") != dispatchcontract.ProtocolVersionV1 {
		t.Errorf("protocol header = %q", request.Header.Get("X-Synchestra-Dispatch-Protocol"))
	}
}

func formatRequest(request *http.Request) string {
	return fmt.Sprintf("%s %s", request.Method, request.URL.RequestURI())
}
