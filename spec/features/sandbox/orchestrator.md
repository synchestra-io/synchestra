# Container Orchestrator

## Overview

The Container Orchestrator is the host-side service component responsible for managing the lifecycle of sandbox containers, maintaining gRPC connections to container agents, performing health monitoring, and routing requests from the HTTP API layer to the appropriate container. It runs as part of `synchestra serve --http`.

**Design principle**: The orchestrator is a stateless router and lifecycle manager. It stores only container metadata (status, resource config, timestamps) and user↔project access mappings in the host database. It never stores credentials, task state, execution logs, or user data — all of that lives inside the container.

**Location in repo**: `internal/sandbox/orchestrator/`

**Socket path**: `/var/run/synchestra-{project_id}.sock`

## Container Lifecycle State Machine

The orchestrator manages each container through a formal state machine. Every container is in exactly one state at any point in time. Transitions are triggered by external events (requests, timeouts) or internal signals (health checks, Docker events).

### States

| State | Description | Container Running? | gRPC Available? |
|---|---|---|---|
| `unprovisioned` | No container exists for this project | No | No |
| `creating` | Container image being pulled, container being created | No | No |
| `starting` | Container started, agent initializing (cloning state repo, setting up workspace) | Yes | No (not yet ready) |
| `running` | Container active, agent ready, accepting commands | Yes | Yes |
| `paused` | Container suspended by orchestrator (idle timeout) | Suspended | No |
| `resuming` | Container being unpaused, agent re-initializing | Yes | No (warming up) |
| `stopping` | Container shutting down gracefully (SIGTERM sent) | Yes (draining) | No (draining) |
| `stopped` | Container exists but is not running | No | No |
| `failed` | Container crashed or health checks failed repeatedly | No/Crashed | No |
| `terminated` | Container and resources explicitly destroyed | No (removed) | No |

### Transitions

All valid state transitions. Any transition not listed here is illegal and must be rejected by the orchestrator.

| From | To | Trigger | Action |
|---|---|---|---|
| `unprovisioned` | `creating` | First request to project / explicit provision | Pull image, create container with resource limits |
| `creating` | `starting` | Container created successfully | Start container, wait for agent readiness |
| `creating` | `failed` | Image pull failure, creation error | Log error, update DB |
| `starting` | `running` | Agent responds to `Ping()` with status `ok` | Update DB, add to connection pool |
| `starting` | `failed` | Startup timeout (60s), `Ping()` failures, entrypoint crash | Log, attempt restart (up to 3 times) |
| `running` | `paused` | Idle timeout exceeded (no active sessions for `IDLE_TIMEOUT`) | Docker pause, close gRPC connection |
| `running` | `stopping` | Explicit stop request, system shutdown | Send SIGTERM, drain active sessions |
| `running` | `failed` | Health check failures exceed threshold (3 consecutive) | Close connection, attempt restart |
| `paused` | `resuming` | New request arrives for this project | Docker unpause |
| `resuming` | `running` | Agent responds to `Ping()` with status `ok` | Re-establish gRPC connection |
| `resuming` | `failed` | Resume timeout (30s), agent unresponsive | Log, attempt full restart |
| `stopping` | `stopped` | All sessions drained, container exited | Update DB, clean up socket |
| `stopped` | `starting` | New request arrives / explicit start | Start existing container |
| `stopped` | `terminated` | Explicit destroy request | Remove container, archive workspace, clean DB |
| `failed` | `starting` | Auto-restart (if `restart_count < max_restarts`) | Recreate and start container |
| `failed` | `terminated` | Max restarts exceeded / explicit destroy | Remove container, clean up |
| `terminated` | `creating` | New request arrives (re-provision) | Create fresh container |

### Transition Diagram

