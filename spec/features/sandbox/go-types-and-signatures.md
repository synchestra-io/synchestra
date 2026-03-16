# Sandbox Go Types and Signatures

All Go type definitions, interface definitions, and function signatures for the sandbox feature. Extracted from the spec documents. **No implementation code — only types, signatures, and call graphs.**

**Sources:**
- [orchestrator-implementation-guide.md](orchestrator/implementation-guide.md)
- [agent-implementation-guide.md](agent/implementation-guide.md)
- [testing.md](observability/testing.md)
- [orchestrator.md](orchestrator/README.md)
- [http-api.md](orchestrator/http-api.md)
- [database-schema.md](orchestrator/database-schema.md)
- [monitoring.md](observability/README.md)

---

## 1. Package `internal/sandbox/orchestrator`

### Types

```go
// Config holds orchestrator configuration loaded from environment variables.
type Config struct {
    Image               string        // SYNCHESTRA_SANDBOX_IMAGE
    WorkspaceRoot       string        // SYNCHESTRA_SANDBOX_WORKSPACE_ROOT
    SocketDir           string        // SYNCHESTRA_SANDBOX_SOCKET_DIR
    DefaultMemoryMB     int           // SYNCHESTRA_SANDBOX_DEFAULT_MEMORY
    DefaultCPU          float64       // SYNCHESTRA_SANDBOX_DEFAULT_CPU
    DefaultDiskGB       int           // SYNCHESTRA_SANDBOX_DEFAULT_DISK
    IdleTimeout         time.Duration // SYNCHESTRA_SANDBOX_IDLE_TIMEOUT
    IdleCheckInterval   time.Duration // SYNCHESTRA_SANDBOX_IDLE_CHECK_INTERVAL
    HealthInterval      time.Duration // SYNCHESTRA_SANDBOX_HEALTH_INTERVAL
    HealthTimeout       time.Duration // SYNCHESTRA_SANDBOX_HEALTH_TIMEOUT
    MaxHealthFailures   int           // SYNCHESTRA_SANDBOX_MAX_HEALTH_FAILURES
    MaxRestarts         int           // SYNCHESTRA_SANDBOX_MAX_RESTARTS
    ProvisionTimeout    time.Duration // SYNCHESTRA_SANDBOX_PROVISION_TIMEOUT
    ResumeTimeout       time.Duration // SYNCHESTRA_SANDBOX_RESUME_TIMEOUT
    ShutdownTimeout     time.Duration // SYNCHESTRA_SANDBOX_SHUTDOWN_TIMEOUT
    MaxContainers       int           // SYNCHESTRA_SANDBOX_MAX_CONTAINERS
    MaxQueuedRequests   int           // SYNCHESTRA_SANDBOX_MAX_QUEUED_REQUESTS
    CacheTTL            time.Duration // SYNCHESTRA_SANDBOX_CACHE_TTL
    DiskCheckInterval   time.Duration // SYNCHESTRA_SANDBOX_DISK_CHECK_INTERVAL
    CircuitResetTimeout time.Duration
    LogLevel            string
}

// orchestrator is the concrete implementation of the Orchestrator interface.
type orchestrator struct {
    docker       client.APIClient           // Docker client (github.com/docker/docker/client)
    db           *sql.DB                    // Host database (metadata only)
    connPool     *ConnectionPool            // gRPC connection pool
    healthMgr    *HealthManager             // Health check goroutines
    idleMgr      *IdleManager               // Idle detection loop
    breakers     map[string]*CircuitBreaker  // Per-project circuit breakers
    containers   map[string]*containerState  // Per-project container state
    requestQueue map[string]chan pendingReq   // Per-project request queues
    config       *Config                     // Loaded configuration

    mu         sync.RWMutex   // Protects breakers, containers, and requestQueue maps
    wg         sync.WaitGroup // For graceful shutdown of background goroutines
    shutdownCh chan struct{}   // Signal shutdown to all goroutines
}

// containerState holds the in-memory state for a single project's container.
// Each instance has its own mutex for independent state transitions.
type containerState struct {
    projectID     string
    status        string         // See orchestrator.md for valid states
    containerID   string         // Docker container ID
    socketPath    string         // /var/run/synchestra/synchestra-{project_id}.sock
    stateRepoURL  string         // State repo URL from project registry lookup
    restartCount  int            // Persisted in sandbox_container_metadata.restart_count
    lastRestartAt time.Time      // Persisted in sandbox_container_metadata.last_restart_at

    mu      sync.Mutex    // Per-container lock for state transitions
    readyCh chan struct{} // Closed when container reaches "running"
}

// pendingReq represents a request waiting for a container to become ready.
type pendingReq struct {
    ctx    context.Context
    doneCh chan error // Signals when container is ready (nil) or failed (error)
}

// containerMeta holds persisted container metadata loaded from the database.
type containerMeta struct {
    ProjectID     string
    ContainerID   string
    SocketPath    string
    Status        string
    MemoryLimitMB int
    CPULimit      float64
    DiskQuotaGB   int
}

// ConnectionPool manages gRPC client connections to container agents, keyed by project ID.
type ConnectionPool struct {
    mu    sync.RWMutex
    conns map[string]*grpc.ClientConn
}

// HealthManager runs periodic health checks against all running containers.
type HealthManager struct {
    orchestrator *orchestrator
    interval     time.Duration
    timeout      time.Duration
    maxFailures  int
}

// IdleManager tracks session activity and auto-pauses idle containers.
type IdleManager struct {
    orchestrator  *orchestrator
    idleTimeout   time.Duration
    checkInterval time.Duration
}

// CircuitBreaker prevents repeated requests to a known-bad container.
// Per-project: failures in one container never affect others.
type CircuitBreaker struct {
    state       string        // "closed", "open", "half-open"
    failures    int
    maxFailures int
    resetAfter  time.Duration
    lastFailure time.Time
    mu          sync.Mutex
}

// Event is the envelope for orchestrator lifecycle events.
type Event struct {
    EventType string      `json:"event_type"` // e.g. "container.started", "session.completed"
    ProjectID string      `json:"project_id"`
    Timestamp time.Time   `json:"timestamp"`
    Payload   interface{} `json:"payload"`
}

// validTransitions defines the complete state machine.
var validTransitions = map[string][]string{
    "unprovisioned": {"creating"},
    "creating":      {"starting", "failed"},
    "starting":      {"running", "failed"},
    "running":       {"paused", "stopping", "failed"},
    "paused":        {"resuming"},
    "resuming":      {"running", "failed"},
    "stopping":      {"stopped"},
    "stopped":       {"starting", "terminated"},
    "failed":        {"starting", "terminated"},
    "terminated":    {"creating"},
}
```

