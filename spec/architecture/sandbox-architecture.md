# Sandbox Architecture

## Overview

The Sandbox feature provides isolated execution environments for running user-initiated commands from the chat interface. Each Synchestra **project** (defined by its state repository) gets its own persistent Docker container with isolated disk space. Multiple users can share the same container for a project, with commands executing in parallel in isolated session directories.

**Key Principle**: The host (`synchestra serve --http`) is **stateless and only routes requests**. All state, task coordination, and secrets are managed **inside containers**. Containers are autonomous and self-sufficient.

## System Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│  Host: HTTP Server + Container Orchestrator             │
│  - Stateless request router                             │
│  - Lifecycle manager for containers                     │
│  - DB: user↔project access mappings only                │
│  - NO state storage, NO secrets                         │
└────────────┬──────────────────────────────────────────┘
             │
             │ gRPC (Unix socket)
             │ user_id + project_id + command
             │
             ▼
┌─────────────────────────────────────────────────────────┐
│  Container (1 per project)                              │
│  - gRPC Agent (listens on Unix socket)                  │
│  - .synchestra/ (git-backed state repo)                 │
│  - Encrypted credential store (AES256-GCM)                │
│  - Session workspaces (per user request)                │
└─────────────────────────────────────────────────────────┘
```

### Components

#### Host-Side

**HTTP Server** (`synchestra serve --http`)
- REST API for web app (`/api/v1/sandbox/*`)
- Routes sandbox requests → gRPC client calls to containers
- Validates user↔project access (authorization gate)
- **Database**: Only `sandbox_user_project_access` and `sandbox_container_metadata` (NO state or secrets)

**Container Orchestrator** (background service)
- Creates containers on first request to a project
- Manages lifecycle: start, pause, resume, destroy
- Maintains gRPC connection pool to containers
- Implements hybrid lifecycle: auto-pause during idle, auto-resume on demand
- Health checks via `Ping()` RPC
- **No state persistence beyond container metadata**

#### Container-Side (Inside Each Container)

**gRPC Agent** (background service)
- Listens on `/var/run/synchestra-{project_id}.sock`
- Implements `SandboxAgent` service (see gRPC Protocol section)
- Routes requests to isolated session working directories
- Manages execution, streaming output, timeouts
- Source of truth for container state (uptime, active sessions)

**State Repository** (`.synchestra/` directory)
- Git clone of the project's state repository (same repo that triggers the container)
- Container pulls/updates independently (no host sync)
- `synchestra` CLI inside container reads from `.synchestra/`
- Enables container to resolve task context during command execution

**Session Manager**
- Creates isolated working directory: `/workspace/{project_id}/sessions/{session_id}/`
- Isolates multiple concurrent users/requests
- Enforces resource limits per session (memory, CPU)
- Cleans up after session completes or times out

**Credential Store** (`.secure/credentials.enc`)
- Encrypted vault for user-provided secrets (git tokens, SSH keys, API keys)
- AES256-GCM encryption with per-container key
- User sends credential → gRPC `StoreCredential()` → container encrypts + stores
- Command execution: container decrypts on-demand, passes to subprocess
- Host has **zero access** to decryption key or unencrypted values

## Data Model

### Host-Side Database (Minimal)

```sql
-- User access to projects (for authorization)
CREATE TABLE sandbox_user_project_access (
    user_id VARCHAR(255),
    project_id VARCHAR(255),
    access_level VARCHAR(50),  -- read, read_write, admin
    PRIMARY KEY (user_id, project_id)
);

-- Container metadata (for lifecycle management)
CREATE TABLE sandbox_container_metadata (
    project_id VARCHAR(255) PRIMARY KEY,
    container_id VARCHAR(255),           -- Docker ID
    container_status VARCHAR(50),        -- running, paused, stopped, failed, terminated
    -- NOTE: The orchestrator defines additional transitional states (unprovisioned, creating,
    -- starting, resuming, stopping) that are tracked in-memory but NOT persisted to the database.
    -- The database stores only 6 stable states: creating, running, paused, stopped, failed, terminated.
    socket_path VARCHAR(255),            -- /var/run/synchestra-{project_id}.sock
    resource_quota_gb INT,               -- max disk
    memory_limit_mb INT,                 -- max memory
    cpu_limit FLOAT,                     -- max CPUs
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    idle_since TIMESTAMP                 -- for scale-down
);
```

**NOT stored in host database:**
- Task state (lives in `.synchestra/` inside container)
- Session data (lives inside container)
- Credentials (encrypted in container only)
- Execution logs (streamed to user, optionally stored in container)
- User data from repos (never leaves container)

### Container-Side File Structure

```
/workspace/{project_id}/
├── .synchestra/                      # Git state repo
│   ├── .git/
│   ├── tasks/                        # Task definitions
│   ├── state.json                    # Current state snapshot
│   └── README.md
│
├── repos/                            # Cloned user repositories
│   ├── repo1/
│   └── repo2/
│
├── sessions/                         # Per-session working dirs
│   ├── {session_id_1}/
│   │   ├── working/                  # Command execution dir
│   │   └── logs/                     # Session output
│   └── {session_id_2}/
│       ├── working/
│       └── logs/
│
└── .secure/
    └── credentials.enc               # Encrypted credential store
```

## Communication Protocol

### gRPC Service (SandboxAgent)

Container listens on Unix socket `/var/run/synchestra-{project_id}.sock` and implements:

```protobuf
service SandboxAgent {
  rpc ExecuteCommand(CommandRequest) returns (stream CommandOutput) {}
  rpc GetStatus(StatusRequest) returns (StatusResponse) {}
  rpc ListSessions(ListSessionsRequest) returns (SessionList) {}
  rpc StreamLogs(StreamLogsRequest) returns (stream LogEntry) {}
  rpc StoreCredential(StoreCredentialRequest) returns (StoreCredentialResponse) {}
  rpc GetTaskState(GetTaskStateRequest) returns (TaskStateResponse) {}
  rpc Ping(google.protobuf.Empty) returns (PingResponse) {}
}
```

**ExecuteCommand Flow:**
1. User sends command via HTTP endpoint
2. Host routes to container's gRPC agent
3. Container creates isolated session directory
4. Container executes command, streams stdout/stderr
5. On completion, container returns exit code
6. Session artifacts (logs) retained in `/workspace/{project_id}/sessions/{session_id}/logs/` until cleanup

**GetTaskState Flow:**
1. Host (or web app) requests task state
2. Container reads from `.synchestra/state.json` and returns
3. No host-side state cache; container is source of truth

**StoreCredential Flow:**
1. User provides token via web UI
2. Host sends via gRPC (TLS-encrypted) to container
3. Container receives, encrypts with local key: `AES256-GCM(token, container_key)`
4. Container stores in `.secure/credentials.enc`
5. Host receives success response (no token echoed back)

See `spec/features/sandbox/protocol.md` for full message definitions.

## Concurrency & Isolation

### Parallel Command Execution

Multiple users can execute commands concurrently in the same container:
- Each request gets unique `session_id`
- Working directory: `/workspace/{project_id}/sessions/{session_id}/working/`
- Separate stdout/stderr logs per session
- Resource limits (cgroups) enforced per session, not per user
- gRPC multiplexing allows 100s of concurrent command streams

### User Isolation

Users executing commands in the same container are isolated by:
1. **Filesystem**: Separate working directories, no cross-session file access
2. **Process**: Commands run in separate processes/subshells
3. **Credentials**: User A's stored credentials not visible to User B
4. **Logs**: Separate log streams per session
5. **gRPC auth**: Each request includes `user_id` (validated at API gateway, not replayed)

**Note**: Multiple users in same container means **they cannot access each other's files or credentials** but **can see each other's existence** (sessions list, shared project structure). This is intentional for collaboration awareness.

### Resource Quotas

Per-session resource limits (enforced via cgroups inside container):
- **Memory**: ~100MB per session (configurable)
- **CPU**: 0.5 cores per session (configurable)
- **Disk**: Shared quota for entire project workspace (e.g., 50GB)

If a session exceeds limits, container kills subprocess and returns error.

## Container Lifecycle

### Startup (First Request to Project)

1. Orchestrator receives request for project X
2. Check if container exists; if not:
   - Create Docker container from sandbox image
   - Bind-mount `/var/lib/synchestra/workspaces/{project_id}/` → `/workspace/{project_id}/`
   - Start gRPC agent inside container
   - Container clones state repo: `git clone {state_repo_url} .synchestra/`
   - Container generates encryption key for credential store
   - Host waits for `Ping()` to succeed
3. Host creates socket connection pool entry
4. Request routed to container

### Running (Active Commands)

- Container actively processes commands
- Update `idle_since = NULL` in host DB
- Connection stays alive for next request

### Idle (No Commands for N Minutes)

1. Orchestrator detects idle: `now() - idle_since > idle_threshold` (default 10 min)
2. Container still running but no active commands
3. Triggers auto-pause: `docker pause {container_id}`
4. Update `container_status = paused` in host DB
5. Save container memory to disk (Docker pause feature)

### Resume (New Command While Paused)

1. New request arrives for paused container
2. Orchestrator: `docker unpause {container_id}`
3. Wait for `Ping()` to succeed (verify ready)
4. Update `container_status = running`
5. Route request to container

### Cleanup (Idle Timeout or Project Deletion)

1. If paused for >M hours (configurable, default 24h):
   - `docker stop {container_id}`
   - Archive workspace: `tar gz /var/lib/synchestra/workspaces/{project_id}/ → backup storage`
   - `docker rm {container_id}`
   - Delete host DB records
2. If project deleted explicitly:
   - Same cleanup immediately

### Failed (Container Crash or Health Check Failure)

1. Container health check (`Ping()` RPC) fails repeatedly
2. After N consecutive failures (configurable, default 3): update `container_status = failed`
3. Orchestrator may attempt automatic restart or escalate to cleanup
4. Failed containers retain workspace for debugging; manual intervention may be needed

### Terminated (Explicit Destruction)

1. Container explicitly destroyed via `DELETE /api/v1/sandbox/{project_id}` or orchestrator cleanup
2. Update `container_status = terminated`
3. Container removed: `docker rm {container_id}`
4. Workspace archived or deleted depending on policy

## State Management

### Container-Authoritative State

- **State repo** (`.synchestra/`) lives inside container only
- Container is source of truth for task state, progress, coordination
- Host has no state repo; cannot read task state directly
- Host can query state via `GetTaskState()` gRPC call if needed

### No State Sync Between Host and Container

- **Removed**: No periodic pull-push sync between host and container
- **Why**: Eliminates conflict resolution, eventual consistency issues, and complexity
- Container manages state independently for its project
- Host purely routes requests and manages lifecycle

### State Updates

- When command executes inside container, container uses `synchestra` CLI to update state
- `synchestra task update --status completed` reads/writes `.synchestra/`
- Changes committed to local git repo
- No push-back to host

## Security Model

See `spec/architecture/sandbox-security.md` for detailed threat analysis and mitigations.

**Quick Summary:**

- **Credential Encryption**: AES256-GCM at rest inside container. Decryption key never leaves container.
- **Host Boundaries**: Host cannot read decrypted credentials, task state, or execution artifacts.
- **Container Hardening**: Drop capabilities, unprivileged UID, cgroups limits, read-only filesystem (where safe).
- **User Isolation**: Separate session directories, credentials, and resource limits per user within same container.
- **Network**: Containers connected to Docker bridge network (isolated from external networks by default).

## HTTP API Surface

Host-side REST endpoints (implemented by Container Orchestrator, routes to container via gRPC):

```
POST   /api/v1/sandbox/{project_id}/execute
       Request: { command: [string], user_id: string, timeout_seconds: int }
       Response: { session_id: string }
       Streams: stdout/stderr (EventSource or WebSocket)
       Default timeout_seconds: 1800 (30 minutes). Range: [1, 86400].

GET    /api/v1/sandbox/{project_id}/status
       Response: { container_status, uptime, active_sessions, resource_usage }

GET    /api/v1/sandbox/{project_id}/sessions
       Query: ?user_id=X (optional)
       Response: [{ session_id, user_id, status, command_count, created_at }]

GET    /api/v1/sandbox/{project_id}/sessions/{session_id}
       Response: { status, exit_code, stdout, stderr, created_at, completed_at }

WebSocket /api/v1/sandbox/{project_id}/sessions/{session_id}/logs
       Stream: real-time log entries (for active sessions)
       Format: { timestamp, stream: "stdout"|"stderr", data: bytes }

POST   /api/v1/sandbox/{project_id}/credentials
       Request: { type: "git_token", value: string, identifier: string }
       Response: { success: bool, message: string }
       Note: sent over TLS; value never echoed back

DELETE /api/v1/sandbox/{project_id}
       Destroys container, archives workspace, deletes host DB records
```

## Deployment & Operations

### Host Infrastructure

- **Docker runtime**: Must be available on host
- **Disk space**: `/var/lib/synchestra/workspaces/{project_id}/` for each container workspace
- **Network**: Docker bridge network (isolated by default)
- **Database**: PostgreSQL or SQLite for `sandbox_user_project_access` and `sandbox_container_metadata`

### Container Image

- Base: Ubuntu/Alpine + Go runtime
- Includes: git, docker CLI (if DinD needed), synchestra CLI
- Entrypoint: Start gRPC agent listening on Unix socket
- Should be versioned and published to container registry

### Scaling Considerations

- **Single host**: All containers run on one machine (typical for < 100 projects)
- **Multi-host**: Different hosts can orchestrate different projects (future: agent discovery, load balancing)
- **Resource limits**: Set per-container quotas to prevent one project starving others
- **Monitoring**: Export container health, resource usage, and command metrics for observability

## Design Rationale

### Why Containers Per Project (Not Per User)

- **Efficiency**: Reuses container and workspace for multiple users
- **Collaboration**: Shared state repository enables multi-user coordination
- **Resource**: Fewer containers to manage (O(projects) not O(users))
- **Isolation**: User isolation within container via sessions is sufficient

### Why Stateless Host

- **Simplicity**: Host is just a router and lifecycle manager
- **Reliability**: No host-side state to corrupt, sync, or recover
- **Security**: Host cannot be compromised to leak secrets (it doesn't have them)
- **Scalability**: Multiple hosts can orchestrate disjoint sets of containers independently

### Why Container-Authoritative State

- **Consistency**: No split-brain issues, no conflict resolution
- **Autonomy**: Container can operate independently if host is down (eventually)
- **Performance**: No network round-trips for every task state read
- **Simplicity**: Container uses standard `synchestra` CLI to manage state

### Why Encrypted Local Store for Credentials

- **Security**: Keys never leave container, not visible to host, not transmitted over network unnecessarily
- **Simplicity**: No external vault dependency (can add later as option)
- **Performance**: Decryption happens locally, no network calls per command
- **Isolation**: Each container has independent credential set; cannot be shared across projects

## Outstanding Questions

1. **External vault integration**: Should we support HashiCorp Vault or AWS Secrets Manager as credential backend? Timeline?
2. **Horizontal scaling**: How should credentials be synchronized if container moves to different host?
3. **Disaster recovery**: Should container workspaces be continuously replicated or just backed up on idle?
4. **Multi-project credential sharing**: Can credentials from project A be used in project B? Access model?
5. **Audit trail**: Should all credential access (decrypt) attempts be logged?
6. **Container image updates**: How should running containers pick up new Synchestra CLI versions?
7. **Docker socket mounting**: Should containers have access to host Docker socket for docker-in-docker workflows?
