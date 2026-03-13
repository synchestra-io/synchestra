# Feature: REST API

The REST API exposes Synchestra's coordination capabilities over HTTP, mirroring the CLI contract with JSON request/response semantics. It is the primary interface for the web UI, external integrations, and agents that prefer HTTP over direct git access.

## Relationship to CLI

The API is a thin HTTP layer over the same operations the CLI performs. Every mutation endpoint maps 1:1 to a CLI command. The API server translates HTTP requests into the equivalent git-backed atomic operations (pull → verify → mutate → commit → push).

| Concept | CLI | API |
|---|---|---|
| Resource identification | `--project`, `--task` flags | `?project=`, `?task=` query params |
| Error signaling | Exit codes (0–4, 10+) | HTTP status codes + error body |
| Output format | `--format` flag | `Accept` header / always JSON |
| Authentication | Local git identity | Bearer token |
| Concurrency control | Push conflict → retry | Optimistic locking via `409 Conflict` |

## Specification

The normative API specification lives in [`spec/api/`](../../api/README.md) as OpenAPI 3.1 files, one per resource:

| Resource | Spec | Status |
|---|---|---|
| [Task](../../api/task/README.md) | [`spec/api/task/openapi.yaml`](../../api/task/openapi.yaml) | In Progress |

## Design Principles

1. **CLI parity** — every CLI mutation command has exactly one API endpoint; read commands map to GET endpoints.
2. **Action-oriented endpoints** — state transitions use dedicated `POST /task/{action}` endpoints, not generic PATCH.
3. **Query-param identification** — project and task identifiers are query params, matching CLI flag conventions and avoiding URL-encoding issues with hierarchical task paths.
4. **Atomic semantics** — each mutation is an atomic commit-and-push; on conflict the server returns `409` with the current state.
5. **Consistent error model** — all errors return `{error, code, request_id}` with HTTP status codes mapping to CLI exit codes.

## Exit Code → HTTP Status Mapping

| CLI Exit Code | Meaning | HTTP Status |
|---|---|---|
| 0 | Success | `200 OK` / `201 Created` |
| 1 | Conflict | `409 Conflict` |
| 2 | Invalid arguments | `400 Bad Request` |
| 3 | Not found | `404 Not Found` |
| 4 | Invalid state transition | `422 Unprocessable Entity` |
| 10+ | Unexpected error | `500 Internal Server Error` |

## Outstanding Questions

- Should the API support WebSocket or SSE for real-time task status updates?
- Should batch operations be supported (e.g., enqueue multiple tasks in one request)?
- How should the API handle long-running git operations — synchronous response or async with polling?