### Sentinel Errors

```go
var (
    ErrNoConnection     = fmt.Errorf("no gRPC connection for project")
    ErrConnectionClosed = fmt.Errorf("gRPC connection is closed")
)
```

### Interfaces

```go
// Orchestrator manages sandbox container lifecycles and request routing.
// Single entry point used by HTTP API handlers.
type Orchestrator interface {
    // Provision ensures a container exists and is running for the project.
    Provision(ctx context.Context, projectID string) error

    // ExecuteCommand routes a command execution request to the project's container.
    ExecuteCommand(ctx context.Context, projectID string, req *pb.CommandRequest) (pb.SandboxAgent_ExecuteCommandClient, error)

    // GetStatus returns the current status of a project's container.
    GetStatus(ctx context.Context, projectID string) (*pb.StatusResponse, error)

    // ListSessions returns sessions for a project's container.
    ListSessions(ctx context.Context, projectID string, req *pb.ListSessionsRequest) (*pb.SessionList, error)

    // StreamLogs streams real-time logs from a session.
    StreamLogs(ctx context.Context, projectID string, req *pb.StreamLogsRequest) (pb.SandboxAgent_StreamLogsClient, error)

    // StoreCredential routes credential storage to the container's encrypted vault.
    StoreCredential(ctx context.Context, projectID string, req *pb.StoreCredentialRequest) (*pb.StoreCredentialResponse, error)

    // StopContainer gracefully stops a project's container.
    StopContainer(ctx context.Context, projectID string) error

    // DestroyContainer terminates and removes a project's container and workspace.
    DestroyContainer(ctx context.Context, projectID string) error

    // GetSessionDetails returns details for a specific session (for reconnection).
    GetSessionDetails(ctx context.Context, projectID, sessionID string) (*pb.SessionInfo, error)

    // Shutdown gracefully shuts down the orchestrator.
    Shutdown(ctx context.Context) error
}

// DockerClient defines the interface for container operations (testable via mocks).
// Source: testing.md
type DockerClient interface {
    CreateContainer(ctx context.Context, opts CreateOpts) (string, error)
    StartContainer(ctx context.Context, containerID string) error
    PauseContainer(ctx context.Context, containerID string) error
    UnpauseContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
    RemoveContainer(ctx context.Context, containerID string) error
    InspectContainer(ctx context.Context, containerID string) (ContainerInfo, error)
}

// EventEmitter publishes orchestrator events. Implementations must be non-blocking.
// Source: orchestrator.md
type EventEmitter interface {
    Emit(ctx context.Context, event Event) error
    Close() error
}
```

### Functions

#### Constructor & Shutdown

```go
// New creates and initializes an orchestrator instance.
// Creates Docker client, verifies daemon, starts health and idle managers.
func New(ctx context.Context, cfg *Config, db *sql.DB) (*orchestrator, error)

// Shutdown gracefully shuts down the orchestrator (stops goroutines, drains connections).
func (o *orchestrator) Shutdown(ctx context.Context) error
```

#### Core Lifecycle

```go
// ensureRunning ensures the container reaches "running" state.
// Handles all transitions transparently: auto-provision, auto-resume, auto-restart.
func (o *orchestrator) ensureRunning(ctx context.Context, projectID string) error

// transition validates and executes a state transition on a container.
// Calls the appropriate do* action, updates state and DB.
func (o *orchestrator) transition(ctx context.Context, cs *containerState, targetStatus string) error

// reconcileContainer checks Docker for actual state and updates DB if it differs.
// Called lazily on first request for each project.
func (o *orchestrator) reconcileContainer(ctx context.Context, projectID string) error

// waitForReady polls the agent's Ping() RPC until it succeeds or context expires.
func (o *orchestrator) waitForReady(ctx context.Context, cs *containerState) error

// isValidTransition checks whether a state transition is allowed by the state machine.
func isValidTransition(from, to string) bool
```

#### Transition Actions

