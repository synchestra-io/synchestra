# Monitoring, Logging, and Alerting

## Overview

Observability strategy for the sandbox feature covering metrics (Prometheus), structured logging (JSON), distributed tracing (OpenTelemetry), alerting rules, and dashboards. Monitoring spans both host-side (orchestrator, HTTP API) and container-side (gRPC agent, command execution).

The host is stateless — all durable state lives in containers and the database. This means monitoring must cover both sides to provide a complete picture: the host reports on routing, lifecycle management, and API health, while the container agent reports on session execution, credential operations, and workspace state.

### Design Principles

- **Prometheus conventions**: all metrics use the `synchestra_sandbox_` prefix (host) or `synchestra_agent_` prefix (container)
- **Structured JSON logs**: machine-parseable, with consistent field names across components
- **No sensitive data in logs or metrics**: credential values, encryption keys, and auth tokens are never emitted
- **Trace context everywhere**: every log entry and span carries `trace_id` for end-to-end correlation
- **Actionable alerts**: every alert rule has a clear severity, routing, and suggested response

### Related Specs

- [orchestrator.md](orchestrator.md) — metrics definitions (source of truth for orchestrator counters/gauges/histograms)
- [lifecycle.md](lifecycle.md) — lifecycle events that drive log entries and alert conditions
- [http-api.md](http-api.md) — API endpoints that expose health checks and generate request metrics
- [credentials.md](credentials.md) — credential operations referenced in logging policy
- [protocol.md](protocol.md) — gRPC protocol between host and container agent

---

## Metrics

### Host-Side Metrics (Orchestrator + HTTP API)

Consolidates and extends the metrics defined in [orchestrator.md](orchestrator.md).

#### Counters

| Metric | Labels | Description |
|--------|--------|-------------|
| `synchestra_sandbox_containers_created_total` | | Containers successfully created |
| `synchestra_sandbox_containers_failed_total` | | Container creation failures |
| `synchestra_sandbox_health_checks_total` | `result="ok\|fail"` | Health check executions |
| `synchestra_sandbox_requests_routed_total` | `status="success\|queued\|failed"` | Requests routed to containers |
| `synchestra_sandbox_auto_pauses_total` | | Idle containers auto-paused |
| `synchestra_sandbox_auto_resumes_total` | | Paused containers auto-resumed |
| `synchestra_sandbox_evictions_total` | | Containers evicted |
| `synchestra_sandbox_commands_executed_total` | `project_id`, `status` | Commands executed via agent |
| `synchestra_sandbox_credentials_stored_total` | `project_id` | Credential store operations |
| `synchestra_sandbox_restarts_total` | `project_id` | Container restart attempts |
| `synchestra_sandbox_http_requests_total` | `method`, `endpoint`, `status_code` | HTTP API requests |
| `synchestra_sandbox_rate_limit_exceeded_total` | `endpoint` | Rate limit hits |

#### Gauges

| Metric | Labels | Description |
|--------|--------|-------------|
| `synchestra_sandbox_containers_active` | `status="running\|paused\|stopped\|failed"` | Containers by state |
| `synchestra_sandbox_active_sessions` | | Current active sessions across all containers |
| `synchestra_sandbox_connection_pool_size` | | gRPC connection pool utilization |
| `synchestra_sandbox_request_queue_depth` | `project_id` | Queued requests per project |
| `synchestra_sandbox_containers_total` | | Total managed containers |
| `synchestra_sandbox_disk_usage_bytes` | `project_id` | Per-project disk usage |
| `synchestra_sandbox_memory_usage_bytes` | `project_id` | Per-container memory usage |

#### Histograms

| Metric | Labels | Description |
|--------|--------|-------------|
| `synchestra_sandbox_provision_duration_seconds` | | Container provisioning time |
| `synchestra_sandbox_resume_duration_seconds` | | Container resume time |
| `synchestra_sandbox_request_latency_seconds` | | End-to-end request latency |
| `synchestra_sandbox_health_check_duration_seconds` | | Health check round-trip time |
| `synchestra_sandbox_command_duration_seconds` | `project_id` | Command execution time |
| `synchestra_sandbox_http_request_duration_seconds` | `method`, `endpoint` | HTTP handler latency |
| `synchestra_sandbox_websocket_connection_duration_seconds` | | WebSocket session length |

### Container-Side Metrics (gRPC Agent)