```
                          ┌─────────────┐
                          │unprovisioned│
                          └──────┬──────┘
                     first request│
                          ┌──────▼──────┐
                     ┌────│  creating   │────┐
                     │    └──────┬──────┘    │
                  success        │        failure
                     │    ┌──────▼──────┐    │
                     │    │  starting   │────┤
                     │    └──────┬──────┘    │
                     │     Ping()│ok         │
                     │    ┌──────▼──────┐    │
              ┌──────┼────│   running   │────┤
              │      │    └───┬─────┬───┘    │
         idle │      │   stop │     │ health │
        timeout      │        │     │  fail  │
              │      │  ┌─────▼──┐  │   ┌────▼────┐
              │      │  │stopping│  │   │ failed  │
              │      │  └────┬───┘  │   └────┬────┘
              │      │ drained│     │  restart│ or destroy
         ┌────▼───┐  │  ┌────▼──┐  │        │
         │ paused │  │  │stopped│──┼────────►│
         └────┬───┘  │  └───┬───┘  │   ┌────▼─────┐
        resume│      │destroy│     │   │terminated│
         ┌────▼────┐ │  ┌───▼─────┐│   └────┬─────┘
         │resuming │ │  │terminated││  re-provision
         └────┬────┘ │  └─────────┘│        │
          Ping│ok    │             │        │
              └──────┘             └────────┘
```

## gRPC Connection Pool

### Connection Management

- One gRPC client connection per running container.
- Connections stored in a thread-safe map: `map[string]*grpc.ClientConn` keyed by `{project_id}`.
- Connection created when container transitions to `running`.
- Connection closed when container leaves `running` state (pause, stop, fail).
- Connection dial target: `unix:///var/run/synchestra-{project_id}.sock`.

### Connection Options

```go
// Dial options for container connections
grpc.WithTransportCredentials(insecure.NewCredentials()) // Unix socket — no TLS needed
grpc.WithDefaultCallOptions(
    grpc.MaxCallRecvMsgSize(10 * 1024 * 1024), // 10MB max message
    grpc.MaxCallSendMsgSize(10 * 1024 * 1024),
)
grpc.WithKeepaliveParams(keepalive.ClientParameters{
    Time:                10 * time.Second,  // Ping every 10s if idle
    Timeout:             5 * time.Second,   // Wait 5s for ping ack
    PermitWithoutStream: true,              // Ping even with no active RPCs
})
```

### Connection Lifecycle

1. Container reaches `running` → dial Unix socket → store connection in pool.
2. Request arrives → lookup connection by `{project_id}` → forward RPC or return error.
3. Connection error detected → mark container as potentially failed → trigger health check.
4. Container leaves `running` → close connection → remove from pool.

### Concurrency

- All connection pool operations protected by `sync.RWMutex`.
- Read lock for lookups (hot path).
- Write lock for add/remove (cold path).
- gRPC client connections are themselves thread-safe for concurrent RPCs.

## Health Monitoring

### Health Check Loop

- A single background goroutine iterates all running containers on each tick.
- Interval: 30 seconds (configurable via `SYNCHESTRA_SANDBOX_HEALTH_INTERVAL`).
- Timeout per check: 5 seconds (configurable via `SYNCHESTRA_SANDBOX_HEALTH_TIMEOUT`).
- Uses the `Ping()` RPC defined in [agent.proto](agent.proto).

### Health Check Logic

```
every HEALTH_CHECK_INTERVAL:
    if container.status != "running":
        skip (only check running containers)

    response, err = grpcClient.Ping(ctx, &Empty{}, timeout=5s)

    if err != nil || response.status == "unhealthy":
        container.health_check_failures++
        update DB: last_health_check = now, health_check_failures++

        if health_check_failures >= MAX_FAILURES (default 3):
            transition container → failed
            trigger restart if restart_count < MAX_RESTARTS (default 3)
    else:
        container.health_check_failures = 0
        update DB: last_health_check = now, health_check_failures = 0

        if response.status == "degraded":
            log warning (don't fail, but alert)
```

### Failure Recovery

- **Transient failure** (1–2 missed pings): Log warning, increment counter, continue monitoring.
- **Persistent failure** (3+ consecutive): Transition to `failed`, attempt restart.
- **Restart strategy**: Stop container → remove → create new → start (full cycle).
- **Max restarts**: 3 per hour (configurable via `SYNCHESTRA_SANDBOX_MAX_RESTARTS`). After max, remain in `failed` until manual intervention.
- **Restart backoff**: 5s, 15s, 45s (exponential with 3× multiplier).
- **Restart persistence**: Restart counts (`restart_count`, `last_restart_at`) are persisted in the host database (`sandbox_container_metadata`) to survive host process restarts. This prevents crash-looping containers from getting infinite restart attempts across host restarts.

