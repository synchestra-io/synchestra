# Container Orchestrator — Implementation Guide

## Overview

> **Related documents:** [orchestrator.md](README.md) (state machine and behavior spec — authoritative), [lifecycle.md](lifecycle.md) (lifecycle phases), [go-types-and-signatures.md](../go-types-and-signatures.md) (consolidated type definitions and call graph), [outstanding-questions.md](../outstanding-questions.md) (open design questions).

The Container Orchestrator manages the lifecycle of sandbox containers on the host side. It is initialized as part of `synchestra serve --http` and provides the bridge between HTTP API requests and gRPC container agents.

**Language**: Go  
**Location in repo**: `internal/sandbox/orchestrator/`  
**Integration point**: Called from HTTP API handlers in `internal/api/sandbox/`  
**Module**: `github.com/synchestra-io/synchestra`

This guide provides practical Go code patterns and examples for building the orchestrator service. It complements the [orchestrator spec](README.md) (state machine, configuration, metrics) and the [agent implementation guide](../agent/implementation-guide.md) (container-side agent).

**Related specs:**
- [orchestrator.md](README.md) — State machine, transitions, Docker config, metrics
- [protocol.md](../agent/README.md) — gRPC service definition (`SandboxAgent`)
- [database-schema.md](database-schema.md) — Host-side DB tables (`sandbox_container_metadata`, `sandbox_user_project_access`)
- [agent-implementation-guide.md](../agent/implementation-guide.md) — Container-side agent implementation

## Core Interface

The `Orchestrator` interface is the single entry point used by HTTP API handlers. Every method accepts a `projectID` and handles all lifecycle transitions transparently — callers never need to know whether a container is paused, stopped, or unprovisioned.

```go
package orchestrator

import (
    "context"

    pb "github.com/synchestra-io/synchestra/internal/sandbox/proto"
)

// Orchestrator manages sandbox container lifecycles and request routing.
type Orchestrator interface {
    // Provision ensures a container exists and is running for the project.
    // Auto-creates if unprovisioned, auto-resumes if paused, auto-starts if stopped.
    // Blocks until container is ready or context is cancelled.
    Provision(ctx context.Context, projectID string) error

    // ExecuteCommand routes a command execution request to the project's container.
    // Returns a stream of CommandOutput. Handles auto-provision/resume transparently.
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

    // GetSessionDetails returns details for a specific session in a project's container.
    // Used for session reconnection after client disconnect or host restart.
    GetSessionDetails(ctx context.Context, projectID, sessionID string) (*pb.SessionInfo, error)

    // Shutdown gracefully shuts down the orchestrator (drains connections, stops health checks).
    Shutdown(ctx context.Context) error
}
```

## Orchestrator Struct

The orchestrator struct is the concrete implementation. It owns all subsystems and coordinates their lifecycle.

```go
type orchestrator struct {
    docker       client.APIClient           // Docker client (github.com/docker/docker/client)
    db           *sql.DB                    // Host database (metadata only)
    connPool     *ConnectionPool            // gRPC connection pool
    healthMgr    *HealthManager             // Health check goroutines
    idleMgr      *IdleManager              // Idle detection loop
    breakers     map[string]*CircuitBreaker // Per-project circuit breakers
    containers   map[string]*containerState // Per-project container state
    requestQueue map[string]chan pendingReq  // Per-project request queues
    config       *Config                    // Loaded configuration

    mu         sync.RWMutex   // Protects breakers, containers, and requestQueue maps
    wg         sync.WaitGroup // For graceful shutdown of background goroutines
    shutdownCh chan struct{}   // Signal shutdown to all goroutines
}

// pendingReq represents a request waiting for a container to become ready.
type pendingReq struct {
    ctx    context.Context
    doneCh chan error // Signals when container is ready (nil) or failed (error)
}
```

**Design notes:**
- `breakers`, `containers`, and `requestQueue` are all keyed by `{project_id}`.
- `mu` protects map-level operations (adding/removing entries). Each `containerState` has its own mutex for state transitions.
- `wg` tracks background goroutines (health checks, idle scans) so `Shutdown` can wait for them.

## Initialization

### Constructor

```go
func New(ctx context.Context, cfg *Config, db *sql.DB) (*orchestrator, error) {
    // 1. Create Docker client
    dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("create docker client: %w", err)
    }

    // 2. Verify Docker daemon is reachable
    if _, err := dockerClient.Ping(ctx); err != nil {
        return nil, fmt.Errorf("docker ping: %w", err)
    }

    // 3. Ensure host directories exist
    for _, dir := range []string{cfg.WorkspaceRoot, cfg.SocketDir} {
        if err := os.MkdirAll(dir, 0750); err != nil {
            return nil, fmt.Errorf("create directory %s: %w", dir, err)
        }
    }

    o := &orchestrator{
        docker:       dockerClient,
        db:           db,
        connPool:     NewConnectionPool(),
        breakers:     make(map[string]*CircuitBreaker),
        containers:   make(map[string]*containerState),
        requestQueue: make(map[string]chan pendingReq),
        config:       cfg,
        shutdownCh:   make(chan struct{}),
    }

    // 4. Start health check manager
    o.healthMgr = NewHealthManager(o, cfg.HealthInterval, cfg.HealthTimeout, cfg.MaxHealthFailures)
    o.wg.Add(1)
    go func() {
        defer o.wg.Done()
        o.healthMgr.Start(o.shutdownCh)
    }()

    // 5. Start idle detection manager
    o.idleMgr = NewIdleManager(o, cfg.IdleTimeout, cfg.IdleCheckInterval)
    o.wg.Add(1)
    go func() {
        defer o.wg.Done()
        o.idleMgr.Start(o.shutdownCh)
    }()

    log.Infof("orchestrator initialized: %d containers recovered", len(o.containers))
    return o, nil
}
```

### Lazy Reconciliation

Instead of reconciling all containers at startup, the orchestrator reconciles Docker state lazily — on the first request for each project. This avoids a potentially slow startup sequence and ensures the orchestrator is ready to serve immediately.

> **Design note**: Event-based reconciliation (Docker event stream listener) and periodic reconciliation loop can be added later for proactive detection. The lazy approach is sufficient for MVP.