```go
// doCreate pulls image, creates Docker container, upserts metadata in DB.
func (o *orchestrator) doCreate(ctx context.Context, cs *containerState) error

// doStart starts the Docker container and updates started_at timestamp.
func (o *orchestrator) doStart(ctx context.Context, cs *containerState) error

// doReady establishes gRPC connection and verifies agent responsiveness via Ping().
func (o *orchestrator) doReady(ctx context.Context, cs *containerState) error

// doPause closes gRPC connection and pauses the Docker container.
func (o *orchestrator) doPause(ctx context.Context, cs *containerState) error

// doResume unpauses the Docker container.
func (o *orchestrator) doResume(ctx context.Context, cs *containerState) error

// doStopping closes gRPC connection and sends SIGTERM to the Docker container.
func (o *orchestrator) doStopping(ctx context.Context, cs *containerState) error

// doStopped cleans up the stale Unix socket.
func (o *orchestrator) doStopped(ctx context.Context, cs *containerState) error

// doFailed closes gRPC connection, cleans up socket, records circuit breaker failure.
func (o *orchestrator) doFailed(ctx context.Context, cs *containerState) error

// doTerminated removes Docker container, cleans up socket, resets restart count.
func (o *orchestrator) doTerminated(ctx context.Context, cs *containerState) error
```

#### Request Routing (Orchestrator Interface Methods)

```go
// ExecuteCommand checks breaker, ensures running, forwards RPC, tracks session.
func (o *orchestrator) ExecuteCommand(ctx context.Context, projectID string, req *pb.CommandRequest) (pb.SandboxAgent_ExecuteCommandClient, error)

// GetStatus checks breaker, ensures running, forwards GetStatus RPC.
func (o *orchestrator) GetStatus(ctx context.Context, projectID string) (*pb.StatusResponse, error)

// StoreCredential checks breaker, ensures running, forwards StoreCredential RPC.
func (o *orchestrator) StoreCredential(ctx context.Context, projectID string, req *pb.StoreCredentialRequest) (*pb.StoreCredentialResponse, error)

// StopContainer transitions a running container to stopping → stopped.
func (o *orchestrator) StopContainer(ctx context.Context, projectID string) error

// DestroyContainer stops (if running) then terminates a container.
func (o *orchestrator) DestroyContainer(ctx context.Context, projectID string) error

// monitorSessionCompletion reads stream messages and calls TrackSessionEnd on command completion.
// Client disconnect (context cancellation) does NOT trigger TrackSessionEnd.
func (o *orchestrator) monitorSessionCompletion(projectID, sessionID string, stream pb.SandboxAgent_ExecuteCommandClient)
```

#### Helpers

```go
// getContainerState returns the in-memory container state for a project.
func (o *orchestrator) getContainerState(projectID string) *containerState

// getBreaker returns (or creates) the circuit breaker for a project.
func (o *orchestrator) getBreaker(projectID string) *CircuitBreaker

// loadContainerMeta loads container metadata from the database.
func (o *orchestrator) loadContainerMeta(ctx context.Context, projectID string) (*containerMeta, error)

// updateContainerStatus updates the container_status column in the database.
func (o *orchestrator) updateContainerStatus(ctx context.Context, projectID, status string)

// cleanupStaleSocket removes a stale Unix socket file.
func (o *orchestrator) cleanupStaleSocket(socketPath string)

// wrapError maps Go errors to gRPC status codes.
func wrapError(err error) error

// wrapGRPCError preserves gRPC status codes from the container agent.
func wrapGRPCError(err error) error
```

#### ConnectionPool Methods

```go
// NewConnectionPool creates an empty connection pool.
func NewConnectionPool() *ConnectionPool

// Get returns the gRPC connection for a project. Thread-safe for concurrent reads.
func (cp *ConnectionPool) Get(projectID string) (*grpc.ClientConn, error)

// Add dials the container's Unix socket and stores the connection.
func (cp *ConnectionPool) Add(projectID string, socketPath string) (*grpc.ClientConn, error)

// Remove closes and removes the connection for a project.
func (cp *ConnectionPool) Remove(projectID string) error

// CloseAll closes all connections. Called during orchestrator shutdown.
func (cp *ConnectionPool) CloseAll()

// Size returns the number of active connections. Used for metrics.
func (cp *ConnectionPool) Size() int
```

#### HealthManager Methods

```go
// NewHealthManager creates a health manager with the given check parameters.
func NewHealthManager(o *orchestrator, interval, timeout time.Duration, maxFailures int) *HealthManager

// Start runs the health check loop until stopCh is closed.
func (hm *HealthManager) Start(stopCh <-chan struct{})

// checkAllContainers snapshots running containers and checks each one.
func (hm *HealthManager) checkAllContainers()

// checkContainer performs a single health check via Ping() RPC.
func (hm *HealthManager) checkContainer(projectID string)

// recordFailure increments failure count in DB and triggers failed transition if threshold exceeded.
func (hm *HealthManager) recordFailure(projectID string)

// recordSuccess resets failure count in DB.
func (hm *HealthManager) recordSuccess(projectID string)
```

#### IdleManager Methods

```go
// NewIdleManager creates an idle manager with the given timeout parameters.
func NewIdleManager(o *orchestrator, idleTimeout, checkInterval time.Duration) *IdleManager

// Start runs the idle detection loop until stopCh is closed.
func (im *IdleManager) Start(stopCh <-chan struct{})

// checkIdleContainers queries DB for idle running containers and auto-pauses them.
func (im *IdleManager) checkIdleContainers()

// TrackSessionStart marks a container as active (clears idle_since).
func (im *IdleManager) TrackSessionStart(projectID string)

// TrackSessionEnd marks a container as potentially idle (sets idle_since = now).
func (im *IdleManager) TrackSessionEnd(projectID string)
```

#### CircuitBreaker Methods

