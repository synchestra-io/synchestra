package dispatchcontract_test

// Features implemented: wb-session-transport
// Features depended on:  dispatch

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestHandlerInvocationRoundTripKeepsPayloadOpaque(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 8, 25, 12, 30, 0, 123, time.FixedZone("caller", 60*60))
	deadline := createdAt.Add(10 * time.Minute)
	payload := []byte(`{"command":"sh -c 'rm -rf /'","argv":["curl","https://example.invalid"]}`)

	invocation, err := dispatchcontract.NewHandlerInvocation(
		"inv_01safe",
		dispatchcontract.HandlerNameWBSessionAcceptV1,
		payload,
		createdAt,
		&deadline,
	)
	if err != nil {
		t.Fatalf("construct invocation: %v", err)
	}
	if invocation.PayloadDigest != dispatchcontract.HandlerPayloadDigest(payload) {
		t.Fatalf("payload digest = %q, want %q", invocation.PayloadDigest, dispatchcontract.HandlerPayloadDigest(payload))
	}
	if invocation.PayloadSize != int64(len(payload)) {
		t.Fatalf("payload size = %d, want %d", invocation.PayloadSize, len(payload))
	}
	if invocation.CreatedAt.Location() != time.UTC || invocation.Deadline == nil || invocation.Deadline.Location() != time.UTC {
		t.Fatalf("timestamps are not canonical UTC: created=%v deadline=%v", invocation.CreatedAt, invocation.Deadline)
	}

	source, err := dispatchcontract.EncodeHandlerInvocation(invocation)
	if err != nil {
		t.Fatalf("encode invocation: %v", err)
	}
	parsed, ok, err := dispatchcontract.ParseHandlerInvocation(source)
	if err != nil {
		t.Fatalf("parse invocation: %v", err)
	}
	if !ok {
		t.Fatal("encoded handler invocation was classified as ordinary dispatch")
	}
	if !reflect.DeepEqual(parsed, invocation) {
		t.Fatalf("parsed invocation = %#v, want %#v", parsed, invocation)
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Fatal("opaque payload changed during compatibility-envelope round trip")
	}

	payload[0] = '!'
	if bytes.Equal(invocation.Payload, payload) {
		t.Fatal("constructor retained the caller's mutable payload slice")
	}
}

func TestHandlerInvocationRejectsUnknownHandlersWithoutEchoingRequestData(t *testing.T) {
	t.Parallel()

	unknown := dispatchcontract.HandlerName("sh -c 'printenv SECRET'")
	payload := []byte(`{"command":"/bin/sh","argv":["-c","payload-must-not-appear"]}`)
	_, err := dispatchcontract.NewHandlerInvocation("inv_01safe", unknown, payload, time.Now(), nil)
	if err == nil {
		t.Fatal("unknown request-controlled handler was accepted")
	}
	if strings.Contains(err.Error(), string(unknown)) || strings.Contains(err.Error(), "payload-must-not-appear") {
		t.Fatalf("validation error exposed request data: %q", err)
	}
}