```go
// reconcileContainer checks Docker for the actual state of a project's container
// and updates the DB if it differs from the recorded state. Called from ensureRunning
// before attempting any state transitions.
func (o *orchestrator) reconcileContainer(ctx context.Context, projectID string) error {
    o.mu.RLock()
    cs, exists := o.containers[projectID]
    o.mu.RUnlock()

    if !exists {
        // No in-memory state — load from DB if available
        meta, err := o.loadContainerMeta(ctx, projectID)
        if err != nil {
            return nil // No DB record either — truly unprovisioned, nothing to reconcile
        }

        cs = &containerState{
            projectID:   meta.ProjectID,
            containerID: meta.ContainerID,
            socketPath:  meta.SocketPath,
            status:      meta.Status,
        }
        o.mu.Lock()
        o.containers[projectID] = cs
        o.mu.Unlock()
    }

    if cs.containerID == "" {
        // No Docker container was ever created; ensure marked as unprovisioned
        if cs.status != "unprovisioned" && cs.status != "terminated" {
            cs.status = "unprovisioned"
            o.updateContainerStatus(ctx, projectID, "unprovisioned")
        }
        return nil
    }

    // Check if Docker container still exists and its actual state
    inspect, err := o.docker.ContainerInspect(ctx, cs.containerID)
    if err != nil {
        // Container removed outside orchestrator control
        log.Warnf("container %s for project %s not found in Docker, marking stopped",
            cs.containerID, projectID)
        cs.status = "stopped"
        o.updateContainerStatus(ctx, projectID, "stopped")
        o.cleanupStaleSocket(cs.socketPath)
        return nil
    }

    // Compare actual Docker state with DB state and update if they differ
    var actualStatus string
    switch {
    case inspect.State.Running:
        actualStatus = "running"
    case inspect.State.Paused:
        actualStatus = "paused"
    default:
        actualStatus = "stopped"
    }

    if cs.status != actualStatus {
        log.Infof("reconcile project %s: DB says %s, Docker says %s — updating",
            projectID, cs.status, actualStatus)
        cs.status = actualStatus
        o.updateContainerStatus(ctx, projectID, actualStatus)

        // If running, try to re-establish gRPC connection
        if actualStatus == "running" {
            conn, err := o.connPool.Add(projectID, cs.socketPath)
            if err != nil {
                log.Warnf("cannot reconnect to %s during reconcile: %v, marking failed",
                    projectID, err)
                cs.status = "failed"
                o.updateContainerStatus(ctx, projectID, "failed")
            } else {
                // Verify agent is responsive
                pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                client := pb.NewSandboxAgentClient(conn)
                _, pingErr := client.Ping(pingCtx, &emptypb.Empty{})
                cancel()

                if pingErr != nil {
                    log.Warnf("agent unresponsive for %s during reconcile: %v, marking failed",
                        projectID, pingErr)
                    o.connPool.Remove(projectID)
                    cs.status = "failed"
                    o.updateContainerStatus(ctx, projectID, "failed")
                }
            }
        }
    }

    return nil
}

func (o *orchestrator) cleanupStaleSocket(socketPath string) {
    if socketPath == "" {
        return
    }
    if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
        log.Warnf("failed to remove stale socket %s: %v", socketPath, err)
    }
}

func (o *orchestrator) updateContainerStatus(ctx context.Context, projectID, status string) {
    _, err := o.db.ExecContext(ctx, `
        UPDATE sandbox_container_metadata
        SET container_status = $1, updated_at = NOW()
        WHERE project_id = $2
    `, status, projectID)
    if err != nil {
        log.Errorf("update container status for %s: %v", projectID, err)
    }
}
```

## Connection Pool Implementation

One gRPC connection per running container, keyed by `{project_id}`. Connections use Unix sockets — no TLS required.

```go
package orchestrator

import (
    "fmt"
    "sync"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/connectivity"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"
)

var (
    ErrNoConnection    = fmt.Errorf("no gRPC connection for project")
    ErrConnectionClosed = fmt.Errorf("gRPC connection is closed")
)

type ConnectionPool struct {
    mu    sync.RWMutex
    conns map[string]*grpc.ClientConn
}

func NewConnectionPool() *ConnectionPool {
    return &ConnectionPool{
        conns: make(map[string]*grpc.ClientConn),
    }
}

// Get returns the gRPC connection for a project. Thread-safe for concurrent reads.
func (cp *ConnectionPool) Get(projectID string) (*grpc.ClientConn, error) {
    cp.mu.RLock()
    defer cp.mu.RUnlock()

    conn, ok := cp.conns[projectID]
    if !ok {
        return nil, ErrNoConnection
    }

    // Reject connections in terminal states
    state := conn.GetState()
    if state == connectivity.Shutdown {
        return nil, ErrConnectionClosed
    }

    return conn, nil
}

// Add dials the container's Unix socket and stores the connection.
// Returns the new connection or an error if the dial fails.
func (cp *ConnectionPool) Add(projectID string, socketPath string) (*grpc.ClientConn, error) {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    // Close existing connection if present (idempotent re-add)
    if existing, ok := cp.conns[projectID]; ok {
        existing.Close()
        delete(cp.conns, projectID)
    }

    conn, err := grpc.NewClient(
        "unix://"+socketPath,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(10*1024*1024), // 10 MB
            grpc.MaxCallSendMsgSize(10*1024*1024),
        ),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             5 * time.Second,
            PermitWithoutStream: true,
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("dial %s: %w", socketPath, err)
    }

    cp.conns[projectID] = conn
    return conn, nil
}

// Remove closes and removes the connection for a project.
func (cp *ConnectionPool) Remove(projectID string) error {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    conn, ok := cp.conns[projectID]
    if !ok {
        return nil // Already removed; not an error
    }

    err := conn.Close()
    delete(cp.conns, projectID)
    return err
}

// CloseAll closes all connections. Called during orchestrator shutdown.
func (cp *ConnectionPool) CloseAll() {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    for projectID, conn := range cp.conns {
        if err := conn.Close(); err != nil {
            log.Warnf("close connection for %s: %v", projectID, err)
        }
    }
    cp.conns = make(map[string]*grpc.ClientConn)
}

// Size returns the number of active connections. Used for metrics.
func (cp *ConnectionPool) Size() int {
    cp.mu.RLock()
    defer cp.mu.RUnlock()
    return len(cp.conns)
}
```

## Lifecycle Manager

### Container State

Each project's container has its own state struct with a dedicated mutex. This avoids a global lock during state transitions — transitions for different projects are fully independent.

```go
type containerState struct {
    projectID     string
    status        string         // See orchestrator.md for valid states
    containerID   string         // Docker container ID
    socketPath    string         // /var/run/synchestra/synchestra-{project_id}.sock
    stateRepoURL  string         // State repo URL from project registry lookup
    restartCount  int            // Persisted in sandbox_container_metadata.restart_count
    lastRestartAt time.Time      // Persisted in sandbox_container_metadata.last_restart_at

    mu     sync.Mutex       // Per-container lock for state transitions
    readyCh chan struct{}    // Closed when container reaches "running"
}
```

> **Design note (restart persistence)**: Restart counts are persisted in the host database (`sandbox_container_metadata.restart_count`, `sandbox_container_metadata.last_restart_at`) to survive host process restarts. This prevents crash-looping containers from getting infinite restart attempts. On startup, `restartCount` and `lastRestartAt` are loaded from the database when a container's state is first accessed.

### State Machine Validation

