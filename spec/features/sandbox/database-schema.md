# Sandbox Database Schema

## Overview

The host-side database (`synchestra serve --http`) stores **minimal data**: only user↔project access mappings and container metadata for lifecycle management. **All state, secrets, and execution data remain inside containers.**

This minimalist approach ensures:
- Host compromise cannot leak secrets (they're not stored there)
- No complex schema migrations or state synchronization logic
- Database is read-heavy (lookups for authorization), write-light (container metadata updates)

## Tables

### `sandbox_user_project_access`

Maps users to projects for access control. Used by HTTP API to validate authorization before routing requests to container.

```sql
CREATE TABLE sandbox_user_project_access (
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    access_level VARCHAR(50) NOT NULL DEFAULT 'read_write',
    -- access_level: 'read', 'write', 'admin', or custom policies
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (user_id, project_id),
    INDEX idx_user_projects (user_id),
    INDEX idx_project_users (project_id)
);
```

**Columns:**
- `user_id`: External user identifier (from auth system, e.g., GitHub username, Auth0 ID)
- `project_id`: Synchestra project identifier (matches state repo name)
- `access_level`: Authorization level for this user on this project
- `created_at`: When access was granted
- `updated_at`: When access was last modified (e.g., downgrade from admin to write)

**Queries:**
```sql
-- Check if user can access project
SELECT access_level FROM sandbox_user_project_access
WHERE user_id = ? AND project_id = ?;

-- List all projects user can access
SELECT project_id, access_level FROM sandbox_user_project_access
WHERE user_id = ?;

-- List all users with access to project
SELECT user_id, access_level FROM sandbox_user_project_access
WHERE project_id = ?;
```

### `sandbox_container_metadata`

Tracks container lifecycle and configuration for each project. Used by Container Orchestrator to manage startup, pause/resume, and cleanup.

```sql
CREATE TABLE sandbox_container_metadata (
    project_id VARCHAR(255) PRIMARY KEY,
    
    -- Container identity
    container_id VARCHAR(255) UNIQUE,
    -- Docker container ID (nullable if container not created yet)
    
    container_status VARCHAR(50) NOT NULL DEFAULT 'stopped',
    -- running, paused, stopped, failed, terminated
    
    socket_path VARCHAR(255),
    -- /var/run/synchestra-{project_id}.sock (nullable until container starts)
    
    -- Resource configuration
    resource_quota_gb INT NOT NULL DEFAULT 50,
    -- Max disk space for /workspace/{project_id}/ (GB)
    
    memory_limit_mb INT NOT NULL DEFAULT 512,
    -- Docker memory limit (MB)
    
    cpu_limit FLOAT NOT NULL DEFAULT 2.0,
    -- Docker CPU limit (cores, can be fractional)
    
    -- Lifecycle metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- When container was first created
    
    started_at TIMESTAMP WITH TIME ZONE,
    -- Last time container transitioned to 'running'
    
    paused_at TIMESTAMP WITH TIME ZONE,
    -- Last time container transitioned to 'paused'
    
    idle_since TIMESTAMP WITH TIME ZONE,
    -- When no commands have been executing (used for auto-pause)
    
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Last metadata update (for optimistic locking or versioning)
    
    -- Observability
    last_health_check TIMESTAMP WITH TIME ZONE,
    -- Last successful Ping() RPC
    
    health_check_failures INT DEFAULT 0,
    -- Consecutive failed health checks (reset on success)
    
    INDEX idx_status (container_status),
    INDEX idx_idle_since (idle_since)
);
```

**Columns:**
- `project_id`: Synchestra project identifier (primary key)
- `container_id`: Docker container ID (nullable until first creation)
- `container_status`: Enum state of container lifecycle
- `socket_path`: Unix socket path for gRPC communication
- `resource_quota_gb`: Disk quota (enforced by Docker volume or kernel quotas)
- `memory_limit_mb`: Docker memory constraint
- `cpu_limit`: Docker CPU constraint
- `created_at`: Container creation timestamp
- `started_at`: When container last started
- `paused_at`: When container was last paused
- `idle_since`: Timestamp used to detect if container should auto-pause
- `updated_at`: Last database row update
- `last_health_check`: Last successful `Ping()` RPC to container
- `health_check_failures`: Counter for circuit breaker pattern

**Queries:**
```sql
-- Get container info for project (for routing requests)
SELECT container_id, socket_path, container_status
FROM sandbox_container_metadata
WHERE project_id = ?;

-- Find idle containers (for auto-pause)
SELECT project_id, container_id FROM sandbox_container_metadata
WHERE container_status = 'running'
  AND idle_since < NOW() - INTERVAL '10 minutes';

-- Find paused containers (for cleanup after 24h idle)
SELECT project_id, container_id FROM sandbox_container_metadata
WHERE container_status = 'paused'
  AND paused_at < NOW() - INTERVAL '24 hours';

-- Update container on-demand (resume from pause)
UPDATE sandbox_container_metadata
SET container_status = 'running', idle_since = NULL, updated_at = NOW()
WHERE project_id = ? AND container_status = 'paused';

-- Mark container idle (for auto-pause decision)
UPDATE sandbox_container_metadata
SET idle_since = NOW(), updated_at = NOW()
WHERE project_id = ? AND container_status = 'running'
  AND idle_since IS NULL;
```

## Data NOT Stored in Host Database

### Intentionally Absent (Security)

- **Credentials**: No tokens, SSH keys, or API keys stored in host DB
- **Decryption keys**: Container manages all key material
- **Unencrypted secrets**: Never appear in host database

### Container-Managed (State Accuracy)

- **Task state**: Live in `.synchestra/` inside container
- **Session data**: Ephemeral, stored in `/workspace/{project}/sessions/{session_id}/`
- **Execution logs**: Streamed to user, optionally retained in container
- **User data**: Cloned repos, working directories, never persisted in host DB

### Why This Design

1. **Host compromise doesn't leak secrets**: Attacker cannot read credentials from host DB
2. **No sync conflicts**: Container is source of truth; no need to merge host state into container
3. **Scalability**: Database remains small (O(projects × users), not O(executions))
4. **Simplicity**: Host is stateless router; container is autonomous

## Indexes

**Recommended indexes for performance:**

```sql
-- Fast access verification
CREATE INDEX idx_user_projects ON sandbox_user_project_access(user_id);
CREATE INDEX idx_project_users ON sandbox_user_project_access(project_id);

-- Lifecycle management (auto-pause, cleanup)
CREATE INDEX idx_container_idle ON sandbox_container_metadata(container_status, idle_since);
CREATE INDEX idx_container_status ON sandbox_container_metadata(container_status);
CREATE INDEX idx_container_paused ON sandbox_container_metadata(paused_at)
WHERE container_status = 'paused';
```

## Retention & Cleanup

### Access Mappings

- Retained indefinitely (unless user deleted from system)
- No auto-cleanup; managed via administrative process

### Container Metadata

- Retained while container exists
- On container cleanup (destroyed after idle timeout):
  - Archive to backup storage (for recovery)
  - Delete from active table (or soft-delete via `deleted_at`)
  - Optional: Retain for 30 days before hard delete

## Migration Strategy

### Initial Deployment

```sql
CREATE TABLE sandbox_user_project_access (...)
CREATE TABLE sandbox_container_metadata (...)
CREATE INDEX idx_user_projects ON sandbox_user_project_access(user_id);
-- (... other indexes)
```

### Scaling (Future)

- **Replication**: Replicate `sandbox_user_project_access` to read replicas for fast authorization
- **Partitioning**: If millions of users, partition by `project_id` (hash partition)
- **Archive**: Move old `paused_at` containers to archive table after 30 days

## Outstanding Questions

1. Should we retain soft-deleted container metadata for audit/recovery purposes?
2. What is the retention policy for access mapping history (audit trail of who had access when)?
3. Should we add a separate audit/event log table (separate from these operational tables)?
4. For multi-tenant deployments, do we need tenant_id isolation at the database level?
5. Should resource_quota fields be enforced at database layer or application layer?
