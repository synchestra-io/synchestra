# CLI: `synchestra agent`

Register and manage AI agents.

**See also:** [API: Agents](../api/agents.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [register](#register) | Register a new agent |
| [list](#list) | List registered agents |
| [get](#get) | Get agent details |
| [heartbeat](#heartbeat) | Send a heartbeat |
| [deregister](#deregister) | Deregister an agent |

---

## register

Register an agent with Synchestra. Call this on agent startup. Re-registering with the same name updates the existing record.

```
synchestra agent register [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--name` | ✅ | Unique name for this agent |
| `--skills` | | Comma-separated list of skill names this agent has |
| `--description` | | Human-readable description of what this agent does |

### Examples

```bash
# Minimal
synchestra agent register --name "coder-agent-1"

# Full registration
synchestra agent register \
  --name "coder-agent-1" \
  --skills "go,typescript,docker,postgres" \
  --description "Writes, refactors, and reviews Go and TypeScript code"

# Capture agent ID
AGENT_ID=$(synchestra agent register \
  --name "my-agent" \
  --skills "python,pandas" \
  --output json | jq -r .id)

# Use in a startup script
#!/bin/bash
export AGENT_ID=$(synchestra agent register \
  --name "data-agent" \
  --skills "python,sql,data-analysis" \
  --output json | jq -r .id)
echo "Registered as $AGENT_ID"
```

**See also:** [POST /api/v1/agents](../api/agents.md#register-agent)

---

## list

List all registered agents.

```
synchestra agent list [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--skill` | Filter agents by skill name |
| `--status` | Filter by status: `active`, `idle`, `offline` |

### Examples

```bash
# All agents
synchestra agent list

# Find active agents with Go skills
synchestra agent list --skill go --status active

# JSON for scripting
synchestra agent list --status idle --output json
```

**See also:** [GET /api/v1/agents](../api/agents.md#list-agents)

---

## get

Get details for a specific agent.

```
synchestra agent get <id> [flags]
```

### Examples

```bash
synchestra agent get agent_abc123

synchestra agent get agent_abc123 --output json | jq .status
```

**See also:** [GET /api/v1/agents/:id](../api/agents.md#get-agent)

---

## heartbeat

Send a heartbeat to signal the agent is alive. Call this in your agent loop — typically every 30–60 seconds.

```
synchestra agent heartbeat <id> [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--status` | Agent status: `active`, `idle` |
| `--current-task` | ID of the task currently being worked on |

### Examples

```bash
# Simple heartbeat
synchestra agent heartbeat $AGENT_ID

# With status and current task
synchestra agent heartbeat $AGENT_ID \
  --status active \
  --current-task $CURRENT_TASK_ID

# In a background heartbeat loop
while true; do
  synchestra agent heartbeat $AGENT_ID --status active --current-task $TASK_ID
  sleep 30
done &
HEARTBEAT_PID=$!
```

**See also:** [POST /api/v1/agents/:id/heartbeat](../api/agents.md#heartbeat)

---

## deregister

Deregister an agent. Call this on graceful shutdown.

```
synchestra agent deregister <id>
```

### Examples

```bash
synchestra agent deregister $AGENT_ID

# In a shutdown trap
trap "synchestra agent deregister $AGENT_ID; kill $HEARTBEAT_PID" EXIT
```

**See also:** [DELETE /api/v1/agents/:id](../api/agents.md#deregister-agent)