```go
// validTransitions defines the complete state machine from orchestrator.md.
// Any transition not listed here is illegal and must be rejected.
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

func isValidTransition(from, to string) bool {
    targets, ok := validTransitions[from]
    if !ok {
        return false
    }
    for _, t := range targets {
        if t == to {
            return true
        }
    }
    return false
}
```

### State Transition Engine

```go
func (o *orchestrator) transition(ctx context.Context, cs *containerState, targetStatus string) error {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    // 1. Validate transition
    if !isValidTransition(cs.status, targetStatus) {
        return fmt.Errorf("invalid transition: %s → %s for project %s",
            cs.status, targetStatus, cs.projectID)
    }

    fromStatus := cs.status
    log.Infof("project %s: %s → %s", cs.projectID, fromStatus, targetStatus)

    // 2. Execute transition action
    var err error
    switch targetStatus {
    case "creating":
        err = o.doCreate(ctx, cs)
    case "starting":
        err = o.doStart(ctx, cs)
    case "running":
        err = o.doReady(ctx, cs)
    case "paused":
        err = o.doPause(ctx, cs)
    case "resuming":
        err = o.doResume(ctx, cs)
    case "stopping":
        err = o.doStopping(ctx, cs)
    case "stopped":
        err = o.doStopped(ctx, cs)
    case "failed":
        err = o.doFailed(ctx, cs)
    case "terminated":
        err = o.doTerminated(ctx, cs)
    default:
        return fmt.Errorf("unknown target status: %s", targetStatus)
    }

    if err != nil {
        log.Errorf("project %s: transition %s → %s failed: %v",
            cs.projectID, fromStatus, targetStatus, err)
        return err
    }

    // 3. Update state and DB
    cs.status = targetStatus
    o.updateContainerStatus(ctx, cs.projectID, targetStatus)

    // 4. If container is now running, signal waiting requests
    if targetStatus == "running" && cs.readyCh != nil {
        close(cs.readyCh)
        cs.readyCh = nil
    }

    return nil
}
```

### Transition Actions

```go
func (o *orchestrator) doCreate(ctx context.Context, cs *containerState) error {
    projectID := cs.projectID

    // Build workspace path and socket path
    workspacePath := filepath.Join(o.config.WorkspaceRoot, projectID)
    socketPath := filepath.Join(o.config.SocketDir, fmt.Sprintf("synchestra-%s.sock", projectID))

    // Ensure workspace directory exists
    if err := os.MkdirAll(workspacePath, 0750); err != nil {
        return fmt.Errorf("create workspace: %w", err)
    }

    // Load resource limits from DB (or use defaults)
    meta, err := o.loadContainerMeta(ctx, projectID)
    if err != nil {
        meta = &containerMeta{
            MemoryLimitMB: o.config.DefaultMemoryMB,
            CPULimit:      o.config.DefaultCPU,
            DiskQuotaGB:   o.config.DefaultDiskGB,
        }
    }

    memoryLimitBytes := int64(meta.MemoryLimitMB) * 1024 * 1024
    nanoCPUs := int64(meta.CPULimit * 1e9)
    pidsLimit := int64(256)

    // Create container via Docker API
    containerConfig := &container.Config{
        Image: o.config.Image,
        Env: []string{
            "SYNCHESTRA_PROJECT_ID=" + projectID,
            "SYNCHESTRA_STATE_REPO_URL=" + cs.stateRepoURL, // Obtained from project registry lookup
            "SYNCHESTRA_LOG_LEVEL=" + o.config.LogLevel,
        },
        User:         "1000:1000",
        Tty:          false,
        AttachStdout: false,
        AttachStderr: false,
        Healthcheck: &container.HealthConfig{
            Test:        []string{"CMD", "synchestra-sandbox-agent", "health"},
            Interval:    30 * time.Second,
            Timeout:     5 * time.Second,
            StartPeriod: 5 * time.Second,
            Retries:     3,
        },
    }

    hostConfig := &container.HostConfig{
        Binds: []string{
            workspacePath + ":/workspace/" + projectID + ":rw",
            // Mount only this project's socket directory (isolate from other containers)
            o.config.SocketDir + "/" + projectID + ":/var/run:rw",
        },
        Resources: container.Resources{
            Memory:    memoryLimitBytes,
            NanoCPUs:  nanoCPUs,
            PidsLimit: &pidsLimit,
        },
        CapDrop:        []string{"ALL"},
        ReadonlyRootfs: true,
        SecurityOpt:    []string{"no-new-privileges:true"},
        Tmpfs: map[string]string{
            "/tmp": "size=64m,noexec,nosuid",
            "/run": "size=16m,noexec,nosuid",
        },
        RestartPolicy: container.RestartPolicy{
            Name: "no", // Orchestrator manages restarts, not Docker
        },
        LogConfig: container.LogConfig{
            Type: "json-file",
            Config: map[string]string{
                "max-size": "10m",
                "max-file": "3",
            },
        },
    }

    containerName := fmt.Sprintf("synchestra-sandbox-%s", projectID)
    resp, err := o.docker.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
    if err != nil {
        return fmt.Errorf("docker create: %w", err)
    }

    cs.containerID = resp.ID
    cs.socketPath = socketPath

    // Upsert container metadata in DB
    _, err = o.db.ExecContext(ctx, `
        INSERT INTO sandbox_container_metadata (project_id, container_id, container_status, socket_path,
            memory_limit_mb, cpu_limit, resource_quota_gb, created_at, updated_at)
        VALUES ($1, $2, 'creating', $3, $4, $5, $6, NOW(), NOW())
        ON CONFLICT (project_id) DO UPDATE SET
            container_id = $2, container_status = 'creating', socket_path = $3, updated_at = NOW()
    `, projectID, resp.ID, socketPath, meta.MemoryLimitMB, meta.CPULimit, meta.DiskQuotaGB)
    if err != nil {
        return fmt.Errorf("upsert container metadata: %w", err)
    }

    return nil
}

func (o *orchestrator) doStart(ctx context.Context, cs *containerState) error {
    if err := o.docker.ContainerStart(ctx, cs.containerID, container.StartOptions{}); err != nil {
        return fmt.Errorf("docker start: %w", err)
    }

    // Update started_at timestamp
    _, err := o.db.ExecContext(ctx, `
        UPDATE sandbox_container_metadata
        SET started_at = NOW(), updated_at = NOW()
        WHERE project_id = $1
    `, cs.projectID)
    if err != nil {
        log.Warnf("update started_at for %s: %v", cs.projectID, err)
    }

    return nil
}

func (o *orchestrator) doReady(ctx context.Context, cs *containerState) error {
    // Establish gRPC connection to the agent
    conn, err := o.connPool.Add(cs.projectID, cs.socketPath)
    if err != nil {
        return fmt.Errorf("add connection: %w", err)
    }

    // Verify agent is responsive
    client := pb.NewSandboxAgentClient(conn)
    pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    _, err = client.Ping(pingCtx, &emptypb.Empty{})
    if err != nil {
        o.connPool.Remove(cs.projectID)
        return fmt.Errorf("agent ping failed: %w", err)
    }

    return nil
}

func (o *orchestrator) doPause(ctx context.Context, cs *containerState) error {
    // Close gRPC connection before pausing
    o.connPool.Remove(cs.projectID)

    if err := o.docker.ContainerPause(ctx, cs.containerID); err != nil {
        return fmt.Errorf("docker pause: %w", err)
    }

    _, err := o.db.ExecContext(ctx, `
        UPDATE sandbox_container_metadata
        SET paused_at = NOW(), updated_at = NOW()
        WHERE project_id = $1
    `, cs.projectID)
    if err != nil {
        log.Warnf("update paused_at for %s: %v", cs.projectID, err)
    }

    return nil
}

func (o *orchestrator) doResume(ctx context.Context, cs *containerState) error {
    if err := o.docker.ContainerUnpause(ctx, cs.containerID); err != nil {
        return fmt.Errorf("docker unpause: %w", err)
    }

    return nil
}

func (o *orchestrator) doStopping(ctx context.Context, cs *containerState) error {
    // Close gRPC connection
    o.connPool.Remove(cs.projectID)

    // Send SIGTERM with a grace period
    timeout := 30 // seconds
    stopOptions := container.StopOptions{Timeout: &timeout}
    if err := o.docker.ContainerStop(ctx, cs.containerID, stopOptions); err != nil {
        return fmt.Errorf("docker stop: %w", err)
    }

    return nil
}

func (o *orchestrator) doStopped(ctx context.Context, cs *containerState) error {
    o.cleanupStaleSocket(cs.socketPath)
    return nil
}

func (o *orchestrator) doFailed(ctx context.Context, cs *containerState) error {
    // Close gRPC connection if still open
    o.connPool.Remove(cs.projectID)
    o.cleanupStaleSocket(cs.socketPath)

    // Record failure for circuit breaker
    o.getBreaker(cs.projectID).RecordFailure()

    return nil
}

func (o *orchestrator) doTerminated(ctx context.Context, cs *containerState) error {
    // Remove Docker container
    o.connPool.Remove(cs.projectID)
    removeOpts := container.RemoveOptions{Force: true, RemoveVolumes: true}
    if err := o.docker.ContainerRemove(ctx, cs.containerID, removeOpts); err != nil {
        log.Warnf("docker remove %s: %v", cs.containerID, err)
    }

    o.cleanupStaleSocket(cs.socketPath)

    cs.containerID = ""
    cs.socketPath = ""
    cs.restartCount = 0

    return nil
}
```

