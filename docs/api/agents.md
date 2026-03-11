# API: Agents

Register and manage AI agents.

**See also:** [CLI: `synchestra agent`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/agents` | [Register agent](#register-agent) |
| `GET` | `/api/v1/agents` | [List agents](#list-agents) |
| `GET` | `/api/v1/agents/:id` | [Get agent](#get-agent) |
| `PUT` | `/api/v1/agents/:id` | [Update agent](#update-agent) |
| `DELETE` | `/api/v1/agents/:id` | [Deregister agent](#deregister-agent) |
| `POST` | `/api/v1/agents/:id/heartbeat` | [Heartbeat](#heartbeat) |

---

## Register Agent

`POST /api/v1/agents`

Register a new agent, or update an existing one with the same name (idempotent).

### Request

```json
{
  "name": "coder-agent-1",
  "skills": ["go", "typescript", "docker"],
  "description": "Writes, refactors, and reviews Go and TypeScript code"
}
```

| Field | Required | Description |
|---|---|---|
| `name` | ✅ | Unique agent name |
| `skills` | | Array of skill names |
| `description` | | Human-readable description |

### Response `201`

```json
{
  "id": "agent_abc123",
  "name": "coder-agent-1",
  "skills": ["go", "typescript", "docker"],
  "description": "Writes, refactors, and reviews Go and TypeScript code",
  "status": "idle",
  "registered_at": "2024-01-15T09:00:00Z",
  "last_heartbeat_at": null,
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: agent register](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#register)

---

## List Agents

`GET /api/v1/agents`

### Query parameters

| Param | Description |
|---|---|
| `skill` | Filter by skill name |
| `status` | Filter by status: `active`, `idle`, `offline` |
| `limit` | Max results (default 20, max 100) |
| `cursor` | Pagination cursor |

### Example

```bash
curl "http://localhost:8080/api/v1/agents?skill=go&status=active" \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "agents": [
    {
      "id": "agent_abc123",
      "name": "coder-agent-1",
      "skills": ["go", "typescript", "docker"],
      "status": "active",
      "current_task_id": "task_def456",
      "last_heartbeat_at": "2024-01-15T14:31:45Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: agent list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#list)

---

## Get Agent

`GET /api/v1/agents/:id`

### Example

```bash
curl http://localhost:8080/api/v1/agents/agent_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "id": "agent_abc123",
  "name": "coder-agent-1",
  "skills": ["go", "typescript", "docker"],
  "description": "Writes, refactors, and reviews Go and TypeScript code",
  "status": "active",
  "current_task_id": "task_def456",
  "registered_at": "2024-01-15T09:00:00Z",
  "last_heartbeat_at": "2024-01-15T14:31:45Z",
  "updated_at": "2024-01-15T14:31:45Z"
}
```

**See also:** [CLI: agent get](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#get)

---

## Update Agent

`PUT /api/v1/agents/:id`

Update agent properties. All fields are optional.

### Request

```json
{
  "skills": ["go", "typescript", "docker", "kubernetes"],
  "description": "Now also handles Kubernetes deployments"
}
```

### Response `200`

Returns the updated agent object.

**See also:** [CLI: agent register](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#register) (re-register to update)

---

## Deregister Agent

`DELETE /api/v1/agents/:id`

Remove an agent from the registry. Call on graceful shutdown. In-progress tasks are flagged for review.

### Response `204`

No content.

**See also:** [CLI: agent deregister](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#deregister)

---

## Heartbeat

`POST /api/v1/agents/:id/heartbeat`

Signal that an agent is alive and report current status. Call every 30–60 seconds.

### Request

```json
{
  "status": "active",
  "current_task_id": "task_def456"
}
```

| Field | Required | Description |
|---|---|---|
| `status` | | `active` or `idle` |
| `current_task_id` | | Task currently being worked on |

### Response `200`

```json
{
  "id": "agent_abc123",
  "status": "active",
  "current_task_id": "task_def456",
  "last_heartbeat_at": "2024-01-15T14:32:00Z"
}
```

**See also:** [CLI: agent heartbeat](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/agent.md#heartbeat) | [Feature: Agent Coordination](../features/agent-coordination.md)
