# API: Status

Get a high-level view of the Synchestra system.

**See also:** [CLI: `synchestra status`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/status.md) | [Feature: Human Steering](../features/human-steering.md)

---

## Endpoint

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/status` | [System status](#system-status) |

---

## System Status

`GET /api/v1/status`

Returns an overview of the running Synchestra instance: server health, agent counts, and task summary.

### Example

```bash
curl http://localhost:8080/api/v1/status \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "server": {
    "status": "ok",
    "version": "0.3.1",
    "uptime_seconds": 51780,
    "started_at": "2024-01-15T00:09:00Z"
  },
  "agents": {
    "total": 5,
    "active": 4,
    "idle": 1,
    "offline": 0
  },
  "tasks": {
    "active": 7,
    "by_status": {
      "pending": 2,
      "in_progress": 4,
      "blocked": 1
    },
    "completed_today": 12,
    "failed_today": 0
  },
  "recent_activity": [
    {
      "type": "task_complete",
      "task_id": "task_abc4",
      "task_title": "Review PR #42",
      "agent_id": "agent_reviewer1",
      "at": "2024-01-15T14:32:00Z"
    },
    {
      "type": "task_log",
      "task_id": "task_abc2",
      "task_title": "Write auth tests",
      "agent_id": "agent_tester1",
      "message": "Running integration test suite...",
      "at": "2024-01-15T14:30:00Z"
    }
  ]
}
```

### Response fields

| Field | Description |
|---|---|
| `server.status` | `ok` or `degraded` |
| `server.version` | Synchestra server version |
| `server.uptime_seconds` | Seconds since server start |
| `agents.total` | Total registered agents |
| `agents.active` | Agents currently working on a task |
| `agents.idle` | Agents registered but not assigned a task |
| `agents.offline` | Agents that missed heartbeat threshold |
| `tasks.active` | Tasks currently in a non-terminal state |
| `tasks.by_status` | Breakdown of active tasks by status |
| `tasks.completed_today` | Tasks completed in the last 24 hours |
| `tasks.failed_today` | Tasks failed in the last 24 hours |
| `recent_activity` | Last 10 notable events |

**See also:** [CLI: status](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/status.md)