### The `ensureRunning` Method

This is the central method called before routing any request. It ensures the container reaches `running` state, handling all transitions transparently.

> **Design note**: Synchronous provisioning — caller blocks until container is ready. Acceptable because container start + agent init typically completes in <10 seconds when the image is pre-pulled. Image pull (which can take longer) should be handled separately via warm pool or pre-pull.

> **Design note**: Uses a channel-based condition variable (`readyCh`) — first caller triggers creation, subsequent callers block on the channel. Chosen over `sync.Cond` for simpler API and natural integration with `select`/context cancellation.

```go
func (o *orchestrator) ensureRunning(ctx context.Context, projectID string) error {
    // 0. Reconcile Docker state for this project before attempting transitions
    if err := o.reconcileContainer(ctx, projectID); err != nil {
        log.Warnf("reconcile %s: %v", projectID, err)
        // Non-fatal — proceed with current state
    }

    o.mu.RLock()
    cs, exists := o.containers[projectID]
    o.mu.RUnlock()

    if !exists {
        // Create new container state for this project
        cs = &containerState{
            projectID: projectID,
            status:    "unprovisioned",
            readyCh:   make(chan struct{}),
        }
        o.mu.Lock()
        o.containers[projectID] = cs
        o.mu.Unlock()
    }

    // Fast path: already running
    if cs.status == "running" {
        return nil
    }

    // Determine transition path based on current state
    switch cs.status {
    case "running":
        return nil // Double-check after potential race

    case "unprovisioned", "terminated":
        // Full provision: creating → starting → running
        cs.readyCh = make(chan struct{})
        if err := o.transition(ctx, cs, "creating"); err != nil {
            return fmt.Errorf("provision %s: %w", projectID, err)
        }
        if err := o.transition(ctx, cs, "starting"); err != nil {
            return fmt.Errorf("start %s: %w", projectID, err)
        }
        if err := o.waitForReady(ctx, cs); err != nil {
            return fmt.Errorf("wait ready %s: %w", projectID, err)
        }
        return o.transition(ctx, cs, "running")

    case "stopped":
        // Re-start existing container: starting → running
        cs.readyCh = make(chan struct{})
        if err := o.transition(ctx, cs, "starting"); err != nil {
            return fmt.Errorf("start %s: %w", projectID, err)
        }
        if err := o.waitForReady(ctx, cs); err != nil {
            return fmt.Errorf("wait ready %s: %w", projectID, err)
        }
        return o.transition(ctx, cs, "running")

    case "paused":
        // Resume: resuming → running
        cs.readyCh = make(chan struct{})
        if err := o.transition(ctx, cs, "resuming"); err != nil {
            return fmt.Errorf("resume %s: %w", projectID, err)
        }
        if err := o.waitForReady(ctx, cs); err != nil {
            return fmt.Errorf("wait ready %s: %w", projectID, err)
        }
        return o.transition(ctx, cs, "running")

    case "failed":
        // Restart if allowed
        if cs.restartCount >= o.config.MaxRestarts {
            return fmt.Errorf("project %s: max restarts (%d) exceeded", projectID, o.config.MaxRestarts)
        }
        cs.restartCount++
        cs.lastRestartAt = time.Now()

        // Full re-provision: creating → starting → running
        cs.readyCh = make(chan struct{})
        if err := o.transition(ctx, cs, "starting"); err != nil {
            return fmt.Errorf("restart %s: %w", projectID, err)
        }
        if err := o.waitForReady(ctx, cs); err != nil {
            return fmt.Errorf("wait ready %s: %w", projectID, err)
        }
        return o.transition(ctx, cs, "running")

    case "creating", "starting", "resuming":
        // Already in a transitional state — wait for readyCh
        if cs.readyCh == nil {
            return fmt.Errorf("project %s is in %s state with no ready channel", projectID, cs.status)
        }
        select {
        case <-cs.readyCh:
            return nil
        case <-ctx.Done():
            return ctx.Err()
        }

    default:
        return fmt.Errorf("unexpected container state %q for project %s", cs.status, projectID)
    }
}

// waitForReady polls the agent's Ping() RPC until it succeeds or the context expires.
func (o *orchestrator) waitForReady(ctx context.Context, cs *containerState) error {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    timeout := o.config.ProvisionTimeout
    if cs.status == "resuming" {
        timeout = o.config.ResumeTimeout
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for agent readiness: %w", ctx.Err())
        case <-ticker.C:
            conn, err := o.connPool.Get(cs.projectID)
            if err != nil {
                // Try to establish connection (socket may not exist yet)
                conn, err = o.connPool.Add(cs.projectID, cs.socketPath)
                if err != nil {
                    continue // Socket not ready yet
                }
            }

            client := pb.NewSandboxAgentClient(conn)
            pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
            _, pingErr := client.Ping(pingCtx, &emptypb.Empty{})
            pingCancel()

            if pingErr == nil {
                return nil // Agent is ready
            }
        }
    }
}
```

