package dispatchcontract

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker, model-selection

import "time"

// DispatchIntent is immutable after creation. Scheduler-owned lifecycle fields
// live on Dispatch instead.
type DispatchIntent struct {
	Source      DispatchSource     `json:"source" firestore:"source"`
	Repository  RepositorySnapshot `json:"repository" firestore:"repository"`
	Requested   RequestedExecution `json:"requested_execution" firestore:"requested_execution"`
	Constraints WorkerConstraints  `json:"worker_constraints,omitempty" firestore:"worker_constraints,omitempty"`
	Priority    int                `json:"priority,omitempty" firestore:"priority,omitempty"`
	NotBefore   *time.Time         `json:"not_before,omitempty" firestore:"not_before,omitempty"`
	RetryPolicy RetryPolicy        `json:"retry_policy,omitempty" firestore:"retry_policy,omitempty"`
}

// DispatchSource is a tagged union. Exactly one payload must match Kind.
type DispatchSource struct {
	Kind      SourceKind       `json:"kind" firestore:"kind"`
	AdHoc     *AdHocSource     `json:"ad_hoc,omitempty" firestore:"ad_hoc,omitempty"`
	SpecScore *SpecScoreSource `json:"specscore,omitempty" firestore:"specscore,omitempty"`
}

// AdHocSource carries a raw repository-scoped instruction.
type AdHocSource struct {
	Prompt         string            `json:"prompt" firestore:"prompt"`
	ProjectContext map[string]string `json:"project_context,omitempty" firestore:"project_context,omitempty"`
}

// SpecScoreSource identifies an immutable Plan or Task snapshot. SnapshotHash
// is a content digest; TargetRevision is the immutable Git revision containing
// that snapshot.
type SpecScoreSource struct {
	TargetKind     SpecScoreTargetKind `json:"target_kind" firestore:"target_kind"`
	TargetID       string              `json:"target_id" firestore:"target_id"`
	TargetPath     string              `json:"target_path,omitempty" firestore:"target_path,omitempty"`
	TargetRevision string              `json:"target_revision" firestore:"target_revision"`
	SnapshotHash   string              `json:"snapshot_hash" firestore:"snapshot_hash"`
	Instruction    string              `json:"instruction,omitempty" firestore:"instruction,omitempty"`
}

// RepositorySnapshot is resolved by the caller without changing its checkout.
// CloneURL never contains inline credentials; CredentialRef names a separately
// authorized secret reference when needed.
type RepositorySnapshot struct {
	CanonicalID   string `json:"canonical_id" firestore:"canonical_id"`
	CloneURL      string `json:"clone_url" firestore:"clone_url"`
	BaseRevision  string `json:"base_revision" firestore:"base_revision"`
	BaseRef       string `json:"base_ref,omitempty" firestore:"base_ref,omitempty"`
	Subdirectory  string `json:"subdirectory,omitempty" firestore:"subdirectory,omitempty"`
	ProjectID     string `json:"project_id,omitempty" firestore:"project_id,omitempty"`
	CredentialRef string `json:"credential_ref,omitempty" firestore:"credential_ref,omitempty"`
}

// RequestedExecution preserves caller intent even after routing.
type RequestedExecution struct {
	Profile       ExecutionProfile `json:"profile,omitempty" firestore:"profile,omitempty"`
	Agent         string           `json:"agent,omitempty" firestore:"agent,omitempty"`
	ModelSelector string           `json:"model_selector,omitempty" firestore:"model_selector,omitempty"`
	Effort        string           `json:"effort,omitempty" firestore:"effort,omitempty"`
	Fallback      FallbackPolicy   `json:"fallback,omitempty" firestore:"fallback,omitempty"`
}

// FallbackPolicy is explicit. An empty Mode is interpreted as reject.
type FallbackPolicy struct {
	Mode          FallbackMode `json:"mode,omitempty" firestore:"mode,omitempty"`
	AllowedModels []string     `json:"allowed_models,omitempty" firestore:"allowed_models,omitempty"`
	Reason        string       `json:"reason,omitempty" firestore:"reason,omitempty"`
}

// ResolvedExecution is immutable once an attempt is leased. It makes profile
// routing and adapter selection auditable.
type ResolvedExecution struct {
	Profile        ExecutionProfile `json:"profile" firestore:"profile"`
	Agent          string           `json:"agent" firestore:"agent"`
	Model          string           `json:"model" firestore:"model"`
	Effort         string           `json:"effort,omitempty" firestore:"effort,omitempty"`
	MappingVersion string           `json:"mapping_version" firestore:"mapping_version"`
	RoutingReason  string           `json:"routing_reason" firestore:"routing_reason"`
}

