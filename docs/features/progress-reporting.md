# Progress Reporting

**Summary:** Agents append structured log entries to tasks as they work. Humans and other agents can query this log to understand current state, recent activity, and the full history of any task.

---

## Overview

Progress reporting solves the visibility problem: when an AI agent is working on a task, what's it actually doing right now? With Synchestra, the agent tells you — continuously, in a queryable log.

A progress log entry is a timestamped record with a human-readable message and optional structured data. Entries are immutable; the log is append-only.

---

## Appending Progress

### CLI

```bash
synchestra task log task_abc123 \
  --message "Analysed codebase; found 3 auth-related files" \
  --data '{"files": ["auth.go", "middleware.go", "tokens.go"]}'
```

### API

```bash
POST /api/v1/tasks/task_abc123/log
{
  "message": "Analysed codebase; found 3 auth-related files",
  "data": {
    "files": ["auth.go", "middleware.go", "tokens.go"]
  }
}
```

See: [CLI task log](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#log) | [API task log](../api/tasks.md#append-log)

---

## Log Entry Format

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique entry ID |
| `task_id` | string | The task this log belongs to |
| `agent_id` | string | The agent that wrote this entry (if applicable) |
| `message` | string | Human-readable description of what happened |
| `data` | object | Optional structured payload — anything your agent wants to record |
| `created_at` | string | ISO 8601 timestamp |

---

## Querying Progress

### Get status with history

```bash
synchestra status task task_abc123
```

Output:

```
Task: task_abc123
Title: Implement user authentication
Status: in_progress (since 2024-01-15T10:05:00Z)
Agent: coder-agent-1

Progress log:
  10:05  Started analysis of existing auth code
  10:12  Analysed codebase; found 3 auth-related files
  10:25  Drafted JWT middleware implementation
  10:40  Writing unit tests...
```

### Via API

```bash
GET /api/v1/tasks/task_abc123/history
```

Returns both status transitions and log entries in chronological order.

---

## Status Transitions as Progress

Status changes are also recorded in the history. The full timeline interleaves log entries and status transitions:

```json
[
  {
    "type": "status_change",
    "from": "pending",
    "to": "in_progress",
    "at": "2024-01-15T10:05:00Z",
    "by": "agent_coder1"
  },
  {
    "type": "log",
    "message": "Started analysis of existing auth code",
    "at": "2024-01-15T10:05:30Z",
    "by": "agent_coder1"
  },
  {
    "type": "log",
    "message": "Analysed codebase; found 3 auth-related files",
    "data": {"files": ["auth.go", "middleware.go", "tokens.go"]},
    "at": "2024-01-15T10:12:00Z",
    "by": "agent_coder1"
  }
]
```

---

## Recommendations for Agent Authors

**Log frequently.** AI agents can run for minutes or hours on a single task. A log entry every few steps gives humans confidence that things are progressing.

**Log meaningfully.** "Working..." is noise. "Analysed 3 files, writing implementation for `auth.go`" is signal.

**Use `data` for machine-readable state.** If a downstream agent or human might want to act on the data (file paths, counts, URLs, error codes), put it in the `data` field as structured JSON rather than embedding it in the message string.

**Log on failure before failing.** Before calling `task fail`, append a log entry explaining what went wrong and what was tried. This creates a useful post-mortem trail.

```bash
synchestra task log task_abc123 \
  --message "Failed to compile: missing dependency 'github.com/foo/bar'" \
  --data '{"error": "exit code 1", "missing_dep": "github.com/foo/bar"}'

synchestra task fail task_abc123 \
  --reason "Build failed due to missing dependency"
```

---

## Related

- [CLI: `synchestra task log`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/task.md#log)
- [CLI: `synchestra status`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/status.md)
- [API: Append log entry](../api/tasks.md#append-log)
- [API: Get task history](../api/tasks.md#get-task-history)
- [Feature: State Synchronization](state-synchronization.md)
- [Feature: Human Steering](human-steering.md)