## Health Manager

A single goroutine iterates all running containers on each tick. This avoids goroutine sprawl and simplifies shutdown.

> **Resolved**: Uses a single goroutine that iterates all running containers sequentially. Simpler shutdown semantics and lower resource usage. Health check latency grows linearly with container count — acceptable for expected scale (<100 containers per host).

```go
type HealthManager struct {
    orchestrator *orchestrator
    interval     time.Duration
    timeout      time.Duration
    maxFailures  int
}

func NewHealthManager(o *orchestrator, interval, timeout time.Duration, maxFailures int) *HealthManager {
    return &HealthManager{
        orchestrator: o,
        interval:     interval,
        timeout:      timeout,
        maxFailures:  maxFailures,
    }
}

// Start runs the health check loop until stopCh is closed.
func (hm *HealthManager) Start(stopCh <-chan struct{}) {
    ticker := time.NewTicker(hm.interval)
    defer ticker.Stop()

    for {
        select {
        case <-stopCh:
            log.Info("health manager stopped")
            return
        case <-ticker.C:
            hm.checkAllContainers()
        }
    }
}

func (hm *HealthManager) checkAllContainers() {
    o := hm.orchestrator
    o.mu.RLock()
    // Snapshot running container IDs to avoid holding the lock during RPCs
    var running []string
    for projectID, cs := range o.containers {
        if cs.status == "running" {
            running = append(running, projectID)
        }
    }
    o.mu.RUnlock()

    for _, projectID := range running {
        hm.checkContainer(projectID)
    }
}

func (hm *HealthManager) checkContainer(projectID string) {
    o := hm.orchestrator

    conn, err := o.connPool.Get(projectID)
    if err != nil {
        log.Warnf("health check %s: no connection: %v", projectID, err)
        hm.recordFailure(projectID)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
    defer cancel()

    client := pb.NewSandboxAgentClient(conn)
    resp, err := client.Ping(ctx, &emptypb.Empty{})
    if err != nil {
        log.Warnf("health check %s: ping failed: %v", projectID, err)
        hm.recordFailure(projectID)
        return
    }

    // Reset failure count on success
    hm.recordSuccess(projectID)

    if resp.GetStatus() == "degraded" {
        log.Warnf("health check %s: agent reports degraded status", projectID)
    }
}

func (hm *HealthManager) recordFailure(projectID string) {
    o := hm.orchestrator

    // Increment health_check_failures in DB
    _, err := o.db.ExecContext(context.Background(), `
        UPDATE sandbox_container_metadata
        SET health_check_failures = health_check_failures + 1,
            last_health_check = NOW(), updated_at = NOW()
        WHERE project_id = $1
    `, projectID)
    if err != nil {
        log.Errorf("update health failures for %s: %v", projectID, err)
    }

    // Check if threshold exceeded
    var failures int
    err = o.db.QueryRowContext(context.Background(), `
        SELECT health_check_failures FROM sandbox_container_metadata
        WHERE project_id = $1
    `, projectID).Scan(&failures)
    if err != nil {
        return
    }

    if failures >= hm.maxFailures {
        log.Errorf("project %s: %d consecutive health check failures, transitioning to failed",
            projectID, failures)
        cs := o.getContainerState(projectID)
        if cs != nil {
            o.transition(context.Background(), cs, "failed")
        }
    }
}

func (hm *HealthManager) recordSuccess(projectID string) {
    o := hm.orchestrator
    _, err := o.db.ExecContext(context.Background(), `
        UPDATE sandbox_container_metadata
        SET health_check_failures = 0, last_health_check = NOW(), updated_at = NOW()
        WHERE project_id = $1
    `, projectID)
    if err != nil {
        log.Errorf("reset health failures for %s: %v", projectID, err)
    }
}
```

## Idle Manager

Tracks session activity per container. When a container has no active sessions for longer than `IdleTimeout`, it is auto-paused.

```go
type IdleManager struct {
    orchestrator  *orchestrator
    idleTimeout   time.Duration
    checkInterval time.Duration
}

func NewIdleManager(o *orchestrator, idleTimeout, checkInterval time.Duration) *IdleManager {
    return &IdleManager{
        orchestrator:  o,
        idleTimeout:   idleTimeout,
        checkInterval: checkInterval,
    }
}

// Start runs the idle detection loop until stopCh is closed.
func (im *IdleManager) Start(stopCh <-chan struct{}) {
    ticker := time.NewTicker(im.checkInterval)
    defer ticker.Stop()

    for {
        select {
        case <-stopCh:
            log.Info("idle manager stopped")
            return
        case <-ticker.C:
            im.checkIdleContainers()
        }
    }
}

func (im *IdleManager) checkIdleContainers() {
    o := im.orchestrator
    rows, err := o.db.QueryContext(context.Background(), `
        SELECT project_id FROM sandbox_container_metadata
        WHERE container_status = 'running'
          AND idle_since IS NOT NULL
          AND idle_since < NOW() - $1::interval
    `, im.idleTimeout.String())
    if err != nil {
        log.Errorf("query idle containers: %v", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var projectID string
        if err := rows.Scan(&projectID); err != nil {
            log.Errorf("scan idle project: %v", err)
            continue
        }

        cs := o.getContainerState(projectID)
        if cs == nil || cs.status != "running" {
            continue
        }

        log.Infof("auto-pausing idle container for project %s", projectID)
        if err := o.transition(context.Background(), cs, "paused"); err != nil {
            log.Errorf("auto-pause %s: %v", projectID, err)
        }
    }
}

// TrackSessionStart marks a container as active (clears idle_since).
// Called when a new session begins (e.g., ExecuteCommand).
func (im *IdleManager) TrackSessionStart(projectID string) {
    _, err := im.orchestrator.db.ExecContext(context.Background(), `
        UPDATE sandbox_container_metadata
        SET idle_since = NULL, updated_at = NOW()
        WHERE project_id = $1 AND container_status = 'running'
    `, projectID)
    if err != nil {
        log.Errorf("track session start for %s: %v", projectID, err)
    }
}

// TrackSessionEnd marks a container as potentially idle (sets idle_since = now).
// Called when a session completes. The idle manager will check later whether
// the container has been idle long enough to pause.
func (im *IdleManager) TrackSessionEnd(projectID string) {
    _, err := im.orchestrator.db.ExecContext(context.Background(), `
        UPDATE sandbox_container_metadata
        SET idle_since = NOW(), updated_at = NOW()
        WHERE project_id = $1 AND container_status = 'running'
          AND idle_since IS NULL
    `, projectID)
    if err != nil {
        log.Errorf("track session end for %s: %v", projectID, err)
    }
}
```

