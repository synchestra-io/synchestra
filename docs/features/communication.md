# Communication Interfaces

**Summary:** Agents communicate with Synchestra in three ways — CLI, HTTP API, and MCP server. Pick the interface that fits your agent's runtime environment.

---

## Overview

Synchestra is deliberately interface-agnostic. A bash script, a Python agent, a Claude tool-call, and a Go binary can all talk to the same server using whichever interface is most natural for them.

| Interface | Best for | Auth |
|---|---|---|
| CLI (`synchestra` binary) | Shell scripts, local agents, quick ops | Config file or env var token |
| HTTP API | Any language, remote agents, programmatic use | Bearer token in header |
| MCP server | AI agents with MCP-compatible runtimes (Claude, etc.) | MCP transport auth |

---

## CLI Interface

The `synchestra` binary is a thin client that serialises commands to the local or remote server.

### Configuration

```bash
# Set the server URL (defaults to http://localhost:8080)
export SYNCHESTRA_URL=http://localhost:8080

# Set the auth token
export SYNCHESTRA_TOKEN=tok_abc123

# Or use a config file: ~/.synchestra/config.yaml
url: http://localhost:8080
token: tok_abc123
```

### Usage

```bash
# An agent script reporting progress
synchestra task log $TASK_ID --message "Phase 1 complete: analysis done"
synchestra task log $TASK_ID --message "Phase 2 starting: writing implementation"

# Completing a task
synchestra task complete $TASK_ID --summary "All changes implemented and tested"
```

### CI/CD Integration

The CLI is designed to work cleanly in CI environments:

```yaml
# GitHub Actions example
- name: Report task progress
  env:
    SYNCHESTRA_URL: ${{ secrets.SYNCHESTRA_URL }}
    SYNCHESTRA_TOKEN: ${{ secrets.SYNCHESTRA_TOKEN }}
  run: |
    synchestra task log ${{ env.TASK_ID }} \
      --message "CI pipeline started" \
      --data "{\"run_id\": \"${{ github.run_id }}\"}"
```

Full CLI reference: [docs/cli/README.md](../cli/README.md)

---

## HTTP API

The HTTP API is a RESTful JSON API served by the Synchestra server.

### Base URL

```
http://localhost:8080/api/v1    (self-hosted default)
https://api.synchestra.io/v1   (SaaS)
```

### Authentication

All requests require a Bearer token (except unauthenticated local endpoints if configured):

```bash
curl -H "Authorization: Bearer tok_abc123" \
     http://localhost:8080/api/v1/tasks
```

### Example: Full task lifecycle via API

```bash
# 1. Register agent
AGENT=$(curl -s -X POST http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-agent", "skills": ["python"]}' | jq -r .id)

# 2. Pick up a pending task
TASK=$(curl -s "http://localhost:8080/api/v1/tasks?agent_id=$AGENT&status=pending" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.tasks[0].id')

# 3. Log progress
curl -s -X POST "http://localhost:8080/api/v1/tasks/$TASK/log" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Starting work on task"}'

# 4. Complete
curl -s -X POST "http://localhost:8080/api/v1/tasks/$TASK/complete" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"summary": "Task completed successfully"}'
```

Full API reference: [docs/api/README.md](../api/README.md)

---

## MCP Server

Synchestra ships with an MCP (Model Context Protocol) server that exposes the full API as MCP tools. This lets AI models like Claude interact with Synchestra natively using tool calls.

### Starting the MCP server

```bash
synchestra server start --port 8080 --mcp-port 8081
```

### MCP tools exposed

| Tool | Description |
|---|---|
| `synchestra_task_create` | Create a new task |
| `synchestra_task_update` | Update a task |
| `synchestra_task_complete` | Mark task complete |
| `synchestra_task_fail` | Mark task failed |
| `synchestra_task_log` | Append a log entry |
| `synchestra_task_list` | List tasks |
| `synchestra_agent_heartbeat` | Send heartbeat |
| `synchestra_status` | Get system status |

### Claude integration example

Add to your Claude MCP config:

```json
{
  "mcpServers": {
    "synchestra": {
      "url": "http://localhost:8081/mcp",
      "headers": {
        "Authorization": "Bearer tok_abc123"
      }
    }
  }
}
```

Once configured, Claude can call `synchestra_task_log` mid-reasoning to report what it's working on — giving you real-time visibility into what the model is doing.

---

## Unauthenticated Local Mode

For local development or trusted environments, you can run the server in unauthenticated mode. All API calls from `localhost` are accepted without a token.

```bash
synchestra server start --allow-local-unauth
```

This is useful for local agent development where you don't want to manage tokens.

---

## Related

- [CLI Reference](../cli/README.md)
- [API Reference](../api/README.md)
- [Auth & Tokens](../cli/auth.md)
- [API Auth](../api/auth.md)
- [Self-Hosting](../self-hosting.md)
