package dispatchcontract

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker, model-selection

import "fmt"

const (
	// ProtocolVersionV1 is the frozen Remote Task Dispatch MVP protocol.
	// All top-level requests and durable records carry this value.
	ProtocolVersionV1 = "synchestra.dispatch.v1"

	// CurrentProtocolVersion is the version emitted by this package.
	CurrentProtocolVersion = ProtocolVersionV1
)

// ProtocolCompatible reports whether a peer version can safely exchange MVP
// dispatch messages with this package. V1 is intentionally exact: additive
// changes remain v1, while a required-field or semantic change requires v2.
func ProtocolCompatible(version string) bool {
	return version == ProtocolVersionV1
}

// RequireCompatibleProtocol returns an actionable error for an incompatible
// peer. Callers should surface CodeIncompatibleProtocol at their API boundary.
func RequireCompatibleProtocol(version string) error {
	if ProtocolCompatible(version) {
		return nil
	}
	return fmt.Errorf("%s: got %q, support %q", CodeIncompatibleProtocol, version, ProtocolVersionV1)
}

// DispatchStatus is the aggregate status of durable work.
type DispatchStatus string

const (
	DispatchStatusQueued    DispatchStatus = "queued"
	DispatchStatusLeased    DispatchStatus = "leased"
	DispatchStatusRunning   DispatchStatus = "running"
	DispatchStatusCompleted DispatchStatus = "completed"
	DispatchStatusFailed    DispatchStatus = "failed"
	DispatchStatusCancelled DispatchStatus = "cancelled"
)

// AttemptStatus is the lifecycle of one claim/execution attempt.
type AttemptStatus string

const (
	AttemptStatusQueued    AttemptStatus = "queued"
	AttemptStatusLeased    AttemptStatus = "leased"
	AttemptStatusRunning   AttemptStatus = "running"
	AttemptStatusCompleted AttemptStatus = "completed"
	AttemptStatusFailed    AttemptStatus = "failed"
	AttemptStatusCancelled AttemptStatus = "cancelled"
	AttemptStatusAbandoned AttemptStatus = "abandoned"
)

// SourceKind distinguishes ad-hoc work from immutable SpecScore targets.
type SourceKind string

const (
	SourceKindAdHoc     SourceKind = "ad_hoc"
	SourceKindSpecScore SourceKind = "specscore"
)

// SpecScoreTargetKind is the supported target resource type.
type SpecScoreTargetKind string

const (
	SpecScoreTargetPlan SpecScoreTargetKind = "plan"
	SpecScoreTargetTask SpecScoreTargetKind = "task"
)

// ExecutionProfile is provider-neutral and stable across model mappings.
type ExecutionProfile string

const (
	ProfileFast     ExecutionProfile = "fast"
	ProfileBalanced ExecutionProfile = "balanced"
	ProfileLarge    ExecutionProfile = "large"
)

// FallbackMode controls exact-selector behavior. Reject is the default. A
// configured fallback is valid only with an explicit allow-list.
type FallbackMode string

const (
	FallbackReject     FallbackMode = "reject"
	FallbackConfigured FallbackMode = "configured"
)

// WorkerDirective tells an active owner what to do after a heartbeat.
type WorkerDirective string

const (
	WorkerDirectiveContinue      WorkerDirective = "continue"
	WorkerDirectiveCancel        WorkerDirective = "cancel"
	WorkerDirectiveOwnershipLost WorkerDirective = "ownership_lost"
)

// ValidationStatus is the outcome of one repository validation command.
type ValidationStatus string

const (
	ValidationPassed  ValidationStatus = "passed"
	ValidationFailed  ValidationStatus = "failed"
	ValidationSkipped ValidationStatus = "skipped"
)

// LogLevel is a worker log event severity.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Stable machine-readable error codes.
const (
	CodeIncompatibleProtocol = "INCOMPATIBLE_PROTOCOL"
	CodeInvalidRequest       = "INVALID_REQUEST"
	CodeConflict             = "CONFLICT"
	CodeNotFound             = "NOT_FOUND"
	CodeInvalidState         = "INVALID_STATE"
	CodeNoEligibleWorker     = "NO_ELIGIBLE_WORKER"
	CodeOwnershipLost        = "OWNERSHIP_LOST"
	CodeUnsupportedSelector  = "UNSUPPORTED_SELECTOR"
	CodeUnauthenticated      = "UNAUTHENTICATED"
)
