# Project API

REST API endpoints for server project management. Maps to [`synchestra server project`](../../features/cli/server/project/README.md) CLI commands.

## Specification

Full OpenAPI 3.1 specification: [`openapi.yaml`](openapi.yaml)

## Endpoints

All endpoints are under `/api/v1/project/`.

### Read Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `GET` | `/project/list` | [`synchestra server project list`](../../features/cli/server/project/list/README.md) | List configured projects |

### Mutation Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `POST` | `/project/add` | [`synchestra server project add`](../../features/cli/server/project/add/README.md) | Add a project to the server |

## Error Codes

| HTTP Status | CLI Exit Code | Error Code | When |
|---|---|---|---|
| `200 OK` | 0 | — | Success |
| `201 Created` | 0 | — | Project added |
| `400 Bad Request` | 2 | `invalid_arguments` | Missing or invalid parameters |
| `409 Conflict` | 1 | `conflict` | Project already exists |
| `500 Internal Server Error` | 10+ | `internal_error` | Unexpected failure |

## Outstanding Questions

None at this time.