```go
// NewCircuitBreaker creates a circuit breaker with the given failure threshold and reset duration.
func NewCircuitBreaker(maxFailures int, resetAfter time.Duration) *CircuitBreaker

// Allow returns true if a request should be permitted through the breaker.
func (cb *CircuitBreaker) Allow() bool

// RecordSuccess resets the breaker to closed state.
func (cb *CircuitBreaker) RecordSuccess()

// RecordFailure increments failure count and opens the breaker if threshold reached.
func (cb *CircuitBreaker) RecordFailure()

// State returns the current breaker state (for observability/logging).
func (cb *CircuitBreaker) State() string
```

---

## 2. Package `internal/sandbox/agent`

### Types

```go
// Session represents an isolated execution environment inside the container.
type Session struct {
    ID        string
    UserID    string
    Path      string           // /workspace/{project_id}/sessions/{session_id}/
    Cgroup    *Cgroup          // Optional cgroup v2 handle
    Metadata  *SessionMetadata
    CreatedAt time.Time
}

// SessionMetadata tracks session lifecycle information persisted to disk.
type SessionMetadata struct {
    SessionID      string
    UserID         string
    CreatedAt      time.Time
    Status         string // "pending", "running", "completed", "failed"
    MemoryUsed     int64
    PID            int
    ExitCode       int32
    RuntimeSeconds int64
}

// SessionResources holds resource usage metrics for a session.
type SessionResources struct {
    MemoryUsedBytes int64
    CPUPercent      float64
    DiskUsedBytes   int64
}

// CommandRequest holds the parameters for a command execution.
// (Mirrors the protobuf CommandRequest for internal use.)
type CommandRequest struct {
    SessionId      string
    UserId         string
    Command        []string
    WorkingDir     string
    EnvVars        map[string]string
    TimeoutSeconds int
    Credentials    []string
    StreamOutput   bool
}

// CommandExecution holds the result of a completed command.
type CommandExecution struct {
    Session       *Session
    ExitCode      int
    Runtime       int64
    Stdout        []byte
    Stderr        []byte
    ErrorMessage  string
    CompletedAt   time.Time
    OutputChannel chan OutputChunk // For streaming mode
}

// OutputChunk represents a chunk of streaming output.
type OutputChunk struct {
    Stdout []byte
    Stderr []byte
}

// StreamWriter captures and streams process output to gRPC clients.
type StreamWriter struct {
    sessionID     string
    stream        string // "stdout" or "stderr"
    ch            chan []byte
    sendFn        func([]byte) error
    buffer        []byte
    bufferMaxSize int
}

// SandboxAgentServer is the gRPC server implementation for the container agent.
type SandboxAgentServer struct {
    projectID       string
    sessionMgr      SessionManager
    credentialVault CredentialVault
    taskStateReader TaskStateReader
    executor        SubprocessExecutor
}

// CredentialVault (struct) is the concrete AES256-GCM encrypted credential store.
type CredentialVaultImpl struct {
    encryptionKey []byte    // 32 bytes for AES256
    storagePath   string    // /workspace/{project_id}/.secure/credentials.enc
    auditLog      *AuditLog
}

// CredentialStore is the top-level vault structure persisted to disk.
type CredentialStore struct {
    Credentials map[string]*Credential
}

// Credential is a single encrypted credential entry.
type Credential struct {
    Type       string         // "git_token", "ssh_key", "api_key", "env_var"
    Identifier string
    Ciphertext string         // Base64-encoded AES256-GCM ciphertext
    Nonce      string         // Base64-encoded 12-byte GCM nonce
    CreatedAt  int64          // Unix timestamp
    ExpiresAt  int64          // Unix timestamp (0 = no expiry)
    AccessLog  []AccessRecord
}

// CredentialMetadata is a non-secret summary of a credential (for listing).
type CredentialMetadata struct {
    Identifier string
    Type       string
    CreatedAt  int64
    ExpiresAt  int64
}

// AccessRecord logs a single credential access event.
type AccessRecord struct {
    UserID    string
    Timestamp time.Time
}

// AuditEvent is a structured entry for the credential audit log.
type AuditEvent struct {
    Event      string // "credential_stored", "credential_accessed", "credential_deleted"
    UserID     string
    Identifier string
    Type       string
    Timestamp  time.Time
}

// AuditLog writes credential access events to a structured log.
type AuditLog struct {
    // Internal fields (file handle, logger, etc.)
}
```

### Sentinel Errors

```go
var (
    ErrPathTraversal      = errors.New("path traversal detected")
    ErrCredentialNotFound = errors.New("credential not found")
    ErrCredentialExpired  = errors.New("credential expired")
    ErrSessionLimitReached = errors.New("session limit reached")
    ErrDiskQuotaExceeded  = errors.New("disk quota exceeded")
)
```

### Interfaces

