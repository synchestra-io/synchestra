package state

// Features implemented: state-store, agent-coordination
// Features depended on:  state-store/topology

import "time"

// Cursor is the domain-level mirror of the replication package's ordered
// position (authority_epoch, sequence). It is redefined here — rather than
// importing pkg/state/replication — so pkg/state stays an interface-only,
// backend-neutral package per its own design principle; each backend's
// Agent() wiring adapts to/from replication.Cursor.
type Cursor struct {
	Epoch    int64
	Sequence int64
}

// IsZero reports whether the cursor has never been advanced.
func (c Cursor) IsZero() bool { return c.Epoch == 0 && c.Sequence == 0 }

// JournalEntry is a read-only, backend-neutral projection of one journal
// event, exposed for audit/inspection callers via JournalStore.
type JournalEntry struct {
	EventID       string
	Kind          string
	Cursor        Cursor
	CorrelationID string
	OccurredAt    time.Time
}

// --- Effort ---

// Effort is the durable unit of requested work. The exact original prompt
// lives in the private local Work Log (owned by Workbench) and is never
// stored here — WorkLogRef is a stable reference only, per
// spec/features/agent-coordination's private-prompt-content rule.
type Effort struct {
	ID             string
	ProjectID      string
	RepositoryID   string
	Title          string
	ScopeAreas     []string
	InitiatorRunID string
	WorkLogRef     string
	CreatedAt      time.Time
}

// EffortCreateParams holds parameters for creating a new effort.
type EffortCreateParams struct {
	ProjectID      string
	RepositoryID   string
	Title          string
	ScopeAreas     []string
	InitiatorRunID string
	WorkLogRef     string
}

// EffortFilter holds optional filters for listing efforts.
type EffortFilter struct {
	ProjectID    string
	RepositoryID string
}

// --- Run ---

// AgentFamily identifies the kind of actor executing a run.
type AgentFamily string

const (
	AgentFamilyCodex  AgentFamily = "codex"
	AgentFamilyClaude AgentFamily = "claude"
	AgentFamilyHuman  AgentFamily = "human"
)

// RunRole distinguishes the primary run of an effort from a subagent run
// spawned within it. A subagent is a run, not an invisible annotation, so
// communication and cost can be audited.
type RunRole string

const (
	RunRolePrimary  RunRole = "primary"
	RunRoleSubagent RunRole = "subagent"
)

// ModelProvenance labels how a non-null model identity was determined. A
// model value is never guessed: it is either observed from the runtime or
// declared by the caller.
type ModelProvenance string

const (
	ModelProvenanceRuntimeObserved ModelProvenance = "runtime_observed"
	ModelProvenanceCallerDeclared  ModelProvenance = "caller_declared"
)

// Run is one execution attempt within an effort. Model is nullable: when the
// runtime exposes no identity it remains nil rather than guessed, and every
// non-nil value carries a ModelProvenance.
type Run struct {
	ID              string
	EffortID        string
	AgentFamily     AgentFamily
	Model           *string
	ModelProvenance ModelProvenance
	Role            RunRole
	ParentRunID     string
	StartedAt       time.Time
	EndedAt         *time.Time
	TerminalReason  string
}

// RunStartParams holds parameters for starting a new run.
type RunStartParams struct {
	EffortID        string
	AgentFamily     AgentFamily
	Model           *string
	ModelProvenance ModelProvenance
	Role            RunRole
	ParentRunID     string
}

// RunFilter holds optional filters for listing runs.
type RunFilter struct {
	EffortID string
}

// RunCorrection appends an audited correction to a run's model identity —
// e.g. a runtime that later turns out to have exposed no reliable model ID,
// or a caller-declared value later proven wrong
// (agent-coordination#ac:optional-model-provenance-is-correctable). It is
// appended as a new event; it never rewrites the original run.started event.
type RunCorrection struct {
	Model           *string
	ModelProvenance ModelProvenance
	Reason          string
}

// --- Worktree claim ---

