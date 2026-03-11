# API Reference

Synchestra exposes a RESTful JSON API. All endpoints are under `/api/v1/`.

---

## Base URL

```
http://localhost:8080/api/v1     (self-hosted default)
https://api.synchestra.io/v1    (SaaS)
```

---

## Authentication

All requests require a Bearer token in the `Authorization` header:

```
Authorization: Bearer tok_your_token_here
```

Create tokens with: `synchestra auth token create` — see [Auth CLI](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md) or [Auth API](auth.md).

Unauthenticated local access can be enabled on self-hosted instances: see [Self-Hosting](../self-hosting.md).

---

## Endpoints

### Tasks

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/tasks` | Create a task | [tasks.md](tasks.md#create-task) |
| `GET` | `/tasks` | List tasks | [tasks.md](tasks.md#list-tasks) |
| `GET` | `/tasks/:id` | Get a task | [tasks.md](tasks.md#get-task) |
| `PUT` | `/tasks/:id` | Update a task | [tasks.md](tasks.md#update-task) |
| `DELETE` | `/tasks/:id` | Delete a task | [tasks.md](tasks.md#delete-task) |
| `POST` | `/tasks/:id/complete` | Mark complete | [tasks.md](tasks.md#complete-task) |
| `POST` | `/tasks/:id/fail` | Mark failed | [tasks.md](tasks.md#fail-task) |
| `POST` | `/tasks/:id/log` | Append log entry | [tasks.md](tasks.md#append-log) |
| `GET` | `/tasks/:id/history` | Get status history | [tasks.md](tasks.md#get-task-history) |
| `GET` | `/tasks/:id/subtasks` | List sub-tasks | [tasks.md](tasks.md#list-subtasks) |

### Agents

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/agents` | Register agent | [agents.md](agents.md#register-agent) |
| `GET` | `/agents` | List agents | [agents.md](agents.md#list-agents) |
| `GET` | `/agents/:id` | Get agent | [agents.md](agents.md#get-agent) |
| `PUT` | `/agents/:id` | Update agent | [agents.md](agents.md#update-agent) |
| `DELETE` | `/agents/:id` | Deregister agent | [agents.md](agents.md#deregister-agent) |
| `POST` | `/agents/:id/heartbeat` | Send heartbeat | [agents.md](agents.md#heartbeat) |

### Projects

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/projects` | Create project | [projects.md](projects.md#create-project) |
| `GET` | `/projects` | List projects | [projects.md](projects.md#list-projects) |
| `GET` | `/projects/:id` | Get project | [projects.md](projects.md#get-project) |
| `PUT` | `/projects/:id` | Update project | [projects.md](projects.md#update-project) |
| `DELETE` | `/projects/:id` | Delete project | [projects.md](projects.md#delete-project) |

### Repos

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/repos` | Add repo | [repos.md](repos.md#add-repo) |
| `GET` | `/repos` | List repos | [repos.md](repos.md#list-repos) |
| `GET` | `/repos/:id` | Get repo | [repos.md](repos.md#get-repo) |
| `POST` | `/repos/:id/link` | Link to project | [repos.md](repos.md#link-repo) |

### Skills

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/skills` | Create skill | [skills.md](skills.md#create-skill) |
| `GET` | `/skills` | List skills | [skills.md](skills.md#list-skills) |
| `GET` | `/skills/:id` | Get skill | [skills.md](skills.md#get-skill) |

### Rules

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/rules` | Create rule | [rules.md](rules.md#create-rule) |
| `GET` | `/rules` | List rules | [rules.md](rules.md#list-rules) |
| `GET` | `/rules/:id` | Get rule | [rules.md](rules.md#get-rule) |

### Auth

| Method | Path | Description | Docs |
|---|---|---|---|
| `POST` | `/auth/tokens` | Create token | [auth.md](auth.md#create-token) |
| `GET` | `/auth/tokens` | List tokens | [auth.md](auth.md#list-tokens) |
| `DELETE` | `/auth/tokens/:id` | Revoke token | [auth.md](auth.md#revoke-token) |

### Status

| Method | Path | Description | Docs |
|---|---|---|---|
| `GET` | `/status` | System status | [status.md](status.md) |

---

## Common Response Codes

| Code | Meaning |
|---|---|
| `200` | Success |
| `201` | Created |
| `400` | Bad request — missing or invalid fields |
| `401` | Unauthorized — missing or invalid token |
| `403` | Forbidden — token lacks required scope |
| `404` | Not found |
| `409` | Conflict — optimistic lock violation (stale `updated_at`) |
| `500` | Internal server error |

---

## Error Format

All error responses follow this shape:

```json
{
  "error": "task not found",
  "code": "not_found",
  "request_id": "req_abc123"
}
```

---

## Pagination

List endpoints support cursor-based pagination:

```
GET /api/v1/tasks?limit=20&cursor=cursor_abc123
```

Response includes:

```json
{
  "tasks": [...],
  "next_cursor": "cursor_def456",
  "has_more": true
}
```
