package dispatchcontract

// Features implemented: wb-session-transport
// Features depended on:  dispatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	// HandlerInvocationVersionV1 is the typed handler payload carried inside
	// the dispatch-v1 compatibility envelope.
	HandlerInvocationVersionV1 = "synchestra.handler-invocation.v1"

	// MaxHandlerPayloadBytes keeps the base64-encoded compatibility envelope
	// below the dispatch API's request-body bound.
	MaxHandlerPayloadBytes = 1 << 20

	handlerInvocationProjectContextKey = "synchestra.internal.handler_invocation.v1"
	handlerInvocationPrompt            = "Invoke a registered Synchestra handler."
	handlerExecutionAgent              = "synchestra-handler"
	handlerCapabilityPrefix            = "synchestra.handler:"
	handoffIdempotencyPrefix           = "wb-handoff:v1:sha256:"
	maxHandlerInvocationIDBytes        = 128
	maxWBHandoffIDBytes                = 1024
	maxHandlerInvocationEnvelopeBytes  = ((MaxHandlerPayloadBytes + 2) / 3 * 4) + 2048
)

var handlerInvocationID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// HandlerName identifies one code-registered handler. It is routing data, not
// an executable, subcommand, or argv fragment.
type HandlerName string

const (
	HandlerNameWBSessionAcceptV1  HandlerName = "wb.session.accept.v1"
	HandlerNameWBSessionMessageV1 HandlerName = "wb.session.message.v1"
)

// HandlerInvocation carries opaque bytes to a closed, code-registered handler.
// It intentionally has no executable, command, shell, argument, or environment
// fields. Operator configuration owns all process construction.
type HandlerInvocation struct {
	ProtocolVersion string      `json:"protocol_version"`
	ID              string      `json:"id"`
	Handler         HandlerName `json:"handler"`
	Payload         []byte      `json:"payload"`
	PayloadDigest   string      `json:"payload_digest"`
	PayloadSize     int64       `json:"payload_size"`
	CreatedAt       time.Time   `json:"created_at"`
	Deadline        *time.Time  `json:"deadline,omitempty"`
}

// NewHandlerInvocation constructs a canonical invocation and copies payload so
// later caller mutation cannot invalidate its digest or size evidence.
func NewHandlerInvocation(id string, handler HandlerName, payload []byte, createdAt time.Time, deadline *time.Time) (HandlerInvocation, error) {
	invocation := HandlerInvocation{
		ProtocolVersion: HandlerInvocationVersionV1,
		ID:              id,
		Handler:         handler,
		Payload:         append([]byte(nil), payload...),
		PayloadDigest:   HandlerPayloadDigest(payload),
		PayloadSize:     int64(len(payload)),
		CreatedAt:       createdAt.UTC(),
	}
	if deadline != nil {
		canonicalDeadline := deadline.UTC()
		invocation.Deadline = &canonicalDeadline
	}
	if err := invocation.Validate(); err != nil {
		return HandlerInvocation{}, err
	}
	return invocation, nil
}