// WorkerConstraints are hard matching requirements. Empty fields are wildcards.
type WorkerConstraints struct {
	RunnerID             string            `json:"runner_id,omitempty" firestore:"runner_id,omitempty"`
	HostID               string            `json:"host_id,omitempty" firestore:"host_id,omitempty"`
	ProjectIDs           []string          `json:"project_ids,omitempty" firestore:"project_ids,omitempty"`
	RepositoryIDs        []string          `json:"repository_ids,omitempty" firestore:"repository_ids,omitempty"`
	RequiredCapabilities []string          `json:"required_capabilities,omitempty" firestore:"required_capabilities,omitempty"`
	RequiredLabels       map[string]string `json:"required_labels,omitempty" firestore:"required_labels,omitempty"`
}

// RetryPolicy retains attempt history; it never rewrites or deletes attempts.
type RetryPolicy struct {
	MaxAttempts    int `json:"max_attempts,omitempty" firestore:"max_attempts,omitempty"`
	BackoffSeconds int `json:"backoff_seconds,omitempty" firestore:"backoff_seconds,omitempty"`
}

// Dispatch is the durable aggregate. Intent fields remain immutable after
// CreateDispatch; scheduler fields change through versioned operations.
type Dispatch struct {
	ProtocolVersion string         `json:"protocol_version" firestore:"protocol_version"`
	ID              string         `json:"id" firestore:"-"`
	OwnerID         string         `json:"owner_id" firestore:"owner_id"`
	CreatedBy       string         `json:"created_by" firestore:"created_by"`
	IdempotencyKey  string         `json:"idempotency_key" firestore:"idempotency_key"`
	Intent          DispatchIntent `json:"intent" firestore:"intent"`
	Status          DispatchStatus `json:"status" firestore:"status"`
	AttemptIDs      []string       `json:"attempt_ids,omitempty" firestore:"attempt_ids,omitempty"`
	ActiveAttemptID string         `json:"active_attempt_id,omitempty" firestore:"active_attempt_id,omitempty"`
	Cancellation    *Cancellation  `json:"cancellation,omitempty" firestore:"cancellation,omitempty"`
	CreatedAt       time.Time      `json:"created_at" firestore:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" firestore:"updated_at"`
}

// Cancellation records durable caller intent and worker acknowledgement.
type Cancellation struct {
	RequestedAt    time.Time  `json:"requested_at" firestore:"requested_at"`
	RequestedBy    string     `json:"requested_by" firestore:"requested_by"`
	Reason         string     `json:"reason,omitempty" firestore:"reason,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty" firestore:"acknowledged_at,omitempty"`
}

// WorkerIdentity uses the registered host identity as the security principal;
// RunnerID optionally narrows the logical capacity pool on that host.
type WorkerIdentity struct {
	WorkerID string `json:"worker_id" firestore:"worker_id"`
	HostID   string `json:"host_id" firestore:"host_id"`
	RunnerID string `json:"runner_id,omitempty" firestore:"runner_id,omitempty"`
}

// AgentCapability advertises exact accepted profiles/models/effort values.
type AgentCapability struct {
	Agent    string             `json:"agent" firestore:"agent"`
	Profiles []ExecutionProfile `json:"profiles" firestore:"profiles"`
	Models   []string           `json:"models" firestore:"models"`
	Efforts  []string           `json:"efforts,omitempty" firestore:"efforts,omitempty"`
}

// WorkerCapabilities is sent on every claim so matching uses current state.
type WorkerCapabilities struct {
	Identity         WorkerIdentity    `json:"identity" firestore:"identity"`
	ProtocolVersions []string          `json:"protocol_versions" firestore:"protocol_versions"`
	Agents           []AgentCapability `json:"agents" firestore:"agents"`
	Capabilities     []string          `json:"capabilities,omitempty" firestore:"capabilities,omitempty"`
	ProjectIDs       []string          `json:"project_ids,omitempty" firestore:"project_ids,omitempty"`
	RepositoryIDs    []string          `json:"repository_ids,omitempty" firestore:"repository_ids,omitempty"`
	Labels           map[string]string `json:"labels,omitempty" firestore:"labels,omitempty"`
	MaxConcurrent    int               `json:"max_concurrent" firestore:"max_concurrent"`
	ActiveAttempts   int               `json:"active_attempts" firestore:"active_attempts"`
	Draining         bool              `json:"draining,omitempty" firestore:"draining,omitempty"`
}

// Lease is stored on an Attempt. Generation increases on reassignment. Worker
// authentication plus (attempt_id, owner.worker_id, generation) forms the
// ownership proof; no bearer secret is included in user-visible records.
type Lease struct {
	Owner           WorkerIdentity `json:"owner" firestore:"owner"`
	Generation      int64          `json:"generation" firestore:"generation"`
	AcquiredAt      time.Time      `json:"acquired_at" firestore:"acquired_at"`
	ExpiresAt       time.Time      `json:"expires_at" firestore:"expires_at"`
	LastHeartbeatAt time.Time      `json:"last_heartbeat_at" firestore:"last_heartbeat_at"`
}