## Circuit Breaker

Per-project circuit breaker prevents repeated requests to a known-bad container. Failures in one container never affect others.

```go
type CircuitBreaker struct {
    state       string        // "closed", "open", "half-open"
    failures    int
    maxFailures int
    resetAfter  time.Duration
    lastFailure time.Time
    mu          sync.Mutex
}

func NewCircuitBreaker(maxFailures int, resetAfter time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state:       "closed",
        maxFailures: maxFailures,
        resetAfter:  resetAfter,
    }
}

// Allow returns true if a request should be permitted through the breaker.
func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case "closed":
        return true
    case "open":
        if time.Since(cb.lastFailure) >= cb.resetAfter {
            cb.state = "half-open"
            return true // Allow one probe request
        }
        return false
    case "half-open":
        return false // Only one probe at a time; reject until probe resolves
    }
    return false
}

// RecordSuccess resets the breaker to closed state.
func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.state = "closed"
    cb.failures = 0
}

// RecordFailure increments failure count and opens the breaker if threshold reached.
func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailure = time.Now()

    if cb.failures >= cb.maxFailures {
        cb.state = "open"
    }
}

// State returns the current breaker state (for observability/logging).
func (cb *CircuitBreaker) State() string {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    return cb.state
}

// getBreaker returns (or creates) the circuit breaker for a project.
func (o *orchestrator) getBreaker(projectID string) *CircuitBreaker {
    o.mu.RLock()
    cb, ok := o.breakers[projectID]
    o.mu.RUnlock()

    if ok {
        return cb
    }

    // Create new breaker under write lock
    o.mu.Lock()
    defer o.mu.Unlock()

    // Double-check after acquiring write lock
    if cb, ok := o.breakers[projectID]; ok {
        return cb
    }

    cb = NewCircuitBreaker(o.config.MaxHealthFailures, o.config.CircuitResetTimeout)
    o.breakers[projectID] = cb
    return cb
}
```

## Request Routing with Auto-Provision

Every API method follows the same pattern: check breaker → ensure running → get connection → forward RPC → track session.

```go
func (o *orchestrator) ExecuteCommand(ctx context.Context, projectID string, req *pb.CommandRequest) (pb.SandboxAgent_ExecuteCommandClient, error) {
    // 1. Check circuit breaker
    breaker := o.getBreaker(projectID)
    if !breaker.Allow() {
        return nil, status.Error(codes.Unavailable, "container temporarily unavailable (circuit open)")
    }

    // 2. Ensure container is running (auto-provision/resume)
    if err := o.ensureRunning(ctx, projectID); err != nil {
        breaker.RecordFailure()
        return nil, wrapError(err)
    }

    // 3. Get gRPC connection
    conn, err := o.connPool.Get(projectID)
    if err != nil {
        breaker.RecordFailure()
        return nil, status.Error(codes.Unavailable, "no connection to container agent")
    }

    // 4. Forward RPC to container agent
    client := pb.NewSandboxAgentClient(conn)
    stream, err := client.ExecuteCommand(ctx, req)
    if err != nil {
        breaker.RecordFailure()
        return nil, wrapGRPCError(err)
    }

    // 5. Track session activity (for idle detection)
    //    NOTE: TrackSessionEnd is NOT called on client disconnect (gRPC stream cancellation).
    //    Sessions are persistent — the command continues in the container.
    //    TrackSessionEnd is only called when the container-side command completes.
    //    See "Session Reconnection" subsection for details.
    o.idleMgr.TrackSessionStart(projectID)

    // 6. Monitor stream completion in background
    go o.monitorSessionCompletion(projectID, req.SessionId, stream)

    breaker.RecordSuccess()
    return stream, nil
}

// monitorSessionCompletion runs in a goroutine per active stream.
// It reads stream messages and calls TrackSessionEnd when the command completes.
// Client disconnect (context cancellation) does NOT trigger TrackSessionEnd.
func (o *orchestrator) monitorSessionCompletion(projectID, sessionID string, stream pb.SandboxAgent_ExecuteCommandClient) {
    for {
        msg, err := stream.Recv()
        if err != nil {
            // Stream ended — check if it was a normal completion or client disconnect
            if status.Code(err) == codes.Canceled {
                // Client disconnected — session still active in container
                return
            }
            // Stream error or EOF — session completed or failed
            o.idleMgr.TrackSessionEnd(projectID)
            return
        }
        if msg.Completed {
            o.idleMgr.TrackSessionEnd(projectID)
            return
        }
    }
}

func (o *orchestrator) GetStatus(ctx context.Context, projectID string) (*pb.StatusResponse, error) {
    breaker := o.getBreaker(projectID)
    if !breaker.Allow() {
        return nil, status.Error(codes.Unavailable, "container temporarily unavailable (circuit open)")
    }

    if err := o.ensureRunning(ctx, projectID); err != nil {
        breaker.RecordFailure()
        return nil, wrapError(err)
    }

    conn, err := o.connPool.Get(projectID)
    if err != nil {
        breaker.RecordFailure()
        return nil, status.Error(codes.Unavailable, "no connection to container agent")
    }

    client := pb.NewSandboxAgentClient(conn)
    resp, err := client.GetStatus(ctx, &pb.StatusRequest{})
    if err != nil {
        breaker.RecordFailure()
        return nil, wrapGRPCError(err)
    }

    breaker.RecordSuccess()
    return resp, nil
}

func (o *orchestrator) StoreCredential(ctx context.Context, projectID string, req *pb.StoreCredentialRequest) (*pb.StoreCredentialResponse, error) {
    breaker := o.getBreaker(projectID)
    if !breaker.Allow() {
        return nil, status.Error(codes.Unavailable, "container temporarily unavailable (circuit open)")
    }

    if err := o.ensureRunning(ctx, projectID); err != nil {
        breaker.RecordFailure()
        return nil, wrapError(err)
    }

    conn, err := o.connPool.Get(projectID)
    if err != nil {
        breaker.RecordFailure()
        return nil, status.Error(codes.Unavailable, "no connection to container agent")
    }

    client := pb.NewSandboxAgentClient(conn)
    resp, err := client.StoreCredential(ctx, req)
    if err != nil {
        breaker.RecordFailure()
        return nil, wrapGRPCError(err)
    }

    breaker.RecordSuccess()
    return resp, nil
}

func (o *orchestrator) StopContainer(ctx context.Context, projectID string) error {
    cs := o.getContainerState(projectID)
    if cs == nil {
        return status.Error(codes.NotFound, "no container for project")
    }

    if cs.status != "running" {
        return status.Errorf(codes.FailedPrecondition, "container is %s, not running", cs.status)
    }

    if err := o.transition(ctx, cs, "stopping"); err != nil {
        return wrapError(err)
    }
    return o.transition(ctx, cs, "stopped")
}

func (o *orchestrator) DestroyContainer(ctx context.Context, projectID string) error {
    cs := o.getContainerState(projectID)
    if cs == nil {
        return status.Error(codes.NotFound, "no container for project")
    }

    // Stop first if running
    if cs.status == "running" {
        if err := o.transition(ctx, cs, "stopping"); err != nil {
            return wrapError(err)
        }
        if err := o.transition(ctx, cs, "stopped"); err != nil {
            return wrapError(err)
        }
    }

    return o.transition(ctx, cs, "terminated")
}
```