// HandlerPayloadDigest returns the lower-case SHA-256 digest spelling used by
// handler invocation records.
func HandlerPayloadDigest(payload []byte) string {
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// IsSupportedHandler reports whether handler names one of the two fixed WB
// entry points shipped by the MVP.
func IsSupportedHandler(handler HandlerName) bool {
	switch handler {
	case HandlerNameWBSessionAcceptV1, HandlerNameWBSessionMessageV1:
		return true
	default:
		return false
	}
}

// Validate checks only transport-owned metadata and payload integrity. It does
// not inspect or interpret payload bytes.
func (i HandlerInvocation) Validate() error {
	if i.ProtocolVersion != HandlerInvocationVersionV1 {
		return errors.New("handler invocation uses an unsupported protocol version")
	}
	if len(i.ID) == 0 || len(i.ID) > maxHandlerInvocationIDBytes || !handlerInvocationID.MatchString(i.ID) {
		return errors.New("handler invocation ID is invalid")
	}
	if !IsSupportedHandler(i.Handler) {
		return errors.New("handler invocation names an unsupported handler")
	}
	if len(i.Payload) == 0 {
		return errors.New("handler invocation payload is empty")
	}
	if len(i.Payload) > MaxHandlerPayloadBytes {
		return errors.New("handler invocation payload exceeds the size limit")
	}
	if i.PayloadSize != int64(len(i.Payload)) {
		return errors.New("handler invocation payload size does not match")
	}
	if i.PayloadDigest != HandlerPayloadDigest(i.Payload) {
		return errors.New("handler invocation payload digest does not match")
	}
	if i.CreatedAt.IsZero() || !isUTC(i.CreatedAt) {
		return errors.New("handler invocation creation time must be canonical UTC")
	}
	if i.Deadline != nil {
		if i.Deadline.IsZero() || !isUTC(*i.Deadline) || !i.Deadline.After(i.CreatedAt) {
			return errors.New("handler invocation deadline must be canonical UTC and after creation")
		}
	}
	return nil
}

// EncodeHandlerInvocation places a validated invocation in the single reserved
// ad-hoc project-context slot used by dispatch-v1 compatibility peers.
func EncodeHandlerInvocation(invocation HandlerInvocation) (DispatchSource, error) {
	if err := invocation.Validate(); err != nil {
		return DispatchSource{}, err
	}
	encoded, err := json.Marshal(invocation)
	if err != nil {
		return DispatchSource{}, errors.New("encode handler invocation envelope")
	}
	return DispatchSource{
		Kind: SourceKindAdHoc,
		AdHoc: &AdHocSource{
			Prompt: handlerInvocationPrompt,
			ProjectContext: map[string]string{
				handlerInvocationProjectContextKey: string(encoded),
			},
		},
	}, nil
}

// ParseHandlerInvocation detects and validates the reserved dispatch-v1
// compatibility envelope. ok is false for an ordinary dispatch. Once the
// reserved key is present, malformed data is an invocation error and must not
// fall through to ordinary agent execution.
func ParseHandlerInvocation(source DispatchSource) (invocation HandlerInvocation, ok bool, err error) {
	if source.AdHoc == nil {
		return HandlerInvocation{}, false, nil
	}
	encoded, ok := source.AdHoc.ProjectContext[handlerInvocationProjectContextKey]
	if !ok {
		return HandlerInvocation{}, false, nil
	}
	if source.Kind != SourceKindAdHoc || source.SpecScore != nil || source.AdHoc.Prompt != handlerInvocationPrompt || len(source.AdHoc.ProjectContext) != 1 {
		return HandlerInvocation{}, true, errors.New("handler invocation envelope is invalid")
	}
	if len(encoded) == 0 || len(encoded) > maxHandlerInvocationEnvelopeBytes {
		return HandlerInvocation{}, true, errors.New("handler invocation envelope is invalid")
	}

	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&invocation); err != nil {
		return HandlerInvocation{}, true, errors.New("handler invocation envelope is invalid")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return HandlerInvocation{}, true, errors.New("handler invocation envelope is invalid")
	}
	if err := invocation.Validate(); err != nil {
		return HandlerInvocation{}, true, err
	}
	canonical, err := json.Marshal(invocation)
	if err != nil || !bytes.Equal(canonical, []byte(encoded)) {
		return HandlerInvocation{}, true, errors.New("handler invocation envelope is not canonical")
	}
	return invocation, true, nil
}

// HandlerRequestedExecution returns the exact synthetic scheduler selector for
// a registered handler. Request data cannot select a normal agent or model.
func HandlerRequestedExecution(handler HandlerName) (RequestedExecution, error) {
	if !IsSupportedHandler(handler) {
		return RequestedExecution{}, errors.New("cannot route an unsupported handler")
	}
	return RequestedExecution{
		Profile:       ProfileBalanced,
		Agent:         handlerExecutionAgent,
		ModelSelector: string(handler),
		Fallback:      FallbackPolicy{Mode: FallbackReject},
	}, nil
}

// HandlerRequiredCapability returns the hard worker capability corresponding
// to a registered handler.
func HandlerRequiredCapability(handler HandlerName) (string, error) {
	if !IsSupportedHandler(handler) {
		return "", errors.New("cannot advertise an unsupported handler")
	}
	return handlerCapabilityPrefix + string(handler), nil
}

// HandlerAgentCapability returns the synthetic scheduler advertisement for a
// worker's fixed handler registry. Handler order is preserved and duplicates
// are removed.
func HandlerAgentCapability(handlers ...HandlerName) (AgentCapability, error) {
	models := make([]string, 0, len(handlers))
	seen := make(map[HandlerName]struct{}, len(handlers))
	for _, handler := range handlers {
		if !IsSupportedHandler(handler) {
			return AgentCapability{}, errors.New("cannot advertise an unsupported handler")
		}
		if _, exists := seen[handler]; exists {
			continue
		}
		seen[handler] = struct{}{}
		models = append(models, string(handler))
	}
	if len(models) == 0 {
		return AgentCapability{}, errors.New("at least one registered handler is required")
	}
	return AgentCapability{
		Agent:    handlerExecutionAgent,
		Profiles: []ExecutionProfile{ProfileBalanced},
		Models:   models,
	}, nil
}

// WBHandoffIdempotencyKey derives a bounded, deterministic dispatch key from a
// WB-owned handoff ID without persisting or surfacing the raw ID in that key.
func WBHandoffIdempotencyKey(handoffID string) (string, error) {
	if strings.TrimSpace(handoffID) == "" || len(handoffID) > maxWBHandoffIDBytes {
		return "", errors.New("WB handoff ID is invalid")
	}
	digest := sha256.Sum256([]byte(handoffID))
	return handoffIdempotencyPrefix + hex.EncodeToString(digest[:]), nil
}

func isUTC(value time.Time) bool {
	_, offset := value.Zone()
	return offset == 0
}
