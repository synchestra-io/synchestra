package runner

// Features implemented: cli/runner/dispatch

import (
	"net/http"
	"testing"
)

func TestNewDispatchClientUsesBoundedDefaultHTTPClient(t *testing.T) {
	client, err := newDispatchClient("https://hub.example.test", "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	httpClient, ok := client.http.(*http.Client)
	if !ok {
		t.Fatalf("default HTTP client type = %T", client.http)
	}
	if httpClient == http.DefaultClient {
		t.Fatal("default dispatch client unexpectedly uses the unbounded shared HTTP client")
	}
	if httpClient.Timeout != defaultHubTimeout {
		t.Fatalf("default HTTP timeout = %s, want %s", httpClient.Timeout, defaultHubTimeout)
	}
}

func TestDefaultDependenciesUseBoundedHTTPClient(t *testing.T) {
	deps := defaultDependencies()
	httpClient, ok := deps.HTTPClient.(*http.Client)
	if !ok {
		t.Fatalf("default dependency HTTP client type = %T", deps.HTTPClient)
	}
	if httpClient.Timeout != defaultHubTimeout {
		t.Fatalf("default dependency HTTP timeout = %s, want %s", httpClient.Timeout, defaultHubTimeout)
	}
}