// LeaseFence proves the caller holds the current authority for a claim or
// lease. It maps to the replication authority epoch beneath it but never
// exposes replication package types directly to domain callers.
type LeaseFence struct {
	Epoch int64  // authority epoch this fence was issued under
	Token string // opaque per-lease fence token
}

// IsZero reports whether the fence has never been issued.
func (f LeaseFence) IsZero() bool { return f.Epoch == 0 && f.Token == "" }

// WorktreeClaim binds exactly one concurrent writer run to one canonical
// repository identity, local worktree location, local branch, target/base
// ref, base SHA, observed head SHA, and lease/fence token. It records Git
// facts and hashes, never credentials or arbitrary workspace contents.
type WorktreeClaim struct {
	ID           string
	ProjectID    string
	RepositoryID string
	RunID        string
	WorktreePath string
	Branch       string
	TargetRef    string
	BaseSHA      string
	HeadSHA      string
	ScopeAreas   []string
	Fence        LeaseFence
	ClaimedAt    time.Time
	RenewedAt    time.Time
	ReleasedAt   *time.Time
}

// WorktreeClaimParams holds parameters for claiming a worktree.
type WorktreeClaimParams struct {
	ProjectID    string
	RepositoryID string
	RunID        string
	WorktreePath string
	Branch       string
	TargetRef    string
	BaseSHA      string
	HeadSHA      string
	ScopeAreas   []string
}

// WorktreeFilter holds optional filters for listing worktree claims.
type WorktreeFilter struct {
	ProjectID    string
	RepositoryID string
	RunID        string
	ActiveOnly   bool // when true, excludes released claims
}

// --- Message ---

// Message is an immutable envelope linked to a sender run, recipient run(s),
// effort, and thread. The message body is private state; callers control
// what is persisted and how list views redact it.
type Message struct {
	ID              string
	EffortID        string
	ThreadID        string
	SenderRunID     string
	RecipientRunIDs []string
	CorrelationID   string
	RepositoryID    string
	ClaimID         string
	Body            string
	SentAt          time.Time
}

// MessageSendParams holds parameters for sending a message.
type MessageSendParams struct {
	EffortID        string
	ThreadID        string
	SenderRunID     string
	RecipientRunIDs []string
	CorrelationID   string
	RepositoryID    string
	ClaimID         string
	Body            string
}

// MessageAck is a recipient's acknowledgement of a message, recorded as a
// separate immutable record from the message itself.
type MessageAck struct {
	MessageID      string
	RecipientRunID string
	AcknowledgedAt time.Time
}

// --- Activity ---

// ActivityRecord is one entry in the append-only activity feed backing
// project/repository agent views.
type ActivityRecord struct {
	ID         string
	EffortID   string
	RunID      string
	Kind       string
	OccurredAt time.Time
	Detail     map[string]any
}

// ActivityRecordParams holds parameters for recording an activity entry.
type ActivityRecordParams struct {
	EffortID string
	RunID    string
	Kind     string
	Detail   map[string]any
}

// ActivityFilter holds optional filters for listing activity entries.
type ActivityFilter struct {
	EffortID string
	RunID    string
	Since    time.Time
}

// --- Authority lease ---

// AuthorityLease is a fenced, renewable hold on a named resource (e.g.
// "worktree:<repository>:<branch>"). It is the primitive WorktreeStore.Claim
// is built on.
type AuthorityLease struct {
	ID          string
	Resource    string
	HolderRunID string
	Fence       LeaseFence
	AcquiredAt  time.Time
	RenewedAt   time.Time
	ExpiresAt   time.Time
}

// LeaseAcquireParams holds parameters for acquiring an authority lease.
type LeaseAcquireParams struct {
	Resource    string
	HolderRunID string
	TTL         time.Duration
}

// --- Health ---

// HealthReport describes this endpoint's current authority/replication
// health, mirroring the backend-neutral fields defined by
// spec/features/state-store/topology without leaking payload content.
type HealthReport struct {
	EndpointID     string
	Role           string // "active" or "replica"
	AuthorityEpoch int64
	Cursor         Cursor
	ReplicaLag     int64
	LastOK         time.Time
	LastError      string
}