### Helper: Container State Lookup

```go
func (o *orchestrator) getContainerState(projectID string) *containerState {
    o.mu.RLock()
    defer o.mu.RUnlock()
    return o.containers[projectID]
}
```

### Session Reconnection

Sessions are persistent and reconnectable — they survive client disconnects, host restarts, and even host+container migration to a different machine.

**Design principle**: The container is the sole source of truth for session state. The host stores no session data and never caches session metadata. Every reconnection queries the container directly. This means reconnection works as long as the host can reach the container, regardless of whether the host process or machine has changed.

**Reconnection scenarios:**

| Scenario | Session Survives? | Mechanism |
|----------|-------------------|-----------|
| Client disconnects (tab closed, network drop) | ✅ Yes | Command continues in container; client re-attaches via `StreamLogs(session_id)` |
| Client reconnects from different device | ✅ Yes | Same user + session_id, routed to same container |
| Host process restarts | ✅ Yes | Lazy reconciliation re-discovers container, re-establishes gRPC |
| Host migrates to different machine | ✅ Yes | New host connects to container (if reachable via socket/network) |
| Container restarts (volume preserved) | ⚠️ Partial | Completed session logs survive on volume; running commands lost |
| Container terminated + workspace destroyed | ❌ No | All session data lost; user starts new session |

**Reconnection flow (implementation):**

```go
// Client reconnects → HTTP API handler:
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
    projectID := chi.URLParam(r, "project_id")
    sessionID := chi.URLParam(r, "session_id")
    
    // 1. Ensure container is running (auto-resume if paused)
    if err := h.orchestrator.Provision(r.Context(), projectID); err != nil {
        // Container unreachable — session may be lost
        respondError(w, err)
        return
    }
    
    // 2. Query container for session state (container is source of truth)
    resp, err := h.orchestrator.GetSessionDetails(r.Context(), projectID, sessionID)
    // ... return session status, logs, exit code
}

// For live re-attachment to a running command:
func (h *Handler) StreamSessionLogs(w http.ResponseWriter, r *http.Request) {
    // 1. Ensure container running
    // 2. Call StreamLogs(session_id) on container
    // 3. Proxy gRPC stream → WebSocket/SSE to client
    // Works identically whether this is the original client or a reconnection
}
```

**Implementation impact on `ExecuteCommand`**: The orchestrator calls `TrackSessionStart` when the RPC is initiated, but must NOT call `TrackSessionEnd` when the gRPC stream context is cancelled by a client disconnect. Instead, `TrackSessionEnd` is called only when the container agent signals command completion (e.g., via a session completion callback or status poll).

**Idle detection**: Based on container-side session state (active commands), not client connections. A disconnected client with a running command keeps the container marked as "active" — which is correct, since the command is still consuming resources.

> **Note**: Session logs are written to the workspace volume (`/workspace/{project_id}/sessions/{session_id}/logs/`), ensuring they survive container restarts as long as the volume is preserved. This is the mechanism that enables partial recovery after container restart.

### Future Enhancement: Warm Pool

Pre-create N containers with no project assignment, ready to be claimed on first request.

**How it works:**
- A background goroutine maintains a pool of N warm containers (configurable).
- On first request for a project: assign a warm container instead of creating from scratch.
- The warm container gets configured with the project's state repo URL and credentials at assignment time.

**Benefits:**
- Reduces first-request latency from ~10s (container create + start + agent init) to <1s (assign + configure).
- Trade-off: idle resource consumption vs. latency.

> **Not implemented in initial version.** The synchronous provisioning path (~10s with pre-pulled image) is acceptable for MVP. Warm pool can be added later when first-request latency becomes a user-facing concern.

## Graceful Shutdown

```go
func (o *orchestrator) Shutdown(ctx context.Context) error {
    log.Info("orchestrator shutting down")

    // 1. Signal all background goroutines to stop
    close(o.shutdownCh)

    // 2. Wait for background goroutines (health checks, idle manager)
    //    with a timeout from the context
    done := make(chan struct{})
    go func() {
        o.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Info("all background goroutines stopped")
    case <-ctx.Done():
        log.Warn("shutdown timed out waiting for background goroutines")
    }

    // 3. Close all gRPC connections
    o.connPool.CloseAll()

    // 4. Close Docker client
    if err := o.docker.Close(); err != nil {
        log.Warnf("close docker client: %v", err)
    }

    log.Info("orchestrator shutdown complete")
    return nil
}
```

## Error Handling Patterns

### Error → gRPC Status Code Mapping

Orchestrator-level errors are mapped to gRPC status codes which the HTTP API gateway translates to HTTP status codes.

| Error Scenario | gRPC Code | HTTP Status | Retry? |
|---|---|---|---|
| Container provisioning failed | `INTERNAL` | 500 | Yes (with backoff) |
| Container paused, resume in progress | `UNAVAILABLE` | 503 | Yes (auto-handled) |
| Circuit breaker open | `UNAVAILABLE` | 503 | Yes (after reset timeout) |
| Project not found | `NOT_FOUND` | 404 | No |
| User has no access | `PERMISSION_DENIED` | 403 | No |
| Request queue full | `RESOURCE_EXHAUSTED` | 429 | Yes (with backoff) |
| Provision timeout | `DEADLINE_EXCEEDED` | 504 | Yes |
| Container terminated | `UNAVAILABLE` | 503 | Yes (auto-re-provision) |
| Invalid state transition | `FAILED_PRECONDITION` | 400 | No |
| Max restarts exceeded | `FAILED_PRECONDITION` | 400 | No (manual intervention) |

### Error Wrapping Helpers

```go
func wrapError(err error) error {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return status.Error(codes.DeadlineExceeded, "operation timed out")
    case errors.Is(err, context.Canceled):
        return status.Error(codes.Cancelled, "operation cancelled")
    default:
        return status.Error(codes.Internal, err.Error())
    }
}

func wrapGRPCError(err error) error {
    // Preserve gRPC status codes from the container agent
    if st, ok := status.FromError(err); ok {
        return st.Err()
    }
    return status.Error(codes.Internal, err.Error())
}
```