## Idle Detection & Auto-Pause

### Idle Tracking

- Track `idle_since` timestamp in container metadata (see [database-schema.md](database-schema.md), `sandbox_container_metadata` table).
- Updated on every session completion: set to `now()` when last active session ends.
- Cleared (set to `NULL`) when a new session starts.

### Auto-Pause Logic

```
every IDLE_CHECK_INTERVAL (default 60s):
    for each container where status == "running":
        if active_sessions > 0:
            continue  // Container is busy

        if idle_since is NULL:
            idle_since = now()  // Just became idle
            continue

        idle_duration = now() - idle_since
        if idle_duration >= IDLE_TIMEOUT (default 10 minutes):
            transition container → paused
            log "Container {project_id} auto-paused after {idle_duration}"
```

### Auto-Resume

When a request arrives for a paused container:

1. Transition: `paused` → `resuming`.
2. Docker unpause container.
3. Wait for `Ping()` to succeed (timeout: 30s).
4. Transition: `resuming` → `running`.
5. Re-establish gRPC connection.
6. Route the original request.

**Resume latency target**: < 3 seconds for a paused container (Docker unpause is near-instant, gRPC reconnect is fast).

**Request queuing during resume**: Incoming requests for a resuming container are queued (buffered channel, max 100) and drained once the container reaches `running`. If resume fails, all queued requests receive `UNAVAILABLE` error.

## Resource Quota Enforcement

### Per-Container Limits (Docker Flags)

| Resource | Default | Docker Flag | Configurable? |
|---|---|---|---|
| Memory | 512 MB | `--memory=512m` | Yes, per project |
| CPU | 2.0 cores | `--cpus=2.0` | Yes, per project |
| PIDs | 256 | `--pids-limit=256` | Yes, per project |
| Disk | 50 GB | Volume quota / monitoring | Yes, per project |
| Tmpfs | 64 MB | `--tmpfs /tmp:size=64m` | No |

### Disk Quota Monitoring

Docker does not provide native disk quotas for bind mounts. The orchestrator monitors workspace size asynchronously:

- Check command: `du -sb /workspace/{project_id}/`
- Check interval: every 5 minutes (configurable via `SYNCHESTRA_SANDBOX_DISK_CHECK_INTERVAL`).
- When usage exceeds **90%** of quota: log warning, emit metric.
- When usage exceeds **100%** of quota: block new commands (return `RESOURCE_EXHAUSTED` gRPC error), notify admin.
- Workspace size check is async and does not block request processing.

### Resource Configuration

Resources are configured per-project in the `sandbox_container_metadata` table (see [database-schema.md](database-schema.md)). Defaults can be overridden by admin API (future).

## Circuit Breaker

Each project has its own independent circuit breaker. Failures in one container do not affect request routing to other containers.

### States

| State | Description |
|---|---|
| **Closed** (normal) | Requests flow through to container |
| **Open** (tripped) | All requests immediately return `UNAVAILABLE` — container is known-bad |
| **Half-Open** (testing) | One probe request allowed through to test recovery |

### Transitions

```
Closed → Open:
    When health_check_failures >= MAX_FAILURES
    OR when container transitions to failed/stopped/paused

Open → Half-Open:
    After CIRCUIT_RESET_TIMEOUT (default 30s)

Half-Open → Closed:
    If probe request (Ping) succeeds

Half-Open → Open:
    If probe request fails
```

### Behavior

- **Open circuit**: Return `UNAVAILABLE` gRPC error immediately (no attempt to reach container).
- **Half-open**: Allow one `Ping()` through. If it succeeds, close circuit and process requests. If it fails, re-open.
- **Per-project**: Each project has its own circuit breaker instance. Failures in one container have zero impact on others.

## Docker API Integration

### Container Creation

```go
containerConfig := &container.Config{
    Image: "synchestra/sandbox-agent:" + imageTag,
    Env: []string{
        "SYNCHESTRA_PROJECT_ID=" + projectID,
        "SYNCHESTRA_STATE_REPO_URL=" + stateRepoURL,
        "SYNCHESTRA_LOG_LEVEL=" + logLevel,
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
        socketDir + ":/var/run:rw",
    },
    Resources: container.Resources{
        Memory:    memoryLimitBytes,
        NanoCPUs:  int64(cpuLimit * 1e9),
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
```