```go
// SessionManager creates and tracks isolated execution environments.
type SessionManager interface {
    // CreateSession creates a new isolated session directory with cgroup limits.
    CreateSession(ctx context.Context, sessionID, userID string) (*Session, error)

    // GetSession retrieves an existing session by ID.
    GetSession(sessionID string) (*Session, error)

    // ListSessions returns all active sessions for a user.
    ListSessions(userID string) ([]*Session, error)

    // CleanupSession removes session state after the retention period.
    CleanupSession(sessionID string, retentionHours int) error

    // GetSessionResources returns resource usage metrics for a session.
    GetSessionResources(sessionID string) (*SessionResources, error)
}

// CredentialVault stores and retrieves encrypted credentials.
type CredentialVault interface {
    // Store encrypts and persists a credential (upsert semantics).
    Store(ctx context.Context, identifier string, credType string, value string, expiresAt time.Time) error

    // Get decrypts and returns a credential value. Checks expiry.
    Get(ctx context.Context, userID string, identifier string) (string, error)

    // Delete removes a credential from the vault.
    Delete(identifier string) error

    // List returns metadata (not values) for all stored credentials.
    List(userID string) ([]CredentialMetadata, error)
}

// SubprocessExecutor runs commands with isolation and resource limits.
type SubprocessExecutor interface {
    // Execute runs a command in the given session with timeout, env, and cgroup isolation.
    Execute(ctx context.Context, req *CommandRequest, session *Session) (*CommandExecution, error)
}

// TaskStateReader queries the local .synchestra/ git repo for task state.
type TaskStateReader interface {
    // GetState returns task state JSON, git ref, and any error.
    GetState(ctx context.Context, queryPath, gitRef string) ([]byte, string, error)
}
```

### Functions

#### Constructors

```go
// NewSessionManager creates a session manager rooted at the given workspace path.
func NewSessionManager(workspacePath string) *sessionManager

// NewCredentialVault creates a credential vault with the given encryption key and storage path.
func NewCredentialVault(encryptionKey []byte, storagePath string) *CredentialVaultImpl

// NewTaskStateReader creates a task state reader for the given .synchestra repo path.
func NewTaskStateReader(repoPath string) *taskStateReader

// NewSubprocessExecutor creates a subprocess executor.
func NewSubprocessExecutor() *subprocessExecutor
```

#### Session Manager

```go
// CreateSession creates an isolated session directory with secure permissions and optional cgroup.
func (s *sessionManager) CreateSession(ctx context.Context, sessionID, userID string) (*Session, error)

// GetSession retrieves an existing session by ID.
func (s *sessionManager) GetSession(sessionID string) (*Session, error)

// ListSessions returns all sessions for a user.
func (s *sessionManager) ListSessions(userID string) ([]*Session, error)

// CleanupSession removes session directory after the retention period.
func (s *sessionManager) CleanupSession(sessionID string, retentionHours int) error

// GetSessionResources returns resource usage for a session.
func (s *sessionManager) GetSessionResources(sessionID string) (*SessionResources, error)

// createCgroup creates a cgroup v2 for resource limits (non-fatal if unavailable).
func (s *sessionManager) createCgroup(sessionID string, memoryLimitMB int, cpuLimit float64) (*Cgroup, error)

// recordMetadata writes session metadata JSON to disk.
func recordMetadata(sessionPath string, metadata *SessionMetadata) error
```

#### Subprocess Executor

```go
// Execute runs a command with isolation, timeout, env merging, and output capture.
func (e *subprocessExecutor) Execute(ctx context.Context, req *CommandRequest, session *Session) (*CommandExecution, error)

// validateWorkingDir checks for path traversal in the working directory.
func validateWorkingDir(sessionPath, workingDir string) (string, error)

// mergeEnvironment merges inherited, provided, and session env vars.
func mergeEnvironment(inherited []string, provided map[string]string, sessionID, userID string) []string

// getExitCode extracts the exit code from a completed command.
func getExitCode(cmd *exec.Cmd, err error) int
```

#### Credential Vault

```go
// Store encrypts a credential with AES256-GCM and persists it to the vault file.
func (v *CredentialVaultImpl) Store(ctx context.Context, identifier, credType, value string, expiresAt time.Time) error

// Get decrypts and returns a credential value. Checks expiry. Records access.
func (v *CredentialVaultImpl) Get(ctx context.Context, userID, identifier string) (string, error)

// Delete removes a credential from the vault.
func (v *CredentialVaultImpl) Delete(identifier string) error

// List returns metadata for all stored credentials.
func (v *CredentialVaultImpl) List(userID string) ([]CredentialMetadata, error)

// loadVault reads and deserializes the vault file from disk.
func (v *CredentialVaultImpl) loadVault() (*CredentialStore, error)

// saveVault serializes and writes the vault file to disk.
func (v *CredentialVaultImpl) saveVault(vault *CredentialStore) error
```

#### Output Streaming

```go
// Write buffers output and flushes when buffer is full or a newline is encountered.
func (w *StreamWriter) Write(p []byte) (int, error)

// flush sends buffered data via channel or gRPC stream.
func (w *StreamWriter) flush() error

// NewTeeWriter creates a writer that tees output to a StreamWriter and a byte buffer.
func NewTeeWriter(sw *StreamWriter, maxBuffer int) *TeeWriter
```

#### gRPC Server

```go
// ExecuteCommand handles the ExecuteCommand RPC: creates session, runs command, streams output.
func (s *SandboxAgentServer) ExecuteCommand(req *pb.CommandRequest, stream pb.SandboxAgent_ExecuteCommandServer) error
```

#### Audit Log

```go
// Log writes an audit event to the structured audit log.
func (a *AuditLog) Log(event AuditEvent)
```

#### Error Mapping

```go
// toGRPCError maps agent-internal errors to gRPC status codes.
func toGRPCError(err error) error
```

#### Agent Entry Point

```go
// main initializes components, clones state repo, starts gRPC server on Unix socket.
func main()

// cloneStateRepo clones the state repository to the target path.
func cloneStateRepo(repoURL, targetPath string) error
```

---

## 3. Package `internal/api/sandbox`

### Package Structure