## Testing Strategy

### Unit Tests: State Machine

```go
func TestValidTransitions(t *testing.T) {
    tests := []struct {
        from    string
        to      string
        allowed bool
    }{
        {"unprovisioned", "creating", true},
        {"creating", "starting", true},
        {"creating", "failed", true},
        {"starting", "running", true},
        {"running", "paused", true},
        {"running", "stopping", true},
        {"running", "failed", true},
        {"paused", "resuming", true},
        {"terminated", "creating", true},

        // Invalid transitions
        {"unprovisioned", "running", false},
        {"paused", "stopped", false},
        {"running", "creating", false},
        {"terminated", "running", false},
        {"stopped", "paused", false},
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("%s→%s", tt.from, tt.to), func(t *testing.T) {
            result := isValidTransition(tt.from, tt.to)
            if result != tt.allowed {
                t.Errorf("isValidTransition(%q, %q) = %v, want %v",
                    tt.from, tt.to, result, tt.allowed)
            }
        })
    }
}
```

### Unit Tests: Circuit Breaker

```go
func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
    cb := NewCircuitBreaker(3, 30*time.Second)

    // Closed state allows requests
    assert.True(t, cb.Allow())
    assert.Equal(t, "closed", cb.State())

    // Record failures below threshold
    cb.RecordFailure()
    cb.RecordFailure()
    assert.True(t, cb.Allow()) // Still closed

    // Third failure trips the breaker
    cb.RecordFailure()
    assert.False(t, cb.Allow()) // Open
    assert.Equal(t, "open", cb.State())
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
    cb := NewCircuitBreaker(1, 100*time.Millisecond)

    cb.RecordFailure() // Opens breaker
    assert.False(t, cb.Allow())

    // Wait for reset timeout
    time.Sleep(150 * time.Millisecond)

    // Should transition to half-open, allowing one probe
    assert.True(t, cb.Allow())
    assert.Equal(t, "half-open", cb.State())

    // Second request in half-open is rejected
    assert.False(t, cb.Allow())
}

func TestCircuitBreaker_SuccessResets(t *testing.T) {
    cb := NewCircuitBreaker(1, 100*time.Millisecond)

    cb.RecordFailure() // Open
    time.Sleep(150 * time.Millisecond)
    cb.Allow() // half-open

    cb.RecordSuccess() // Back to closed
    assert.Equal(t, "closed", cb.State())
    assert.True(t, cb.Allow())
}
```

### Unit Tests: Connection Pool

```go
func TestConnectionPool_GetMissing(t *testing.T) {
    pool := NewConnectionPool()
    _, err := pool.Get("nonexistent-project")
    assert.ErrorIs(t, err, ErrNoConnection)
}

func TestConnectionPool_AddAndGet(t *testing.T) {
    pool := NewConnectionPool()

    // Create a test Unix socket
    socketPath := filepath.Join(t.TempDir(), "test.sock")
    lis, err := net.Listen("unix", socketPath)
    require.NoError(t, err)
    defer lis.Close()

    conn, err := pool.Add("project-1", socketPath)
    require.NoError(t, err)
    assert.NotNil(t, conn)

    got, err := pool.Get("project-1")
    require.NoError(t, err)
    assert.Equal(t, conn, got)

    assert.Equal(t, 1, pool.Size())
}

func TestConnectionPool_RemoveIdempotent(t *testing.T) {
    pool := NewConnectionPool()
    // Removing a non-existent entry should not error
    assert.NoError(t, pool.Remove("nonexistent-project"))
}
```

### Integration Tests

Integration tests require Docker and use real containers. Gate them behind a build tag.

```go
//go:build integration

func TestProvisionAndPing(t *testing.T) {
    ctx := context.Background()
    cfg := testConfig(t)
    db := testDB(t)

    orch, err := New(ctx, cfg, db)
    require.NoError(t, err)
    defer orch.Shutdown(ctx)

    // Provision should create and start a container
    err = orch.Provision(ctx, "test-project-1")
    require.NoError(t, err)

    // GetStatus should succeed
    resp, err := orch.GetStatus(ctx, "test-project-1")
    require.NoError(t, err)
    assert.Equal(t, "ok", resp.GetStatus())

    // Cleanup
    err = orch.DestroyContainer(ctx, "test-project-1")
    require.NoError(t, err)
}
```

### Concurrency Tests

```go
func TestConcurrentEnsureRunning(t *testing.T) {
    // Verify that concurrent requests for the same project
    // result in exactly one container creation.
    o := newTestOrchestrator(t)

    var wg sync.WaitGroup
    errs := make([]error, 10)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            errs[idx] = o.ensureRunning(context.Background(), "project-concurrent")
        }(i)
    }

    wg.Wait()

    // All should succeed
    for i, err := range errs {
        assert.NoError(t, err, "goroutine %d failed", i)
    }

    // Exactly one container should exist
    cs := o.getContainerState("project-concurrent")
    assert.Equal(t, "running", cs.status)
}
```

### Test Approach Summary

| Layer | Approach | Dependencies |
|---|---|---|
| State machine | Table-driven tests for `isValidTransition` | None |
| Circuit breaker | Time-based tests with short timeouts | None |
| Connection pool | Real Unix sockets in `t.TempDir()` | None (no Docker) |
| Lifecycle transitions | Mock Docker client (`github.com/docker/docker/client` interface), mock gRPC | `testify/mock` |
| Integration | Real Docker containers with build tag `integration` | Docker daemon, test image |
| Concurrency | `sync.WaitGroup` + parallel `ensureRunning` | Mock Docker |

## Go Package Structure

```
internal/sandbox/orchestrator/
├── orchestrator.go          // Orchestrator interface, struct, New(), Shutdown()
├── lifecycle.go             // State machine, transitions, ensureRunning, Docker API calls
├── connection_pool.go       // gRPC connection pool (Add, Get, Remove, CloseAll)
├── health.go                // HealthManager: health check loop, failure detection
├── idle.go                  // IdleManager: idle detection, auto-pause, session tracking
├── circuit_breaker.go       // Per-project CircuitBreaker (closed/open/half-open)
├── router.go                // Request routing methods (ExecuteCommand, GetStatus, etc.)
├── config.go                // Config struct, LoadFromEnv() — reads SYNCHESTRA_SANDBOX_* vars
├── errors.go                // Error wrapping helpers, gRPC status code mapping
├── metrics.go               // Prometheus metrics (future, see orchestrator.md)
├── orchestrator_test.go     // Unit tests: state machine, circuit breaker, connection pool
├── lifecycle_test.go        // Unit tests: transitions with mock Docker client
├── health_test.go           // Unit tests: health check logic with mock gRPC
├── idle_test.go             // Unit tests: idle detection timing
└── integration_test.go      // Integration tests (build tag: integration)
```

## Outstanding Questions

None at this time.
