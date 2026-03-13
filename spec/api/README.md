# API Specifications

OpenAPI 3.1 specifications for the Synchestra REST API. Each resource has its own directory with an `openapi.yaml` file and a `README.md` describing the resource's API surface.

The feature description and design principles live in [`spec/features/api/`](../features/api/README.md).

## Contents

| Directory | Resource | Status |
|---|---|---|
| [task/](task/README.md) | Task lifecycle operations | In Progress |
| [projects/](projects/README.md) | Server project management | In Progress |

Future resources (not yet specified):

| Directory | Resource | Status |
|---|---|---|
| `agent/` | Agent registration and heartbeat | Planned |
| `auth/` | Token management | Planned |
| `skill/` | Skill registry | Planned |
| `repo/` | Repository management | Planned |

### task

Task lifecycle operations — create, query, and transition tasks through the status model. Every endpoint maps 1:1 to a `synchestra task <action>` CLI command. See [`task/README.md`](task/README.md).

### projects

Server project management — list and add projects to a running server or server configuration. Endpoints map to `synchestra server projects` CLI commands. See [`projects/README.md`](projects/README.md).

## Common Conventions

These conventions apply across all resource specs:

### Base URL

```
/api/v1
```

### Authentication

Bearer token in the `Authorization` header:

```
Authorization: Bearer <token>
```

### Error Response Format

```json
{
  "error": "human-readable message",
  "code": "machine_readable_code",
  "request_id": "req_abc123"
}
```

### Pagination

List endpoints use cursor-based pagination:

```
GET /api/v1/task/list?project=my-project&limit=20&cursor=abc123
```

```json
{
  "items": [...],
  "next_cursor": "def456",
  "has_more": true
}
```

## Outstanding Questions

- Should common schemas (error response, pagination) be extracted into a shared `_common/` directory?
- Should we provide a combined `openapi.yaml` that merges all resource specs for tooling that expects a single file?
