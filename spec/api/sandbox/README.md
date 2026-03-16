# Sandbox API

REST API endpoints for sandbox container management — command execution, session streaming, credential storage, and admin lifecycle operations. Served by `synchestra serve --http`.

> **Feature specification:** The full design, behavior, and rationale for the sandbox HTTP API live in the feature spec at [`spec/features/sandbox/orchestrator/http-api.md`](../../features/sandbox/orchestrator/http-api.md). This directory contains the formal OpenAPI contract.

## Specification

Full OpenAPI 3.1 specification: [`openapi.yaml`](openapi.yaml)

## Endpoints

All sandbox endpoints are scoped to a project: `/api/v1/sandbox/{project_id}/...`

Admin endpoints use `/api/v1/admin/sandbox/...`.

### Sandbox Operations

| Method | Path | Access Level | Description |
|---|---|---|---|
| `POST` | `/sandbox/{project_id}/execute` | `read_write` | Execute a command in the project's sandbox container |
| `GET` | `/sandbox/{project_id}/status` | `read` | Get container status and resource usage |
| `GET` | `/sandbox/{project_id}/sessions` | `read` | List sessions with filtering and pagination |
| `GET` | `/sandbox/{project_id}/sessions/{session_id}` | `read` | Get detailed session info (reconnection endpoint) |
| `WS` | `/sandbox/{project_id}/sessions/{session_id}/logs` | `read` | Real-time log streaming via WebSocket |
| `POST` | `/sandbox/{project_id}/credentials` | `read_write` | Store or update an encrypted credential |
| `DELETE` | `/sandbox/{project_id}/credentials/{identifier}` | `read_write` | Delete a stored credential |
| `DELETE` | `/sandbox/{project_id}` | `admin` | Destroy container and optionally clear workspace cache |

### Admin Operations

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/sandbox/containers` | List all containers across all projects |
| `POST` | `/admin/sandbox/{project_id}/stop` | Force-stop a container |
| `POST` | `/admin/sandbox/{project_id}/restart` | Force-restart a container |
| `POST` | `/admin/sandbox/{project_id}/evict` | Evict container from active pool |
| `PATCH` | `/admin/sandbox/{project_id}/config` | Update resource limits |
| `PATCH` | `/admin/sandbox/{project_id}/image` | Update container image |

## Access Levels

| Level | Description |
|-------|-------------|
| `read` | View container status, list sessions, stream logs |
| `read_write` | Everything in `read`, plus execute commands and manage credentials |
| `admin` | Everything in `read_write`, plus destroy containers and access admin endpoints |

## Error Codes

| HTTP Status | gRPC Code | Error Code | When |
|---|---|---|---|
| `200 OK` | `OK` | — | Success |
| `201 Created` | `OK` | — | Command accepted |
| `400 Bad Request` | `INVALID_ARGUMENT` | `INVALID_ARGUMENT` | Missing or invalid parameters |
| `401 Unauthorized` | `UNAUTHENTICATED` | `UNAUTHENTICATED` | Invalid or missing auth token |
| `403 Forbidden` | `PERMISSION_DENIED` | `PERMISSION_DENIED` | Insufficient access level |
| `404 Not Found` | `NOT_FOUND` | `NOT_FOUND` | Session or credential not found |
| `429 Too Many Requests` | `RESOURCE_EXHAUSTED` | `RESOURCE_EXHAUSTED` | Rate limit or resource limit exceeded |
| `503 Service Unavailable` | `UNAVAILABLE` | `UNAVAILABLE` | Container temporarily unreachable |
| `504 Gateway Timeout` | `DEADLINE_EXCEEDED` | `DEADLINE_EXCEEDED` | Command or provisioning timed out |

## Outstanding Questions

None at this time — see the [feature spec](../../features/sandbox/orchestrator/http-api.md) for open design questions.
