# Projects API

REST API endpoints for server project management. Maps to [`synchestra server projects`](../../features/cli/server/projects/README.md) CLI commands.

## Specification

Full OpenAPI 3.1 specification: [`openapi.yaml`](openapi.yaml)

## Endpoints

All endpoints are under `/api/v1/projects/`.

### Read Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `GET` | `/projects/list` | [`synchestra server projects`](../../features/cli/server/projects/README.md) | List configured projects |

### Mutation Operations

| Method | Path | CLI Equivalent | Description |
|---|---|---|---|
| `POST` | `/projects/add` | [`synchestra server projects add`](../../features/cli/server/projects/add/README.md) | Add a project to the server |

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
