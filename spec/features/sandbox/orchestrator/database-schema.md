# Sandbox Database Schema

## Overview

> **Related documents:** [orchestrator.md](README.md) (container metadata usage), [lifecycle.md](lifecycle.md) (state transitions that update the database), [outstanding-questions.md](../outstanding-questions.md) (open design questions).

The host-side database (`synchestra serve --http`) stores **minimal data**: only user↔project access mappings and container metadata for lifecycle management. **All state, secrets, and execution data remain inside containers.**

This minimalist approach ensures:
- Host compromise cannot leak secrets (they're not stored there)
- No complex schema migrations or state synchronization logic
- Database is read-heavy (lookups for authorization), write-light (container metadata updates)

## Database Engine

**SQLite 3.35+** — embedded, zero-config, single-file database at `SYNCHESTRA_DATABASE_PATH` (default: `~/.synchestra/sandbox.db`).

SQLite is the right choice because:
- Both tables are small: O(projects) rows for metadata, O(users × projects) for access cache
- Single-process access: only one `synchestra serve --http` process per host
- Read-heavy, write-light workload — no contention concerns
- No external dependency to install, configure, or maintain
- The orchestrator opens the database at startup via Go's `database/sql` with `mattn/go-sqlite3` (or `modernc.org/sqlite` for pure-Go builds)

The DDL below uses SQLite-compatible syntax.

## Tables

### `sandbox_user_project_access` (cache)

**Local cache** of user↔project access mappings. The source of truth is the external/cloud database; this table is a read-through cache refreshed on access or periodically.

```sql
CREATE TABLE sandbox_user_project_access (
    user_id    TEXT NOT NULL,
    project_id TEXT NOT NULL,
    access_level TEXT NOT NULL DEFAULT 'read_write',
    -- access_level: 'read', 'read_write', 'admin'
    cached_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    -- When this row was fetched from the external DB (for TTL-based eviction)
    
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX idx_user_projects ON sandbox_user_project_access(user_id);
CREATE INDEX idx_project_users ON sandbox_user_project_access(project_id);
CREATE INDEX idx_cache_age ON sandbox_user_project_access(cached_at);
```

**Columns:**
- `user_id`: External user identifier (from auth system, e.g., GitHub username, Auth0 ID)
- `project_id`: Synchestra project identifier (matches state repo name)
- `access_level`: Authorization level for this user on this project
- `cached_at`: When this row was fetched from the external DB — rows older than the cache TTL (default: 5 min) are re-validated on next access

**Cache behavior:**
- On API request: check local cache → if miss or stale (`cached_at` older than TTL), fetch from external DB, upsert into cache
- Periodic cleanup: `DELETE FROM sandbox_user_project_access WHERE cached_at < datetime('now', '-1 hour')` removes entries not accessed recently
- On external DB unavailability: serve from stale cache (log warning)

**Queries:**
```sql
-- Check if user can access project (cache hit)
SELECT access_level, cached_at FROM sandbox_user_project_access
WHERE user_id = ? AND project_id = ?;

-- Evict stale cache entries
DELETE FROM sandbox_user_project_access
WHERE cached_at < datetime('now', '-1 hour');
```

### `sandbox_container_metadata`

Local source of truth for container lifecycle on this host. One row per project. Used by the orchestrator for status queries (auto-pause, health checks, routing).

```sql
CREATE TABLE sandbox_container_metadata (
    project_id TEXT PRIMARY KEY,
    
    -- Container identity
    container_id TEXT UNIQUE,
    -- Docker container ID (NULL if container not created yet)
    
    container_image TEXT NOT NULL DEFAULT 'synchestra/sandbox-agent:latest',
    -- Per-project image override
    
    container_status TEXT NOT NULL DEFAULT 'stopped',
    -- Status values:
    -- 'creating'    - Container image being pulled/built
    -- 'running'     - Container is active and accepting commands
    -- 'paused'      - Container is suspended (idle timeout)
    -- 'stopped'     - Container is shut down but preserved
    -- 'failed'      - Container crashed or health checks failed
    -- 'terminated'  - Container was explicitly destroyed/removed
    
    socket_path TEXT,
    -- /var/run/synchestra-{project_id}.sock (NULL until container starts)
    
    -- Resource configuration
    resource_quota_gb INTEGER NOT NULL DEFAULT 50,
    memory_limit_mb  INTEGER NOT NULL DEFAULT 512,
    cpu_limit         REAL NOT NULL DEFAULT 2.0,
    
    -- Lifecycle timestamps (ISO 8601 strings)
    created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    started_at  TEXT,
    paused_at   TEXT,
    idle_since  TEXT,
    
    -- Restart tracking
    restart_count   INTEGER DEFAULT 0,
    last_restart_at TEXT,
    
    updated_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    
    -- Health
    last_health_check    TEXT,
    health_check_failures INTEGER DEFAULT 0
);

CREATE INDEX idx_container_status ON sandbox_container_metadata(container_status);
CREATE INDEX idx_container_idle ON sandbox_container_metadata(container_status, idle_since);
```

**Columns:**
- `project_id`: Synchestra project identifier (primary key)
- `container_id`: Docker container ID (NULL until first creation)
- `container_status`: Lifecycle state of the container
- `socket_path`: Unix socket path for gRPC communication
- `resource_quota_gb`: Disk quota (enforced by Docker volume or kernel quotas)
- `memory_limit_mb`: Docker memory constraint
- `cpu_limit`: Docker CPU constraint (cores, fractional)
- `created_at`: Container creation timestamp (ISO 8601)
- `started_at`: When container last started
- `paused_at`: When container was last paused
- `idle_since`: When the container became idle (used for auto-pause)
- `restart_count`: Consecutive restart attempts (persisted to survive host process restarts)
- `last_restart_at`: When last restart was attempted
- `updated_at`: Last database row update
- `last_health_check`: Last successful `Ping()` RPC to container
- `health_check_failures`: Consecutive failures — feeds circuit breaker pattern

**Queries:**
```sql
-- Get container info for project (for routing requests)
SELECT container_id, socket_path, container_status
FROM sandbox_container_metadata
WHERE project_id = ?;

-- Find idle containers (for auto-pause)
SELECT project_id, container_id FROM sandbox_container_metadata
WHERE container_status = 'running'
  AND idle_since < datetime('now', '-10 minutes');

-- Find paused containers (for cleanup after 24h idle)
SELECT project_id, container_id FROM sandbox_container_metadata
WHERE container_status = 'paused'
  AND paused_at < datetime('now', '-24 hours');

-- Resume from pause
UPDATE sandbox_container_metadata
SET container_status = 'running', idle_since = NULL,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE project_id = ? AND container_status = 'paused';

-- Mark container idle
UPDATE sandbox_container_metadata
SET idle_since = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
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
- **Session data**: Ephemeral, stored in `/workspace/{project_id}/sessions/{session_id}/`
- **Execution logs**: Streamed to user, optionally retained in container
- **User data**: Cloned repos, working directories, never persisted in host DB

### Why This Design

1. **Host compromise doesn't leak secrets**: Attacker cannot read credentials from host DB
2. **No sync conflicts**: Container is source of truth; no need to merge host state into container
3. **Scalability**: Database remains small (O(projects × users), not O(executions))
4. **Simplicity**: Host is stateless router; container is autonomous

## Indexes

Indexes are defined inline in the DDL above. Summary:

| Index | Table | Purpose |
|---|---|---|
| `idx_user_projects` | `sandbox_user_project_access` | Fast lookup by user |
| `idx_project_users` | `sandbox_user_project_access` | Fast lookup by project |
| `idx_cache_age` | `sandbox_user_project_access` | TTL-based cache eviction |
| `idx_container_status` | `sandbox_container_metadata` | Filter by lifecycle state |
| `idx_container_idle` | `sandbox_container_metadata` | Auto-pause candidate queries |

## Retention & Cleanup

### Access Cache
- Entries evicted automatically when older than cache TTL (default: 5 min on read, 1 hour hard cleanup)
- Full cache cleared on orchestrator restart (rebuilt on demand)

### Container Metadata
- Retained while container exists
- On container termination:
  - Soft-delete via `container_status = 'terminated'`
  - Hard-delete after 30 days (periodic cleanup job)

## Migration Strategy

### Initial Deployment

The orchestrator creates the database and tables at startup if they don't exist (embedded DDL in Go). No external migration tool required.

```go
// On startup: os.MkdirAll(filepath.Dir(dbPath), 0700)
// Then: sql.Open("sqlite3", dbPath)
// Then: db.Exec(ddl) for each CREATE TABLE IF NOT EXISTS
```

### Scaling (Future)

SQLite is sufficient for single-host deployments (hundreds of projects). If Synchestra grows to support multi-host orchestration, the access cache and container metadata would need to move to a shared store (PostgreSQL, CockroachDB). The `*sql.DB` interface makes this a driver swap, not a rewrite.

## Outstanding Questions

1. ~~Should we retain soft-deleted container metadata for audit/recovery purposes?~~ **Resolved**: Yes.
2. What is the retention policy for access mapping history (audit trail of who had access when)?
3. Should we add a separate audit/event log table (separate from these operational tables)?
4. For multi-tenant deployments, do we need tenant_id isolation at the database level?
5. Should resource_quota fields be enforced at database layer or application layer?
