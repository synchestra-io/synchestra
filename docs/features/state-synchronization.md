# State Synchronization

**Summary:** Synchestra is the single source of truth for task and agent state across a distributed set of agents. All state changes are atomic, versioned, and queryable.

---

## The Problem

When multiple AI agents work on a project simultaneously, state gets messy fast:

- Agent A thinks task X is `in_progress`; Agent B thinks it's `pending`
- An agent crashes mid-task and leaves state dangling
- A human makes a change that agents don't know about
- Two agents try to claim the same task

Synchestra solves this by centralising all state in a single server that agents communicate with synchronously.

---

## State Model

Every resource in Synchestra has a `status` field managed by the server. Transitions are validated —  you can't move a task from `complete` back to `pending` without an explicit override.

### Task States

```
pending → in_progress → complete
                     ↘ failed
                     ↘ blocked
                     ↘ cancelled
```

### Agent States

```
registered → active → idle → offline
```

---

## Optimistic Locking

All update operations include an `updated_at` timestamp. If two agents try to update the same resource simultaneously, the second write will fail with a `409 Conflict` if the `updated_at` they sent doesn't match the server's current value.

```bash
# Agent A reads task, gets updated_at: "2024-01-15T10:00:00Z"
# Agent B reads task, gets updated_at: "2024-01-15T10:00:00Z"
# Agent A updates task —  succeeds, updated_at becomes "2024-01-15T10:01:00Z"
# Agent B tries to update with old updated_at —  409 Conflict
```

---

## Event Log

Every state change is appended to an immutable event log. This gives you:

- Full audit trail of what happened and when
- The ability to reconstruct state at any point in time
- Debugging data when things go wrong

```bash
# View full history for a task
synchestra status task task_abc123

# Via API
GET /api/v1/tasks/task_abc123/history
```

Response:

```json
[
  {"status": "pending",     "at": "2024-01-15T10:00:00Z", "by": "human_xyz"},
  {"status": "in_progress", "at": "2024-01-15T10:05:00Z", "by": "agent_coder1"},
  {"status": "blocked",     "at": "2024-01-15T11:00:00Z", "by": "agent_coder1", "reason": "Waiting for DB schema"},
  {"status": "in_progress", "at": "2024-01-15T11:30:00Z", "by": "human_xyz"},
  {"status": "complete",    "at": "2024-01-15T12:00:00Z", "by": "agent_coder1"}
]
```

---

## Consistency Guarantees

- **Read-your-writes** —  An agent that writes a state change will immediately see that change on subsequent reads
- **Monotonic reads** —  State only moves forward; an agent will never read an older version than one it previously read
- **No phantom updates** —  An update to a resource that doesn't exist returns `404`, not a silent no-op

---

## Sync Patterns for Agents

### Polling

The simplest pattern: agents poll for their assigned tasks on a schedule.

```bash
# In your agent loop:
while true; do
  tasks=$(synchestra task list --agent my-agent --status pending)
  # process tasks...
  sleep 10
done
```

### Webhook Push

Configure Synchestra to push events to your agent's endpoint when tasks are assigned.

See: [Self-Hosting Config](../self-hosting.md) for webhook configuration.

### MCP

If your agent runtime supports MCP (Model Context Protocol), use the Synchestra MCP server for tight integration.

See: [Communication Interfaces](communication.md)

---

## Related

- [Feature: Progress Reporting](progress-reporting.md)
- [CLI: `synchestra status`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/status.md)
- [API: Tasks history](../api/tasks.md#get-task-history)