```
internal/api/sandbox/
├── handler.go           // HTTP handler registration, middleware chain
├── execute.go           // POST /execute handler
├── status.go            // GET /status handler
├── sessions.go          // GET /sessions, GET /sessions/{id} handlers
├── websocket.go         // WebSocket /sessions/{id}/logs handler
├── credentials.go       // POST /credentials, DELETE /credentials/{id}
├── destroy.go           // DELETE /{project_id}
├── admin.go             // All admin endpoint handlers
├── middleware.go         // Auth, rate limiting, request ID, logging
├── errors.go            // gRPC→HTTP error mapping
└── handler_test.go      // Handler tests (mock orchestrator)
```

### Types

```go
// Handler holds dependencies for all sandbox HTTP handlers.
type Handler struct {
    orchestrator orchestrator.Orchestrator
    // Additional dependencies: auth, rate limiter, logger
}

// ErrorMapping maps gRPC status codes to HTTP status codes and error code strings.
// Source: http-api.md
var grpcToHTTPStatus = map[codes.Code]int{
    codes.OK:                 200,
    codes.InvalidArgument:    400,
    codes.DeadlineExceeded:   504,
    codes.NotFound:           404,
    codes.PermissionDenied:   403,
    codes.ResourceExhausted:  429,
    codes.FailedPrecondition: 412,
    codes.Aborted:            409,
    codes.Internal:           500,
    codes.Unavailable:        503,
    codes.Unauthenticated:    401,
}
```

### Functions

#### Handler Registration

```go
// RegisterRoutes registers all sandbox HTTP routes on the given router.
func (h *Handler) RegisterRoutes(r chi.Router)
```

#### Sandbox Endpoint Handlers

```go
// HandleExecute handles POST /api/v1/sandbox/{project_id}/execute.
// Forwards command to orchestrator, returns session_id and stream_url.
func (h *Handler) HandleExecute(w http.ResponseWriter, r *http.Request)

// HandleGetStatus handles GET /api/v1/sandbox/{project_id}/status.
// Returns container status and resource usage.
func (h *Handler) HandleGetStatus(w http.ResponseWriter, r *http.Request)

// HandleListSessions handles GET /api/v1/sandbox/{project_id}/sessions.
// Returns paginated session list with optional filters.
func (h *Handler) HandleListSessions(w http.ResponseWriter, r *http.Request)

// HandleGetSession handles GET /api/v1/sandbox/{project_id}/sessions/{session_id}.
// Returns session details (primary reconnection endpoint).
func (h *Handler) HandleGetSession(w http.ResponseWriter, r *http.Request)

// HandleStreamLogs handles WebSocket /api/v1/sandbox/{project_id}/sessions/{session_id}/logs.
// Upgrades to WebSocket, proxies gRPC StreamLogs to client. Supports reconnection via ?since.
func (h *Handler) HandleStreamLogs(w http.ResponseWriter, r *http.Request)

// HandleStoreCredential handles POST /api/v1/sandbox/{project_id}/credentials.
// Stores an encrypted credential in the container vault.
func (h *Handler) HandleStoreCredential(w http.ResponseWriter, r *http.Request)

// HandleDeleteCredential handles DELETE /api/v1/sandbox/{project_id}/credentials/{identifier}.
// Deletes a credential from the container vault.
func (h *Handler) HandleDeleteCredential(w http.ResponseWriter, r *http.Request)

// HandleDestroyContainer handles DELETE /api/v1/sandbox/{project_id}.
// Destroys the container and optionally clears workspace cache.
func (h *Handler) HandleDestroyContainer(w http.ResponseWriter, r *http.Request)
```

#### Admin Endpoint Handlers

```go
// HandleAdminListContainers handles GET /api/v1/admin/sandbox/containers.
// Lists all containers across all projects.
func (h *Handler) HandleAdminListContainers(w http.ResponseWriter, r *http.Request)

// HandleAdminStopContainer handles POST /api/v1/admin/sandbox/{project_id}/stop.
// Force-stops a container (SIGTERM → SIGKILL).
func (h *Handler) HandleAdminStopContainer(w http.ResponseWriter, r *http.Request)

// HandleAdminRestartContainer handles POST /api/v1/admin/sandbox/{project_id}/restart.
// Force-restarts a container (stop + start).
func (h *Handler) HandleAdminRestartContainer(w http.ResponseWriter, r *http.Request)

// HandleAdminEvictContainer handles POST /api/v1/admin/sandbox/{project_id}/evict.
// Evicts a container from the active pool (preserves workspace cache).
func (h *Handler) HandleAdminEvictContainer(w http.ResponseWriter, r *http.Request)

// HandleAdminUpdateConfig handles PATCH /api/v1/admin/sandbox/{project_id}/config.
// Updates resource limits (effective on next restart).
func (h *Handler) HandleAdminUpdateConfig(w http.ResponseWriter, r *http.Request)

// HandleAdminUpdateImage handles PATCH /api/v1/admin/sandbox/{project_id}/image.
// Updates container image (effective on next restart).
func (h *Handler) HandleAdminUpdateImage(w http.ResponseWriter, r *http.Request)
```

#### Error Mapping

```go
// respondError maps a gRPC or orchestrator error to the appropriate HTTP status and JSON envelope.
func respondError(w http.ResponseWriter, err error)
```

---

## 4. Package `internal/sandbox/observability`

### Package Structure

```
internal/sandbox/observability/
├── metrics.go        // Prometheus metric definitions and registration
├── logging.go        // Structured logger setup with component context
├── tracing.go        // OpenTelemetry tracer initialization
├── alerts.go         // Alert rule definitions (for documentation/testing)
└── health.go         // Health check endpoint handlers
```

