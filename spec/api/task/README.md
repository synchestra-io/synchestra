# Task API

REST API endpoints for task lifecycle operations. Every endpoint maps 1:1 to a [`synchestra task <action>`](../../features/cli/task/README.md) CLI command.

## Specification

Full OpenAPI 3.1 specification: [`openapi.yaml`](openapi.yaml)

## Endpoints

All endpoints are under `/api/v1/task/`. Project and task identifiers are passed as query parameters.

### Read Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `GET` | `/task/list` | [`synchestra task list`](../../features/cli/task/list/README.md) | List tasks with optional filtering |
| `GET` | `/task/info` | [`synchestra task info`](../../features/cli/task/info/README.md) | Get full task details |
| `GET` | `/task/status` | [`synchestra task status`](../../features/cli/task/status/README.md) (query mode) | Get current task status |

### Mutation Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `POST` | `/task/create` | [`synchestra task create`](../../features/cli/task/create/README.md) | Create a new task |
| `POST` | `/task/enqueue` | [`synchestra task enqueue`](../../features/cli/task/enqueue/README.md) | Move task from `planning` → `queued` |
| `POST` | `/task/claim` | [`synchestra task claim`](../../features/cli/task/claim/README.md) | Claim a queued task |
| `POST` | `/task/start` | [`synchestra task start`](../../features/cli/task/start/README.md) | Begin work: `claimed` → `in_progress` |
| `POST` | `/task/complete` | [`synchestra task complete`](../../features/cli/task/complete/README.md) | Mark task completed |
| `POST` | `/task/fail` | [`synchestra task fail`](../../features/cli/task/fail/README.md) | Mark task failed |
| `POST` | `/task/block` | [`synchestra task block`](../../features/cli/task/block/README.md) | Mark task blocked |
| `POST` | `/task/unblock` | [`synchestra task unblock`](../../features/cli/task/unblock/README.md) | Resume blocked task |
| `POST` | `/task/release` | [`synchestra task release`](../../features/cli/task/release/README.md) | Release claimed task back to queue |
| `POST` | `/task/abort` | [`synchestra task abort`](../../features/cli/task/abort/README.md) | Request task abortion (sets flag) |
| `POST` | `/task/aborted` | [`synchestra task aborted`](../../features/cli/task/aborted/README.md) | Confirm task was aborted |
| `POST` | `/task/status` | [`synchestra task status`](../../features/cli/task/status/README.md) (update mode) | Generic status transition |

## Common Parameters

| Parameter | In | Type | Required | Description |
|---|---|---|---|---|
| `project` | query | string | Yes (except where noted) | Project identifier. Maps to CLI `--project`. |
| `task` | query | string | Yes (except `list`) | Hierarchical task path (e.g., `parent/subtask`). Maps to CLI `--task`. |

## Status Transitions

The API enforces the same state machine as the CLI:

```
planning → queued → claimed → in_progress → completed
                                           → failed
                                           → blocked → in_progress
                                           → aborted
```

Invalid transitions return `422 Unprocessable Entity`.

## Error Codes

| HTTP Status | CLI Exit Code | Error Code | When |
|---|---|---|---|
| `200 OK` | 0 | — | Success |
| `201 Created` | 0 | — | Task created |
| `400 Bad Request` | 2 | `invalid_arguments` | Missing or invalid parameters |
| `404 Not Found` | 3 | `not_found` | Task or project not found |
| `409 Conflict` | 1 | `conflict` | Push conflict or claim race |
| `422 Unprocessable Entity` | 4 | `invalid_state_transition` | Status guard failed |
| `500 Internal Server Error` | 10+ | `internal_error` | Unexpected failure |

## Outstanding Questions

- Should `GET /task/status` and `POST /task/status` be split into separate paths to avoid method overloading on the same path?
- Should the API return the full task object in mutation responses, or just a status acknowledgment?
