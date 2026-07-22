package dispatchcontract

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker

import "time"

// CreateDispatchRequest is idempotent per authenticated owner and key.
type CreateDispatchRequest struct {
	ProtocolVersion string         `json:"protocol_version"`
	IdempotencyKey  string         `json:"idempotency_key"`
	CreatedBy       string         `json:"created_by"`
	Intent          DispatchIntent `json:"intent"`
}

type CreateDispatchResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
	Created         bool     `json:"created"`
}

type GetDispatchResponse struct {
	ProtocolVersion string    `json:"protocol_version"`
	Dispatch        Dispatch  `json:"dispatch"`
	Attempts        []Attempt `json:"attempts"`
}

type GetLogsResponse struct {
	ProtocolVersion string       `json:"protocol_version"`
	Reference       LogReference `json:"reference"`
	Events          []LogEvent   `json:"events"`
	NextCursor      int64        `json:"next_cursor"`
}

// CancelDispatchRequest is a caller operation, not an attempt-owner mutation.
type CancelDispatchRequest struct {
	ProtocolVersion string `json:"protocol_version"`
	DispatchID      string `json:"dispatch_id"`
	OperationID     string `json:"operation_id"`
	RequestedBy     string `json:"requested_by"`
	Reason          string `json:"reason,omitempty"`
}

type CancelDispatchResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
}

// RetryDispatchRequest creates a new Attempt and preserves all prior attempts.
type RetryDispatchRequest struct {
	ProtocolVersion string `json:"protocol_version"`
	DispatchID      string `json:"dispatch_id"`
	OperationID     string `json:"operation_id"`
	RequestedBy     string `json:"requested_by"`
	Reason          string `json:"reason,omitempty"`
}

type RetryDispatchResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
	Attempt         Attempt  `json:"attempt"`
}

// ClaimDispatchRequest is authenticated as Capabilities.Identity.HostID.
// RequestID makes a retried long-poll idempotent for that worker.
type ClaimDispatchRequest struct {
	ProtocolVersion string             `json:"protocol_version"`
	RequestID       string             `json:"request_id"`
	Capabilities    WorkerCapabilities `json:"capabilities"`
	LeaseSeconds    int                `json:"lease_seconds,omitempty"`
}

type ClaimAssignment struct {
	Dispatch Dispatch `json:"dispatch"`
	Attempt  Attempt  `json:"attempt"`
}

// ClaimDispatchResponse has a nil Assignment when no matching work is ready.
type ClaimDispatchResponse struct {
	ProtocolVersion   string           `json:"protocol_version"`
	Assignment        *ClaimAssignment `json:"assignment,omitempty"`
	RetryAfterSeconds int              `json:"retry_after_seconds,omitempty"`
}

// AttemptMutation identifies the current owner and lease generation. Every
// mutation is idempotent per OperationID.
type AttemptMutation struct {
	ProtocolVersion string `json:"protocol_version"`
	DispatchID      string `json:"dispatch_id"`
	AttemptID       string `json:"attempt_id"`
	WorkerID        string `json:"worker_id"`
	LeaseGeneration int64  `json:"lease_generation"`
	OperationID     string `json:"operation_id"`
}

type StartAttemptRequest struct {
	AttemptMutation
	Session SessionReference `json:"session"`
}

type StartAttemptResponse struct {
	ProtocolVersion string  `json:"protocol_version"`
	Attempt         Attempt `json:"attempt"`
}

type HeartbeatRequest struct {
	AttemptMutation
	ObservedAt  time.Time `json:"observed_at"`
	LogSequence int64     `json:"log_sequence,omitempty"`
	Stage       string    `json:"stage,omitempty"`
}

type HeartbeatResponse struct {
	ProtocolVersion string          `json:"protocol_version"`
	Directive       WorkerDirective `json:"directive"`
	LeaseExpiresAt  *time.Time      `json:"lease_expires_at,omitempty"`
}

type AppendLogsRequest struct {
	AttemptMutation
	Events []LogEvent `json:"events"`
}

type AppendLogsResponse struct {
	ProtocolVersion string       `json:"protocol_version"`
	Reference       LogReference `json:"reference"`
}

type CompleteAttemptRequest struct {
	AttemptMutation
	Result BranchResult `json:"result"`
}

type CompleteAttemptResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
	Attempt         Attempt  `json:"attempt"`
}

type FailAttemptRequest struct {
	AttemptMutation
	Failure TerminalFailure `json:"failure"`
}

type FailAttemptResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
	Attempt         Attempt  `json:"attempt"`
}

type AcknowledgeCancellationRequest struct {
	AttemptMutation
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

type AcknowledgeCancellationResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Dispatch        Dispatch `json:"dispatch"`
	Attempt         Attempt  `json:"attempt"`
}