### Types

```go
// No custom struct types are defined in the spec. Metrics use prometheus.CounterVec,
// prometheus.GaugeVec, prometheus.HistogramVec from the Prometheus client library.
```

### Metric Constants

All host-side metrics use the `synchestra_sandbox_` prefix. Container-side metrics use `synchestra_agent_`.

#### Counters

| Metric | Labels |
|--------|--------|
| `synchestra_sandbox_containers_created_total` | — |
| `synchestra_sandbox_containers_failed_total` | — |
| `synchestra_sandbox_health_checks_total` | `result` |
| `synchestra_sandbox_requests_routed_total` | `status` |
| `synchestra_sandbox_auto_pauses_total` | — |
| `synchestra_sandbox_auto_resumes_total` | — |
| `synchestra_sandbox_evictions_total` | — |
| `synchestra_sandbox_commands_executed_total` | `project_id`, `status` |
| `synchestra_sandbox_credentials_stored_total` | `project_id` |
| `synchestra_sandbox_restarts_total` | `project_id` |
| `synchestra_sandbox_http_requests_total` | `method`, `endpoint`, `status_code` |
| `synchestra_sandbox_rate_limit_exceeded_total` | `endpoint` |

#### Gauges

| Metric | Labels |
|--------|--------|
| `synchestra_sandbox_containers_active` | `status` |
| `synchestra_sandbox_active_sessions` | — |
| `synchestra_sandbox_connection_pool_size` | — |
| `synchestra_sandbox_request_queue_depth` | `project_id` |
| `synchestra_sandbox_containers_total` | — |
| `synchestra_sandbox_disk_usage_bytes` | `project_id` |
| `synchestra_sandbox_memory_usage_bytes` | `project_id` |

#### Histograms

| Metric | Labels |
|--------|--------|
| `synchestra_sandbox_provision_duration_seconds` | — |
| `synchestra_sandbox_resume_duration_seconds` | — |
| `synchestra_sandbox_request_latency_seconds` | — |
| `synchestra_sandbox_health_check_duration_seconds` | — |
| `synchestra_sandbox_command_duration_seconds` | `project_id` |
| `synchestra_sandbox_http_request_duration_seconds` | `method`, `endpoint` |
| `synchestra_sandbox_websocket_connection_duration_seconds` | — |

### Functions

```go
// RegisterMetrics registers all synchestra_sandbox_* metrics with the Prometheus default registry.
func RegisterMetrics()

// NewLogger returns a slog.Logger pre-configured with JSON output, base fields, and component name.
func NewLogger(component string) *slog.Logger

// InitTracer initializes the OpenTelemetry TracerProvider with W3C propagation and OTLP exporter.
func InitTracer(ctx context.Context, serviceName string) (*trace.TracerProvider, error)

// HandleHealthz implements the /healthz liveness endpoint.
func HandleHealthz(w http.ResponseWriter, r *http.Request)

// HandleReadyz implements the /readyz readiness endpoint (Docker reachable, DB connected).
func HandleReadyz(w http.ResponseWriter, r *http.Request)

// HandleSandboxHealth implements the /healthz/sandbox subsystem health endpoint.
func HandleSandboxHealth(w http.ResponseWriter, r *http.Request)
```

---

## 5. Call Graph

### Command Execution Flow

```
HTTP POST /api/v1/sandbox/{project_id}/execute
  → handler.HandleExecute()
    → orchestrator.ExecuteCommand(ctx, projectID, req)
      → orchestrator.getBreaker(projectID)
      → breaker.Allow()
      → orchestrator.ensureRunning(ctx, projectID)
        → orchestrator.reconcileContainer(ctx, projectID)
          → orchestrator.loadContainerMeta(ctx, projectID)
          → docker.ContainerInspect()
          → orchestrator.updateContainerStatus()
          → connPool.Add()                    [if running, reconnect]
          → agent.Ping()                      [verify responsive]
          → connPool.Remove()                 [if unresponsive]
        → orchestrator.transition(ctx, cs, "creating")
          → isValidTransition()
          → orchestrator.doCreate(ctx, cs)
            → os.MkdirAll()                   [workspace dir]
            → orchestrator.loadContainerMeta() [resource limits]
            → docker.ContainerCreate()
            → db.ExecContext()                 [upsert metadata]
          → orchestrator.updateContainerStatus()
        → orchestrator.transition(ctx, cs, "starting")
          → orchestrator.doStart(ctx, cs)
            → docker.ContainerStart()
            → db.ExecContext()                 [update started_at]
        → orchestrator.waitForReady(ctx, cs)
          → connPool.Get() / connPool.Add()
          → agent.Ping()                       [poll until success]
        → orchestrator.transition(ctx, cs, "running")
          → orchestrator.doReady(ctx, cs)
            → connPool.Add()
            → agent.Ping()                     [verify]
      → connPool.Get(projectID)
      → agent.ExecuteCommand(ctx, req)         [gRPC stream]
      → idleMgr.TrackSessionStart(projectID)
      → go monitorSessionCompletion(projectID, sessionID, stream)
        → stream.Recv()                        [loop until completion]
        → idleMgr.TrackSessionEnd(projectID)   [on command completion only]
      → breaker.RecordSuccess()
```

### Container-Side Command Execution (gRPC Agent)

