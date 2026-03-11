# API: Tasks

Manage tasks — the core unit of work in Synchestra.

**See also:** [CLI: `synchestra task`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/tasks` | [Create task](#create-task) |
| `GET` | `/api/v1/tasks` | [List tasks](#list-tasks) |
| `GET` | `/api/v1/tasks/:id` | [Get task](#get-task) |
| `PUT` | `/api/v1/tasks/:id` | [Update task](#update-task) |
| `DELETE` | `/api/v1/tasks/:id` | [Delete task](#delete-task) |
| `POST` | `/api/v1/tasks/:id/complete` | [Complete task](#complete-task) |
| `POST` | `/api/v1/tasks/:id/fail` | [Fail task](#fail-task) |
| `POST` | `/api/v1/tasks/:id/log` | [Append log](#append-log) |
| `GET` | `/api/v1/tasks/:id/history` | [Get history](#get-task-history) |
| `GET` | `/api/v1/tasks/:id/subtasks` | [List subtasks](#list-subtasks) |

---

## Create Task

`POST /api/v1/tasks`

### Request

```json
{
  "title": "Implement password reset flow",
  "description": "User should be able to reset password via email. Use existing SendGrid integration.",
  "project_id": "proj_abc123",
  "parent_id": "task_parent_xyz",
  "agent_id": "agent_coder1",
  "criteria": "Password reset works end-to-end in staging; unit tests added and passing"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | ✅ | Short title |
| `description` | string | | Detailed description |
| `project_id` | string | | Project to associate with |
| `parent_id` | string | | Parent task ID (for sub-tasks) |
| `agent_id` | string | | Agent to assign to |
| `criteria` | string | | Acceptance criteria |

### Response `201`

```json
{
  "id": "task_def456",
  "title": "Implement password reset flow",
  "description": "User should be able to reset password via email. Use existing SendGrid integration.",
  "project_id": "proj_abc123",
  "parent_id": "task_parent_xyz",
  "agent_id": "agent_coder1",
  "criteria": "Password reset works end-to-end in staging; unit tests added and passing",
  "status": "pending",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**See also:** [CLI: task create](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#create)

---

## List Tasks

`GET /api/v1/tasks`

### Query parameters

| Param | Description |
|---|---|
| `project_id` | Filter by project |
| `status` | Filter by status: `pending`, `in_progress`, `complete`, `failed`, `blocked`, `cancelled` |
| `agent_id` | Filter by assigned agent |
| `limit` | Max results (default 20, max 100) |
| `cursor` | Pagination cursor from previous response |

### Example

```bash
curl "http://localhost:8080/api/v1/tasks?agent_id=agent_coder1&status=pending" \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "tasks": [
    {
      "id": "task_def456",
      "title": "Implement password reset flow",
      "status": "pending",
      "agent_id": "agent_coder1",
      "project_id": "proj_abc123",
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: task list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#list)

---

## Get Task

`GET /api/v1/tasks/:id`

### Example

```bash
curl http://localhost:8080/api/v1/tasks/task_def456 \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "id": "task_def456",
  "title": "Implement password reset flow",
  "description": "User should be able to reset password via email. Use existing SendGrid integration.",
  "project_id": "proj_abc123",
  "parent_id": "task_parent_xyz",
  "agent_id": "agent_coder1",
  "criteria": "Password reset works end-to-end in staging; unit tests added and passing",
  "status": "in_progress",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:05:00Z"
}
```

**See also:** [CLI: task get](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#get)

---

## Update Task

`PUT /api/v1/tasks/:id`

### Request

```json
{
  "title": "Implement password reset flow (revised)",
  "agent_id": "agent_coder2",
  "status": "in_progress"
}
```

All fields are optional. Only provided fields are updated.

### Response `200`

Returns the updated task object.

**See also:** [CLI: task update](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#update)

---

## Delete Task

`DELETE /api/v1/tasks/:id`

Permanently deletes a task and all associated log entries and history. Use with care.

### Response `204`

No content.

---

## Complete Task

`POST /api/v1/tasks/:id/complete`

Mark a task as complete. Transitions status to `complete`.

### Request

```json
{
  "summary": "Implemented password reset using SendGrid. Added 8 unit tests and 2 integration tests, all passing."
}
```

| Field | Required | Description |
|---|---|---|
| `summary` | | Summary of what was done |

### Response `200`

```json
{
  "id": "task_def456",
  "status": "complete",
  "completed_at": "2024-01-15T12:30:00Z",
  "summary": "Implemented password reset using SendGrid. Added 8 unit tests and 2 integration tests, all passing.",
  "updated_at": "2024-01-15T12:30:00Z"
}
```

**See also:** [CLI: task complete](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#complete)

---

## Fail Task

`POST /api/v1/tasks/:id/fail`

Mark a task as failed. Transitions status to `failed`.

### Request

```json
{
  "reason": "SendGrid API returned 403. API key may be expired or rate-limited."
}
```

| Field | Required | Description |
|---|---|---|
| `reason` | ✅ | Why the task failed |

### Response `200`

```json
{
  "id": "task_def456",
  "status": "failed",
  "failed_at": "2024-01-15T11:45:00Z",
  "failure_reason": "SendGrid API returned 403. API key may be expired or rate-limited.",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

**See also:** [CLI: task fail](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#fail)

---

## Append Log

`POST /api/v1/tasks/:id/log`

Append a progress log entry to a task. Log entries are immutable.

### Request

```json
{
  "message": "Analysed existing auth code; identified 3 files to modify",
  "data": {
    "files": ["auth.go", "middleware.go", "email.go"],
    "estimated_changes": "~120 lines"
  }
}
```

| Field | Required | Description |
|---|---|---|
| `message` | ✅ | Human-readable log message |
| `data` | | Structured JSON payload |

### Response `201`

```json
{
  "id": "log_ghi789",
  "task_id": "task_def456",
  "agent_id": "agent_coder1",
  "message": "Analysed existing auth code; identified 3 files to modify",
  "data": {
    "files": ["auth.go", "middleware.go", "email.go"],
    "estimated_changes": "~120 lines"
  },
  "created_at": "2024-01-15T10:07:00Z"
}
```

**See also:** [CLI: task log](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#log)

---

## Get Task History

`GET /api/v1/tasks/:id/history`

Returns the full chronological history for a task: status transitions interleaved with log entries.

### Response `200`

```json
{
  "task_id": "task_def456",
  "history": [
    {
      "type": "status_change",
      "from": "pending",
      "to": "in_progress",
      "at": "2024-01-15T10:05:00Z",
      "by": "agent_coder1"
    },
    {
      "type": "log",
      "id": "log_ghi789",
      "message": "Analysed existing auth code; identified 3 files to modify",
      "data": {"files": ["auth.go", "middleware.go", "email.go"]},
      "at": "2024-01-15T10:07:00Z",
      "by": "agent_coder1"
    },
    {
      "type": "status_change",
      "from": "in_progress",
      "to": "complete",
      "at": "2024-01-15T12:30:00Z",
      "by": "agent_coder1"
    }
  ]
}
```

**See also:** [CLI: status task](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/status.md#task) | [Feature: Progress Reporting](../features/progress-reporting.md)

---

## List Subtasks

`GET /api/v1/tasks/:id/subtasks`

List all direct sub-tasks of a task.

### Response `200`

```json
{
  "parent_id": "task_parent_xyz",
  "subtasks": [
    {
      "id": "task_def456",
      "title": "Implement password reset flow",
      "status": "complete",
      "agent_id": "agent_coder1"
    },
    {
      "id": "task_def457",
      "title": "Write password reset tests",
      "status": "in_progress",
      "agent_id": "agent_tester1"
    }
  ]
}
```

**See also:** [CLI: task subtasks](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#subtasks)