// Attempt is one immutable history entry plus its current state. Resolution,
// lease owner, session, logs, failure, and result are never moved to another
// attempt during retry.
type Attempt struct {
	ProtocolVersion string              `json:"protocol_version" firestore:"protocol_version"`
	ID              string              `json:"id" firestore:"-"`
	DispatchID      string              `json:"dispatch_id" firestore:"dispatch_id"`
	Number          int                 `json:"number" firestore:"number"`
	Status          AttemptStatus       `json:"status" firestore:"status"`
	Requested       RequestedExecution  `json:"requested_execution" firestore:"requested_execution"`
	Resolved        *ResolvedExecution  `json:"resolved_execution,omitempty" firestore:"resolved_execution,omitempty"`
	Worker          *WorkerCapabilities `json:"worker,omitempty" firestore:"worker,omitempty"`
	Lease           *Lease              `json:"lease,omitempty" firestore:"lease,omitempty"`
	Session         *SessionReference   `json:"session,omitempty" firestore:"session,omitempty"`
	Logs            *LogReference       `json:"logs,omitempty" firestore:"logs,omitempty"`
	Result          *BranchResult       `json:"result,omitempty" firestore:"result,omitempty"`
	Failure         *TerminalFailure    `json:"failure,omitempty" firestore:"failure,omitempty"`
	CancellationAck *time.Time          `json:"cancellation_acknowledged_at,omitempty" firestore:"cancellation_acknowledged_at,omitempty"`
	CreatedAt       time.Time           `json:"created_at" firestore:"created_at"`
	StartedAt       *time.Time          `json:"started_at,omitempty" firestore:"started_at,omitempty"`
	FinishedAt      *time.Time          `json:"finished_at,omitempty" firestore:"finished_at,omitempty"`
}

// SessionReference ties one concrete agent process to one attempt.
type SessionReference struct {
	ID        string        `json:"id" firestore:"id"`
	Runtime   string        `json:"runtime" firestore:"runtime"`
	StartedAt time.Time     `json:"started_at" firestore:"started_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty" firestore:"ended_at,omitempty"`
	Logs      *LogReference `json:"logs,omitempty" firestore:"logs,omitempty"`
}

// LogReference points to the one canonical session/attempt stream. Dispatch
// records do not duplicate log text.
type LogReference struct {
	SessionID      string     `json:"session_id" firestore:"session_id"`
	StreamID       string     `json:"stream_id" firestore:"stream_id"`
	LastSequence   int64      `json:"last_sequence" firestore:"last_sequence"`
	Href           string     `json:"href,omitempty" firestore:"href,omitempty"`
	RetentionUntil *time.Time `json:"retention_until,omitempty" firestore:"retention_until,omitempty"`
}

// LogEvent is append-only. Messages must be redacted before submission.
type LogEvent struct {
	Sequence  int64     `json:"sequence" firestore:"sequence"`
	Timestamp time.Time `json:"timestamp" firestore:"timestamp"`
	Level     LogLevel  `json:"level" firestore:"level"`
	Stage     string    `json:"stage,omitempty" firestore:"stage,omitempty"`
	Message   string    `json:"message" firestore:"message"`
}

// TerminalFailure makes unsuccessful work durable and actionable.
type TerminalFailure struct {
	Stage     string        `json:"stage" firestore:"stage"`
	Code      string        `json:"code" firestore:"code"`
	Message   string        `json:"message" firestore:"message"`
	Retryable bool          `json:"retryable" firestore:"retryable"`
	Logs      *LogReference `json:"logs,omitempty" firestore:"logs,omitempty"`
}

// ValidationEvidence is a bounded summary, not raw command output.
type ValidationEvidence struct {
	Name           string           `json:"name" firestore:"name"`
	Command        string           `json:"command,omitempty" firestore:"command,omitempty"`
	Status         ValidationStatus `json:"status" firestore:"status"`
	ExitCode       int              `json:"exit_code" firestore:"exit_code"`
	DurationMillis int64            `json:"duration_ms,omitempty" firestore:"duration_ms,omitempty"`
	Summary        string           `json:"summary,omitempty" firestore:"summary,omitempty"`
	ArtifactRef    string           `json:"artifact_ref,omitempty" firestore:"artifact_ref,omitempty"`
}

// BranchResult is the only successful MVP publication result.
type BranchResult struct {
	RepositoryID string               `json:"repository_id" firestore:"repository_id"`
	BaseRevision string               `json:"base_revision" firestore:"base_revision"`
	Branch       string               `json:"branch" firestore:"branch"`
	Commit       string               `json:"commit" firestore:"commit"`
	Summary      string               `json:"summary" firestore:"summary"`
	Validation   []ValidationEvidence `json:"validation" firestore:"validation"`
	PublishedAt  time.Time            `json:"published_at" firestore:"published_at"`
}

// APIError is the stable JSON error response across CLI and worker operations.
type APIError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}