func TestHandlerInvocationParserRejectsCommandFieldsAndDigestTampering(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"body":"opaque-payload-secret"}`)
	invocation, err := dispatchcontract.NewHandlerInvocation(
		"inv_01safe",
		dispatchcontract.HandlerNameWBSessionMessageV1,
		payload,
		time.Date(2026, 8, 25, 12, 30, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("construct invocation: %v", err)
	}
	source, err := dispatchcontract.EncodeHandlerInvocation(invocation)
	if err != nil {
		t.Fatalf("encode invocation: %v", err)
	}

	contextKey, encoded := onlyProjectContextEntry(t, source)
	var envelope map[string]any
	if err := json.Unmarshal([]byte(encoded), &envelope); err != nil {
		t.Fatalf("decode test envelope: %v", err)
	}
	envelope["argv"] = []string{"sh", "-c", "printenv opaque-payload-secret"}
	source.AdHoc.ProjectContext[contextKey] = mustMarshalString(t, envelope)
	_, ok, err := dispatchcontract.ParseHandlerInvocation(source)
	if !ok || err == nil {
		t.Fatal("request-controlled argv field was accepted")
	}
	if strings.Contains(err.Error(), "printenv") || strings.Contains(err.Error(), "opaque-payload-secret") {
		t.Fatalf("parser error exposed request data: %q", err)
	}

	delete(envelope, "argv")
	envelope["payload_digest"] = "sha256:" + strings.Repeat("0", 64)
	source.AdHoc.ProjectContext[contextKey] = mustMarshalString(t, envelope)
	_, ok, err = dispatchcontract.ParseHandlerInvocation(source)
	if !ok || err == nil {
		t.Fatal("payload digest tampering was accepted")
	}
	if strings.Contains(err.Error(), "opaque-payload-secret") {
		t.Fatalf("digest error exposed payload data: %q", err)
	}

	envelope["payload_digest"] = invocation.PayloadDigest
	envelope["handler"] = "unknown.handler; printenv opaque-payload-secret"
	source.AdHoc.ProjectContext[contextKey] = mustMarshalString(t, envelope)
	_, ok, err = dispatchcontract.ParseHandlerInvocation(source)
	if !ok || err == nil {
		t.Fatal("unknown request-controlled handler was accepted by the runner-facing parser")
	}
	if strings.Contains(err.Error(), "printenv") || strings.Contains(err.Error(), "opaque-payload-secret") {
		t.Fatalf("unknown-handler error exposed request data: %q", err)
	}
}

func TestHandlerInvocationEnforcesPayloadBounds(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 8, 25, 12, 30, 0, 0, time.UTC)
	for _, test := range []struct {
		name    string
		payload []byte
	}{
		{name: "empty", payload: nil},
		{name: "too large", payload: bytes.Repeat([]byte("s"), dispatchcontract.MaxHandlerPayloadBytes+1)},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := dispatchcontract.NewHandlerInvocation(
				"inv_01safe",
				dispatchcontract.HandlerNameWBSessionAcceptV1,
				test.payload,
				createdAt,
				nil,
			)
			if err == nil {
				t.Fatal("out-of-bounds payload was accepted")
			}
			if len(test.payload) > 0 && strings.Contains(err.Error(), string(test.payload[:min(len(test.payload), 32)])) {
				t.Fatalf("size error exposed payload data: %q", err)
			}
		})
	}
}

func TestHandlerInvocationHelpersUseClosedSyntheticRouting(t *testing.T) {
	t.Parallel()

	requested, err := dispatchcontract.HandlerRequestedExecution(dispatchcontract.HandlerNameWBSessionAcceptV1)
	if err != nil {
		t.Fatalf("handler requested execution: %v", err)
	}
	if requested.Profile != dispatchcontract.ProfileBalanced ||
		requested.Agent == "" ||
		requested.ModelSelector != string(dispatchcontract.HandlerNameWBSessionAcceptV1) ||
		requested.Fallback.Mode != dispatchcontract.FallbackReject {
		t.Fatalf("synthetic requested execution = %+v", requested)
	}

	capability, err := dispatchcontract.HandlerRequiredCapability(dispatchcontract.HandlerNameWBSessionAcceptV1)
	if err != nil {
		t.Fatalf("handler capability: %v", err)
	}
	if capability == "" || capability == string(dispatchcontract.HandlerNameWBSessionAcceptV1) {
		t.Fatalf("handler capability is not namespaced: %q", capability)
	}

	agent, err := dispatchcontract.HandlerAgentCapability(
		dispatchcontract.HandlerNameWBSessionAcceptV1,
		dispatchcontract.HandlerNameWBSessionMessageV1,
	)
	if err != nil {
		t.Fatalf("handler agent capability: %v", err)
	}
	if agent.Agent != requested.Agent || !reflect.DeepEqual(agent.Profiles, []dispatchcontract.ExecutionProfile{dispatchcontract.ProfileBalanced}) {
		t.Fatalf("synthetic handler agent = %+v", agent)
	}
	if !reflect.DeepEqual(agent.Models, []string{
		string(dispatchcontract.HandlerNameWBSessionAcceptV1),
		string(dispatchcontract.HandlerNameWBSessionMessageV1),
	}) {
		t.Fatalf("synthetic handler selectors = %v", agent.Models)
	}

	requestData := dispatchcontract.HandlerName("/bin/sh")
	if _, err := dispatchcontract.HandlerRequestedExecution(requestData); err == nil {
		t.Fatal("unknown handler produced a synthetic selector")
	}
	if _, err := dispatchcontract.HandlerRequiredCapability(requestData); err == nil {
		t.Fatal("unknown handler produced a worker capability")
	}
}

func TestWBHandoffIdempotencyKeyIsDeterministicAndDoesNotExposeTheID(t *testing.T) {
	t.Parallel()

	handoffID := "handoff-01; printenv SECRET"
	first, err := dispatchcontract.WBHandoffIdempotencyKey(handoffID)
	if err != nil {
		t.Fatalf("derive idempotency key: %v", err)
	}
	second, err := dispatchcontract.WBHandoffIdempotencyKey(handoffID)
	if err != nil {
		t.Fatalf("derive idempotency key again: %v", err)
	}
	other, err := dispatchcontract.WBHandoffIdempotencyKey("handoff-02")
	if err != nil {
		t.Fatalf("derive other idempotency key: %v", err)
	}
	if first != second || first == other {
		t.Fatalf("idempotency keys first=%q second=%q other=%q", first, second, other)
	}
	if strings.Contains(first, handoffID) || len(first) > 128 {
		t.Fatalf("idempotency key exposes or fails to bound handoff ID: %q", first)
	}
	if _, err := dispatchcontract.WBHandoffIdempotencyKey(" \t\n"); err == nil {
		t.Fatal("blank handoff ID produced an idempotency key")
	}
}

func TestOrdinaryDispatchSourceJSONRemainsByteCompatible(t *testing.T) {
	t.Parallel()

	source := dispatchcontract.DispatchSource{
		Kind: dispatchcontract.SourceKindAdHoc,
		AdHoc: &dispatchcontract.AdHocSource{
			Prompt:         "ordinary dispatch",
			ProjectContext: map[string]string{"ticket": "ABC-123"},
		},
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal ordinary source: %v", err)
	}
	want := `{"kind":"ad_hoc","ad_hoc":{"prompt":"ordinary dispatch","project_context":{"ticket":"ABC-123"}}}`
	if string(encoded) != want {
		t.Fatalf("ordinary source JSON = %s, want %s", encoded, want)
	}
	if _, ok, err := dispatchcontract.ParseHandlerInvocation(source); err != nil || ok {
		t.Fatalf("ordinary source parsed as handler invocation: ok=%v err=%v", ok, err)
	}
}

func onlyProjectContextEntry(t *testing.T, source dispatchcontract.DispatchSource) (string, string) {
	t.Helper()
	if source.AdHoc == nil || len(source.AdHoc.ProjectContext) != 1 {
		t.Fatalf("encoded source project context = %#v", source.AdHoc)
	}
	for key, value := range source.AdHoc.ProjectContext {
		return key, value
	}
	panic("unreachable")
}

func mustMarshalString(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return string(encoded)
}