### Container Naming Convention

Container name: `synchestra-sandbox-{project_id}`

### Socket Mount Strategy

1. Host creates directory: `/var/run/synchestra/`
2. Container binds: `/var/run/synchestra/:/var/run/:rw`
3. Agent inside container creates socket at: `/var/run/synchestra-{project_id}.sock`
4. Host accesses socket at: `/var/run/synchestra/synchestra-{project_id}.sock`

## Request Routing

### Request Flow

```
HTTP Request → Auth middleware → Orchestrator.Route(project_id, request)
    │
    ├─ Container running? → Get gRPC connection → Forward request → Stream response
    │
    ├─ Container paused? → Auto-resume → Queue request → Forward after resume
    │
    ├─ Container stopped/failed? → Auto-start → Queue request → Forward after ready
    │
    ├─ Container unprovisioned? → Auto-provision → Queue request → Forward after ready
    │
    └─ Container terminated? → Re-provision → Queue request → Forward after ready
```

### Auto-Provision on First Request

When a request arrives for a project with no container:

1. Check user has access to project (DB lookup against `sandbox_user_project_access`).
2. Look up project's state repo URL (from project registry or config).
3. Create container with default resource limits.
4. Start container, wait for readiness.
5. Route original request.

**Provision timeout**: 120 seconds (image pull + container start + agent init). Configurable via `SYNCHESTRA_SANDBOX_PROVISION_TIMEOUT`.

### Request Queue

- Per-project buffered channel (capacity: 100 requests, configurable via `SYNCHESTRA_SANDBOX_REQUEST_QUEUE_SIZE`).
- Requests queued when container is in a transitional state (`creating`, `starting`, `resuming`).
- Drained FIFO when container reaches `running`.
- Timeout per queued request: 120 seconds.
- On timeout or failure: return appropriate gRPC error (`UNAVAILABLE` or `DEADLINE_EXCEEDED`) to all queued requests.

### Session Reconnection

Sessions are persistent and reconnectable. They survive client disconnects, host restarts, and even host+container migration to a different machine.

**Design principle**: The container is the sole source of truth for session state. The host stores no session data — it is purely a router. This means reconnection works as long as the host can reach the container, regardless of whether the host process or machine has changed.

**Reconnection scenarios:**

| Scenario | Session Survives? | Mechanism |
|----------|-------------------|-----------|
| Client disconnects (tab closed, network drop) | ✅ Yes — command continues in container | Client re-attaches via `StreamLogs(session_id)` |
| Client reconnects from different device | ✅ Yes — same user, same session_id | Route to same container via project_id lookup |
| Host process restarts | ✅ Yes — container still running | Lazy reconciliation re-discovers container |
| Host migrates to different machine | ✅ Yes — if container is reachable | New host connects to container via socket/network |
| Container restarts (workspace volume preserved) | ⚠️ Partial — completed session logs survive, running commands lost | Session logs at `/workspace/{project_id}/sessions/` persist on volume |
| Container terminated + workspace destroyed | ❌ No — all session data lost | User must start a new session |

**Key behaviors:**

- Sessions are NOT tied to a single WebSocket/gRPC stream connection.
- The host never caches session state — every reconnection queries the container via `ListSessions` or `StreamLogs`.
- Idle detection is based on container-side active commands, not client connections. A disconnected client with a running command keeps the container marked as "active."
- Session logs are written to the workspace volume (`/workspace/{project_id}/sessions/{session_id}/logs/`), so they survive container restarts as long as the volume is preserved.

### Future Enhancement: Warm Pool

Pre-created containers with no project assignment, ready to be claimed on first request for faster first-request latency. Reduces provisioning time from ~10s to <1s by skipping container creation and startup.

