# CLI: `synchestra task`

Manage tasks — the core unit of work in Synchestra.

**See also:** [API: Tasks](../api/tasks.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [create](#create) | Create a new task |
| [list](#list) | List tasks with filters |
| [get](#get) | Get task details |
| [update](#update) | Update task fields |
| [complete](#complete) | Mark task complete |
| [fail](#fail) | Mark task failed |
| [log](#log) | Append a progress log entry |
| [subtasks](#subtasks) | List sub-tasks of a task |

---

## create

Create a new task.

```
synchestra task create [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--title` | ✅ | Short title for the task |
| `--description` | | Longer description of what needs to be done |
| `--project` | | Project ID or name to associate this task with |
| `--parent` | | Parent task ID (for sub-tasks) |
| `--agent` | | Agent ID or name to assign the task to |
| `--criteria` | | Acceptance criteria — what "done" looks like |

### Examples

```bash
# Minimal
synchestra task create --title "Fix login redirect bug"

# Full
synchestra task create \
  --title "Implement password reset flow" \
  --description "User should be able to reset password via email link. Use existing email service." \
  --project my-app \
  --agent coder-agent-1 \
  --criteria "Password reset works end-to-end in staging; unit tests added"

# Sub-task
synchestra task create \
  --title "Write unit tests for password reset" \
  --parent task_abc123 \
  --agent tester-agent

# Capture ID for use in scripts
TASK_ID=$(synchestra task create --title "Deploy hotfix" --output json | jq -r .id)
```

**See also:** [POST /api/v1/tasks](../api/tasks.md#create-task)

---

## list

List tasks, optionally filtered.

```
synchestra task list [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--project` | Filter by project ID or name |
| `--status` | Filter by status: `pending`, `in_progress`, `complete`, `failed`, `blocked`, `cancelled` |
| `--agent` | Filter by agent ID or name |

### Examples

```bash
# All tasks
synchestra task list

# Pending tasks for my agent
synchestra task list --agent my-agent --status pending

# All tasks in a project
synchestra task list --project my-app

# JSON output for scripting
synchestra task list --agent my-agent --status pending --output json
```

**See also:** [GET /api/v1/tasks](../api/tasks.md#list-tasks)

---

## get

Get full details for a single task.

```
synchestra task get <id> [flags]
```

### Examples

```bash
synchestra task get task_abc123

# Get a specific field
synchestra task get task_abc123 --output json | jq .status
```

**See also:** [GET /api/v1/tasks/:id](../api/tasks.md#get-task)

---

## update

Update one or more fields of an existing task.

```
synchestra task update <id> [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--title` | New title |
| `--description` | New description |
| `--agent` | Reassign to a different agent |
| `--status` | Force a status change (use with care) |

### Examples

```bash
# Reassign to different agent
synchestra task update task_abc123 --agent coder-agent-2

# Add more context to description
synchestra task update task_abc123 \
  --description "Focus on the UserRepository only. Skip OrderRepository."

# Unblock a stuck task
synchestra task update task_abc123 --status in_progress
```

**See also:** [PUT /api/v1/tasks/:id](../api/tasks.md#update-task)

---

## complete

Mark a task as complete. Optionally include a summary of what was done.

```
synchestra task complete <id> [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--summary` | Human-readable summary of what was completed |

### Examples

```bash
synchestra task complete task_abc123

synchestra task complete task_abc123 \
  --summary "Implemented password reset. Sent email via SendGrid. Added 12 unit tests, all passing."
```

**See also:** [POST /api/v1/tasks/:id/complete](../api/tasks.md#complete-task)

---

## fail

Mark a task as failed. Always include a reason.

```
synchestra task fail <id> [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--reason` | ✅ | Why the task failed |

### Examples

```bash
synchestra task fail task_abc123 --reason "Email service returned 503 after 3 retries"

synchestra task fail task_abc123 \
  --reason "Could not find DB migration file; migration history may be corrupt"
```

**See also:** [POST /api/v1/tasks/:id/fail](../api/tasks.md#fail-task)

---

## log

Append a progress log entry to a task. Use this frequently to give humans (and other agents) visibility into what you're doing.

```
synchestra task log <id> [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--message` | ✅ | Human-readable log message |
| `--data` | | JSON object with structured data to attach |

### Examples

```bash
# Simple progress note
synchestra task log task_abc123 --message "Analysing existing codebase"

# With structured data
synchestra task log task_abc123 \
  --message "Found 3 files to modify" \
  --data '{"files": ["auth.go", "middleware.go", "tokens.go"]}'

# Logging a decision
synchestra task log task_abc123 \
  --message "Chose JWT over sessions: stateless, easier to scale" \
  --data '{"decision": "JWT", "alternatives_considered": ["sessions", "opaque_tokens"]}'
```

**See also:** [POST /api/v1/tasks/:id/log](../api/tasks.md#append-log)

---

## subtasks

List all sub-tasks of a given parent task.

```
synchestra task subtasks <id> [flags]
```

### Examples

```bash
synchestra task subtasks task_abc123

synchestra task subtasks task_abc123 --output json
```

**See also:** [GET /api/v1/tasks/:id/subtasks](../api/tasks.md#list-subtasks)