```
agent.ExecuteCommand(req, stream)
  → sessionMgr.CreateSession(ctx, sessionID, userID)
    → os.MkdirAll()                           [session directory]
    → createCgroup()                           [optional cgroup v2]
    → recordMetadata()                         [write metadata.json]
  → executor.Execute(ctx, req, session)
    → validateWorkingDir(session.Path, req.WorkingDir)
    → mergeEnvironment(os.Environ(), req.EnvVars, ...)
    → exec.CommandContext()
    → cmd.Start()
    → session.Cgroup.AddProcess(cmd.Process.Pid)
    → recordMetadata()                         [update PID, status=running]
    → cmd.Wait()
    → getExitCode(cmd, err)
    → recordMetadata()                         [update status=completed, exit code]
  → stream.Send(&CommandOutput{})              [stream output chunks]
  → stream.Send(&CommandOutput{Completed: true}) [final message]
  → sessionMgr.CleanupSession(sessionID, retentionHours)
```

### Credential Store Flow

```
HTTP POST /api/v1/sandbox/{project_id}/credentials
  → handler.HandleStoreCredential()
    → orchestrator.StoreCredential(ctx, projectID, req)
      → orchestrator.getBreaker(projectID)
      → breaker.Allow()
      → orchestrator.ensureRunning(ctx, projectID)     [auto-provision if needed]
      → connPool.Get(projectID)
      → agent.StoreCredential(ctx, req)                [gRPC]
        → credentialVault.Store(ctx, identifier, credType, value, expiresAt)
          → credentialVault.loadVault()                [read vault file]
          → rand.Read(nonce)                           [12-byte GCM nonce]
          → aes.NewCipher(encryptionKey)
          → cipher.NewGCM(block)
          → gcm.Seal()                                 [encrypt]
          → credentialVault.saveVault(vault)            [write vault file]
          → auditLog.Log(AuditEvent{Event: "credential_stored"})
      → breaker.RecordSuccess()
```

### Credential Retrieval (During Command Execution)

```
agent.Execute() [command with credential injection]
  → credentialVault.Get(ctx, userID, identifier)
    → credentialVault.loadVault()                     [read vault file]
    → check expiry (ExpiresAt vs now)
    → base64.StdEncoding.DecodeString(ciphertext)
    → base64.StdEncoding.DecodeString(nonce)
    → aes.NewCipher(encryptionKey)
    → cipher.NewGCM(block)
    → gcm.Open()                                      [decrypt]
    → record AccessRecord in cred.AccessLog
    → auditLog.Log(AuditEvent{Event: "credential_accessed"})
    → defer zero(plaintext)                            [clear from memory]
```

### Health Check Flow

```
healthMgr.Start(stopCh)
  → ticker (every HealthInterval)
    → healthMgr.checkAllContainers()
      → orchestrator.mu.RLock()                [snapshot running project IDs]
      → for each running projectID:
        → healthMgr.checkContainer(projectID)
          → connPool.Get(projectID)
          → agent.Ping(ctx)                    [gRPC with HealthTimeout]
          → ON SUCCESS:
            → healthMgr.recordSuccess(projectID)
              → db.ExecContext()               [reset health_check_failures = 0]
            → check resp.Status == "degraded"  [log warning]
          → ON FAILURE:
            → healthMgr.recordFailure(projectID)
              → db.ExecContext()               [increment health_check_failures]
              → db.QueryRowContext()            [read failure count]
              → IF failures >= maxFailures:
                → orchestrator.getContainerState(projectID)
                → orchestrator.transition(ctx, cs, "failed")
                  → orchestrator.doFailed(ctx, cs)
                    → connPool.Remove(projectID)
                    → cleanupStaleSocket(cs.socketPath)
                    → getBreaker(projectID).RecordFailure()
```

### Idle Detection Flow

```
idleMgr.Start(stopCh)
  → ticker (every IdleCheckInterval)
    → idleMgr.checkIdleContainers()
      → db.QueryContext()                      [SELECT project_id WHERE running AND idle long enough]
      → for each idle projectID:
        → orchestrator.getContainerState(projectID)
        → orchestrator.transition(ctx, cs, "paused")
          → orchestrator.doPause(ctx, cs)
            → connPool.Remove(projectID)
            → docker.ContainerPause(ctx, containerID)
            → db.ExecContext()                 [update paused_at]
```

### Auto-Resume Flow

```
HTTP request arrives for paused project
  → handler (any endpoint)
    → orchestrator.<Method>(ctx, projectID, ...)
      → orchestrator.ensureRunning(ctx, projectID)
        → reconcileContainer()                  [verify Docker state = paused]
        → cs.status == "paused"
        → orchestrator.transition(ctx, cs, "resuming")
          → orchestrator.doResume(ctx, cs)
            → docker.ContainerUnpause(ctx, containerID)
        → orchestrator.waitForReady(ctx, cs)
          → connPool.Add(projectID, socketPath)
          → agent.Ping()                        [poll until success, ResumeTimeout]
        → orchestrator.transition(ctx, cs, "running")
          → orchestrator.doReady(ctx, cs)
            → connPool.Add(projectID, socketPath)
            → agent.Ping()                      [verify]
```

### Graceful Shutdown Flow

```
orchestrator.Shutdown(ctx)
  → close(shutdownCh)                          [signal all goroutines]
  → wg.Wait()                                  [wait for healthMgr, idleMgr]
    → healthMgr.Start() returns                [stopCh closed]
    → idleMgr.Start() returns                  [stopCh closed]
  → connPool.CloseAll()                        [close all gRPC connections]
  → docker.Close()                             [close Docker client]
```

---

## Outstanding Questions

None at this time.