> **Not implemented in initial version.** Synchronous provisioning with pre-pulled images is acceptable for MVP.

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SYNCHESTRA_SANDBOX_IMAGE` | `synchestra/sandbox-agent:latest` | Container image |
| `SYNCHESTRA_SANDBOX_IDLE_TIMEOUT` | `10m` | Time before auto-pause |
| `SYNCHESTRA_SANDBOX_HEALTH_INTERVAL` | `30s` | Health check interval |
| `SYNCHESTRA_SANDBOX_HEALTH_TIMEOUT` | `5s` | Health check timeout |
| `SYNCHESTRA_SANDBOX_MAX_HEALTH_FAILURES` | `3` | Consecutive failures before marking failed |
| `SYNCHESTRA_SANDBOX_MAX_RESTARTS` | `3` | Max restart attempts per hour |
| `SYNCHESTRA_SANDBOX_PROVISION_TIMEOUT` | `120s` | Timeout for container provisioning |
| `SYNCHESTRA_SANDBOX_RESUME_TIMEOUT` | `30s` | Timeout for container resume |
| `SYNCHESTRA_SANDBOX_REQUEST_QUEUE_SIZE` | `100` | Max queued requests per container |
| `SYNCHESTRA_SANDBOX_WORKSPACE_ROOT` | `/var/lib/synchestra/workspaces` | Host workspace directory |
| `SYNCHESTRA_SANDBOX_SOCKET_DIR` | `/var/run/synchestra` | Host socket directory |
| `SYNCHESTRA_SANDBOX_DEFAULT_MEMORY` | `512m` | Default container memory limit |
| `SYNCHESTRA_SANDBOX_DEFAULT_CPU` | `2.0` | Default container CPU limit |
| `SYNCHESTRA_SANDBOX_DEFAULT_DISK` | `50g` | Default workspace disk quota |
| `SYNCHESTRA_SANDBOX_DISK_CHECK_INTERVAL` | `5m` | Disk usage check interval |

## Go Package Structure

```
internal/sandbox/orchestrator/
├── orchestrator.go          // Orchestrator struct, New(), Shutdown()
├── lifecycle.go             // State machine, transitions, Docker API calls
├── connection_pool.go       // gRPC connection pool (dial, close, get)
├── health.go                // Health check loop, failure detection
├── idle.go                  // Idle detection, auto-pause, auto-resume
├── circuit_breaker.go       // Per-project circuit breaker
├── router.go                // Request routing, auto-provision, queuing
├── config.go                // Configuration loading from env vars
├── metrics.go               // Prometheus metrics (future)
└── orchestrator_test.go     // Unit tests (mock Docker client, mock gRPC)
```

## Metrics (Observability)

All metrics use the `synchestra_sandbox_` prefix. Labels follow Prometheus naming conventions.

### Counters

| Metric | Description |
|---|---|
| `synchestra_sandbox_containers_created_total` | Containers created |
| `synchestra_sandbox_containers_failed_total` | Containers that entered `failed` state |
| `synchestra_sandbox_health_checks_total{result="ok\|fail"}` | Health check results |
| `synchestra_sandbox_requests_routed_total{status="success\|queued\|failed"}` | Request routing outcomes |
| `synchestra_sandbox_auto_pauses_total` | Auto-pause events |
| `synchestra_sandbox_auto_resumes_total` | Auto-resume events |

### Gauges

| Metric | Description |
|---|---|
| `synchestra_sandbox_containers_active{status="running\|paused\|stopped\|failed"}` | Current container counts by status |
| `synchestra_sandbox_active_sessions` | Total active sessions across all containers |
| `synchestra_sandbox_connection_pool_size` | Current gRPC connections in pool |
| `synchestra_sandbox_request_queue_depth{project_id}` | Queued requests per project |

### Histograms

| Metric | Description |
|---|---|
| `synchestra_sandbox_provision_duration_seconds` | Time to provision a new container |
| `synchestra_sandbox_resume_duration_seconds` | Time to resume a paused container |
| `synchestra_sandbox_request_latency_seconds` | End-to-end request latency (excluding command execution) |
| `synchestra_sandbox_health_check_duration_seconds` | Health check RPC latency |

## Outstanding Questions

1. Should the orchestrator support multiple container images per project (e.g., different language runtimes)?
2. Should workspace data be preserved across container termination, or archived to object storage?
3. Should there be admin-only endpoints for force-stopping containers or adjusting resource limits at runtime?
4. Should the orchestrator emit events to a message bus (e.g., NATS, Redis Pub/Sub) for external monitoring integration?
5. What is the maximum number of concurrent containers the host should support? Should there be a global limit?