Metrics exposed by the agent inside each container. Scraped via the orchestrator proxy or pushed via the event bus (see [Outstanding Questions](#outstanding-questions)).

| Metric | Labels | Description |
|--------|--------|-------------|
| `synchestra_agent_sessions_active` | | Current active sessions in this container |
| `synchestra_agent_commands_executed_total` | `status` | Commands run (success/failure/timeout) |
| `synchestra_agent_credential_operations_total` | `operation="store\|get\|delete"` | Credential store/get/delete operations |
| `synchestra_agent_disk_usage_bytes` | | Workspace disk usage |
| `synchestra_agent_uptime_seconds` | | Agent uptime since last start |

### Metric Exposition

| Source | Transport | Endpoint | Scrape Interval |
|--------|-----------|----------|-----------------|
| Host (orchestrator + HTTP API) | Prometheus pull | `/metrics` on HTTP server | 15s |
| Container (gRPC agent) | gRPC `GetStatus` response or dedicated metrics RPC | Proxied through orchestrator | 30s |

---

## Structured Logging

### Log Format

All logs use JSON format for machine parsing. Every log entry includes a consistent set of base fields:

```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "component": "orchestrator",
  "project_id": "proj-123",
  "message": "Container auto-paused after idle timeout",
  "idle_duration_seconds": 600,
  "trace_id": "abc123def456",
  "span_id": "span-789",
  "request_id": "req-456"
}
```

Base fields present on every entry:

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string (ISO 8601) | Event time in UTC |
| `level` | string | Log level (`error`, `warn`, `info`, `debug`) |
| `component` | string | Emitting component (e.g., `orchestrator`, `api`, `agent`) |
| `message` | string | Human-readable description |
| `trace_id` | string | OpenTelemetry trace ID (when available) |
| `span_id` | string | OpenTelemetry span ID (when available) |

Context fields added when applicable: `project_id`, `session_id`, `request_id`, `container_id`.

### Log Levels

| Level | Usage | Examples |
|-------|-------|---------|
| `error` | Failures requiring attention | Container crash, credential decryption failure, Docker API error |
| `warn` | Degraded state, approaching limits | Health check failure, disk >90%, approaching rate limit |
| `info` | Normal operations | Container created, session started, credential stored |
| `debug` | Verbose diagnostics | gRPC messages, state transitions, config loaded |

### Log Categories

#### Host-Side

| Category | Description | Key Fields |
|----------|-------------|------------|
| `orchestrator.lifecycle` | Container state transitions (maps to [lifecycle events](lifecycle.md)) | `project_id`, `from_state`, `to_state`, `duration_ms` |
| `orchestrator.health` | Health check results | `project_id`, `result`, `latency_ms` |
| `orchestrator.routing` | Request routing decisions | `project_id`, `action` (`route`, `queue`, `reject`) |
| `api.request` | HTTP request/response | `method`, `path`, `status_code`, `latency_ms` |
| `api.websocket` | WebSocket connection lifecycle | `project_id`, `session_id`, `event` (`connected`, `disconnected`) |
| `api.auth` | Authentication/authorization decisions | `user_id`, `action`, `result` |

#### Container-Side

| Category | Description | Key Fields |
|----------|-------------|------------|
| `agent.session` | Session create/complete/fail | `session_id`, `status`, `duration_ms` |
| `agent.command` | Command start/finish | `session_id`, `exit_code`, `duration_ms` |
| `agent.credential` | Credential operations | `session_id`, `operation`, `credential_id` |
| `agent.startup` | Initialization steps | `stage`, `duration_ms` |

### Sensitive Data Policy

**NEVER log:**
- Credential values (tokens, keys, passwords)
- Encryption keys or key material
- Full command stdout/stderr (use session output logs for that)
- User authentication tokens or cookies
- Environment variable values that may contain secrets

**ALWAYS log:**
- Credential identifiers (name, not value)
- User IDs and project IDs
- Session IDs and request IDs
- Exit codes and error messages
- Timing information (durations, latencies)

---

## Distributed Tracing

### Trace Propagation

Trace context is propagated across the full request path:

```
HTTP request → orchestrator → gRPC → container agent
```

- Uses **OpenTelemetry W3C Trace Context** format (`traceparent` / `tracestate` headers)
- gRPC metadata carries trace context between host and container
- `trace_id` and `span_id` are included in all structured log entries for correlation

### Span Hierarchy

```
HTTP Request (external span)
  └── Orchestrator.Route
      ├── EnsureRunning (if container not running)
      │   ├── Docker.Create (if provisioning)
      │   │   └── container.creating → container.running events
      │   ├── Docker.Unpause (if resuming from paused)
      │   │   └── container.resumed event
      │   └── Agent.Ping (readiness check)
      └── Agent.ExecuteCommand (gRPC call)
          ├── CredentialInjection
          ├── CommandExecution
          └── SessionCleanup
```

Each span records:
- `status` — ok / error
- `duration_ms` — wall-clock time
- `project_id` — associated project
- Error details (on failure) — error message, error type

---

## Alerting Rules

Prometheus alerting rules organized by severity. Each rule includes the condition, duration, and routing.

### Critical (PagerDuty / immediate response)

| Alert | Condition | For | Description |
|-------|-----------|-----|-------------|
| `SandboxContainerCrashLooping` | `increase(synchestra_sandbox_restarts_total{project_id=~".+"}[1h]) >= 3` | 0m | Container restarted ≥3 times within 1 hour |
| `SandboxAllContainersDown` | `synchestra_sandbox_containers_active{status="running"} == 0` | 5m | No running containers for 5 minutes |
| `SandboxCredentialDecryptionFailure` | `increase(synchestra_sandbox_credentials_stored_total{status="decryption_error"}[5m]) > 0` | 0m | Any credential decryption error |
| `SandboxDiskFull` | `synchestra_sandbox_disk_usage_bytes / synchestra_sandbox_disk_quota_bytes > 1.0` | 0m | Disk usage exceeds quota |

### Warning (Slack / business hours)

| Alert | Condition | For | Description |
|-------|-----------|-----|-------------|
| `SandboxHighMemoryUsage` | `synchestra_sandbox_memory_usage_bytes / synchestra_sandbox_memory_limit_bytes > 0.9` | 10m | Memory >90% of limit for 10 minutes |
| `SandboxDiskNearFull` | `synchestra_sandbox_disk_usage_bytes / synchestra_sandbox_disk_quota_bytes > 0.9` | 5m | Disk >90% of quota |
| `SandboxHighErrorRate` | `rate(synchestra_sandbox_http_requests_total{status_code=~"5.."}[5m]) / rate(synchestra_sandbox_http_requests_total[5m]) > 0.05` | 5m | HTTP 5xx rate >5% |
| `SandboxSlowProvision` | `histogram_quantile(0.99, rate(synchestra_sandbox_provision_duration_seconds_bucket[5m])) > 30` | 5m | P99 provisioning time >30s |
| `SandboxHealthCheckFailures` | `increase(synchestra_sandbox_health_checks_total{result="fail"}[5m]) > 5` | 0m | >5 health check failures in 5 minutes |
| `SandboxHighEvictionRate` | `increase(synchestra_sandbox_evictions_total[1h]) > 10` | 0m | >10 evictions in 1 hour |

### Info (Dashboard only)

| Alert | Condition | Description |
|-------|-----------|-------------|
| `SandboxContainerCountHigh` | `synchestra_sandbox_containers_total / synchestra_sandbox_max_containers > 0.8` | Container count >80% of max capacity |
| `SandboxHighIdleRatio` | `synchestra_sandbox_containers_active{status="paused"} / synchestra_sandbox_containers_total > 0.5` | >50% of containers are paused |

---

## Dashboards

### Overview Dashboard

Top-level view of sandbox system health. Intended for on-call engineers and ops.

| Panel | Visualization | Query Summary |
|-------|---------------|---------------|
| Containers by State | Stacked gauge | `synchestra_sandbox_containers_active` grouped by `status` |
| Active Sessions | Single stat | `synchestra_sandbox_active_sessions` |
| Request Rate | Time series | `rate(synchestra_sandbox_http_requests_total[5m])` |
| Error Rate | Time series | `rate(synchestra_sandbox_http_requests_total{status_code=~"5.."}[5m])` |
| Provision Latency (P50/P95/P99) | Time series | `histogram_quantile` over `synchestra_sandbox_provision_duration_seconds` |
| Resume Latency (P50/P95/P99) | Time series | `histogram_quantile` over `synchestra_sandbox_resume_duration_seconds` |
| Eviction Rate | Time series | `rate(synchestra_sandbox_evictions_total[5m])` |

### Per-Project Dashboard

Drill-down for a specific `{project_id}`. Uses a dashboard variable for project selection.

| Panel | Visualization | Query Summary |
|-------|---------------|---------------|
| Container Status | State timeline | `synchestra_sandbox_containers_active` filtered by `{project_id}` |
| Container Uptime | Single stat | Derived from lifecycle events |
| Active Sessions | Time series | `synchestra_sandbox_active_sessions` filtered by `{project_id}` |
| Command History | Time series | `rate(synchestra_sandbox_commands_executed_total{project_id="$project_id"}[5m])` |
| Memory Usage vs Limit | Time series + threshold | `synchestra_sandbox_memory_usage_bytes{project_id="$project_id"}` |
| Disk Usage vs Quota | Time series + threshold | `synchestra_sandbox_disk_usage_bytes{project_id="$project_id"}` |
| Credential Operations | Time series | `rate(synchestra_sandbox_credentials_stored_total{project_id="$project_id"}[5m])` |
| Health Check Success Rate | Time series | `synchestra_sandbox_health_checks_total{result="ok"}` / total |

### Operations Dashboard

Internal systems dashboard for debugging infrastructure issues.

| Panel | Visualization | Query Summary |
|-------|---------------|---------------|
| Docker API Latency | Time series | Internal Docker client metrics |
| gRPC Connection Pool | Gauge | `synchestra_sandbox_connection_pool_size` |
| Circuit Breaker States | State map | Per-project circuit breaker status |
| Restart Events | Annotations | `synchestra_sandbox_restarts_total` increases overlaid on timeline |
| Eviction Events | Annotations | `synchestra_sandbox_evictions_total` increases overlaid on timeline |
| Rate Limit Hits | Time series | `rate(synchestra_sandbox_rate_limit_exceeded_total[5m])` by `endpoint` |
| Request Queue Depth | Time series | `synchestra_sandbox_request_queue_depth` by `{project_id}` |

---

## Health Endpoints

### Host Health

| Endpoint | Type | Checks | Failure Meaning |
|----------|------|--------|-----------------|
| `GET /healthz` | Liveness | HTTP server responding | Process needs restart |
| `GET /readyz` | Readiness | Docker reachable, database connected | Not ready to serve traffic |
| `GET /healthz/sandbox` | Subsystem | Orchestrator initialized, ≥1 container reachable | Sandbox feature degraded |

All health endpoints return:

```json
{
  "status": "ok|degraded|unhealthy",
  "checks": {
    "docker": "ok",
    "database": "ok",
    "orchestrator": "ok"
  },
  "timestamp": "2024-01-15T10:30:00.123Z"
}
```

HTTP status codes: `200` for ok, `503` for degraded/unhealthy.

### Container Health

| Mechanism | Check | Interval |
|-----------|-------|----------|
| Docker `HEALTHCHECK` | `synchestra-sandbox-agent health` command | Defined in [Dockerfile.md](Dockerfile.md) |
| gRPC `Ping()` → `PingResponse` | Agent responds with `status: ok\|degraded\|unhealthy` | On demand (routing, readiness) |

---

## Go Package Structure

```
internal/sandbox/observability/
├── metrics.go        // Prometheus metric definitions and registration
├── logging.go        // Structured logger setup with component context
├── tracing.go        // OpenTelemetry tracer initialization
├── alerts.go         // Alert rule definitions (for documentation/testing)
└── health.go         // Health check endpoint handlers
```

- `metrics.go` — registers all `synchestra_sandbox_*` and `synchestra_agent_*` metrics with the Prometheus default registry
- `logging.go` — provides `NewLogger(component string)` that returns a `slog.Logger` pre-configured with JSON output and base fields
- `tracing.go` — initializes the OpenTelemetry `TracerProvider` with W3C propagation and configurable exporter (OTLP)
- `alerts.go` — codifies alert rule definitions for use in integration tests that verify alerting conditions
- `health.go` — implements `/healthz`, `/readyz`, and `/healthz/sandbox` HTTP handlers

---

## Outstanding Questions

1. Should container-side metrics be pushed (via event bus to the host) or pulled (via orchestrator proxy scraping the agent's gRPC `GetStatus`)? Push reduces scrape complexity; pull is more Prometheus-native.
2. Should there be a dedicated log aggregation pipeline (ELK/Loki) or is stdout-based collection (e.g., Docker log driver → CloudWatch/GCP Logging) sufficient for MVP?
3. Should session command output be indexed for search (e.g., "find all sessions that had OOM errors")? This would require a separate log stream with different retention/indexing policies.
4. What OpenTelemetry exporter should be the default — OTLP (gRPC), Jaeger, or stdout for local development?
5. Should alert thresholds (e.g., 90% disk, 5% error rate) be configurable at runtime or fixed at deploy time?
