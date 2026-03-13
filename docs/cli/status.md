# CLI: `synchestra status`

Check system status and inspect individual task history.

**See also:** [API: Status](../api/status.md) | [Feature: Human Steering](../features/human-steering.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [(root)](#root) | Show overall system status |
| [task](#task) | Show task status with full history |

---

## root

Show an overview of the entire Synchestra system: agents, active tasks, and recent activity.

```
synchestra status [flags]
```

### Examples

```bash
synchestra status
```

Output:

```
Synchestra Status
=================
Server:  running (uptime 14h 23m)
Version: 0.3.1

Agents:  4 active, 1 idle, 0 offline
  ● coder-agent-1      active    task_abc1  (35m)
  ● coder-agent-2      active    task_abc3  (2h)
  ● tester-agent       active    task_abc2  (12m)
  ● reviewer-agent     active    task_abc4  (8m)
  ○ deployer-agent     idle      —

Tasks:
  7 active (4 in_progress, 2 pending, 1 blocked)
  12 completed today
  0 failed today

Recent activity:
  14:32  task_abc4   reviewer-agent    "Completed review of PR #42"
  14:30  task_abc2   tester-agent      "Running integration test suite..."
  14:28  task_abc1   coder-agent-1     "Phase 2: writing implementation"
```

```bash
# JSON output for monitoring integrations
synchestra status --output json
```

**See also:** [GET /api/v1/status](../api/status.md)

---

## task

Show full details and history for a specific task — status transitions, progress log entries, and assignments.

```
synchestra status task <id> [flags]
```

### Examples

```bash
synchestra status task task_abc3
```

Output:

```
Task: task_abc3
===============
Title:   Refactor DB layer
Project: my-app
Status:  blocked (since 2024-01-15T12:00:00Z)
Agent:   coder-agent-2
Parent:  task_parent_xyz (Ship DB refactor)

Acceptance criteria:
  All DB queries go through repository interfaces; no raw SQL in service layer

History:
  10:00  [status]  pending → in_progress    coder-agent-2
  10:02  [log]     Started analysis of existing DB code
  10:15  [log]     Found 7 files with raw SQL queries
  10:30  [log]     Refactored UserRepository — done
  11:00  [log]     Blocked: OrderRepository depends on legacy join that needs schema change
  12:00  [status]  in_progress → blocked    coder-agent-2
                   Reason: Waiting for schema migration approval
```

```bash
# JSON for programmatic use
synchestra status task task_abc3 --output json
```

**See also:** [GET /api/v1/tasks/:id/history](../api/tasks.md#get-task-history) | [Feature: Progress Reporting](../features/progress-reporting.md)
