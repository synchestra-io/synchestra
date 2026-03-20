# CLI Reference

The `synchestra` CLI is how agents and humans interact with the Synchestra server from the terminal.

---

## Setup

```bash
# Set server URL (default: http://localhost:8080)
export SYNCHESTRA_URL=http://localhost:8080

# Set auth token
export SYNCHESTRA_TOKEN=tok_your_token_here

# Or configure via file: ~/.synchestra/config.yaml
```

---

## Commands

| Command | Description | Docs |
|---|---|---|
| `synchestra task new` | Create a new task | [task.md](task.md) |
| `synchestra task list` | List tasks with filters | [task.md](task.md#list) |
| `synchestra task get <id>` | Get task details | [task.md](task.md#get) |
| `synchestra task update <id>` | Update task fields | [task.md](task.md#update) |
| `synchestra task complete <id>` | Mark task complete | [task.md](task.md#complete) |
| `synchestra task fail <id>` | Mark task failed | [task.md](task.md#fail) |
| `synchestra task log <id>` | Append a progress log entry | [task.md](task.md#log) |
| `synchestra task subtasks <id>` | List sub-tasks of a task | [task.md](task.md#subtasks) |
| `synchestra agent register` | Register an agent | [agent.md](agent.md) |
| `synchestra agent list` | List registered agents | [agent.md](agent.md#list) |
| `synchestra agent get <id>` | Get agent details | [agent.md](agent.md#get) |
| `synchestra agent heartbeat <id>` | Send a heartbeat | [agent.md](agent.md#heartbeat) |
| `synchestra agent deregister <id>` | Deregister an agent | [agent.md](agent.md#deregister) |
| `synchestra project create` | Create a project | [project.md](project.md) |
| `synchestra project list` | List projects | [project.md](project.md#list) |
| `synchestra project get <id>` | Get project details | [project.md](project.md#get) |
| `synchestra project update <id>` | Update a project | [project.md](project.md#update) |
| `synchestra repo add` | Add a repository | [repo.md](repo.md) |
| `synchestra repo list` | List repos | [repo.md](repo.md#list) |
| `synchestra repo get <id>` | Get repo details | [repo.md](repo.md#get) |
| `synchestra repo link <id>` | Link repo to a project | [repo.md](repo.md#link) |
| `synchestra skill create` | Create a skill definition | [skill.md](skill.md) |
| `synchestra skill list` | List skills | [skill.md](skill.md#list) |
| `synchestra skill get <id>` | Get skill details | [skill.md](skill.md#get) |
| `synchestra rule create` | Create a rule | [rule.md](rule.md) |
| `synchestra rule list` | List rules | [rule.md](rule.md#list) |
| `synchestra rule get <id>` | Get rule details | [rule.md](rule.md#get) |
| `synchestra status` | Show overall system status | [status.md](status.md) |
| `synchestra status task <id>` | Show task status with history | [status.md](status.md#task) |
| `synchestra server start` | Start the Synchestra server | [server.md](server.md) |
| `synchestra auth token create` | Create an API token | [auth.md](auth.md) |
| `synchestra auth token list` | List API tokens | [auth.md](auth.md#list) |
| `synchestra auth token revoke <id>` | Revoke a token | [auth.md](auth.md#revoke) |

---

## Global Flags

These flags apply to all commands:

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--url` | `SYNCHESTRA_URL` | `http://localhost:8080` | Synchestra server URL |
| `--token` | `SYNCHESTRA_TOKEN` | — | API auth token |
| `--output` | — | `text` | Output format: `text`, `json`, `yaml` |
| `--quiet` | — | false | Suppress all output except errors |
| `--config` | `SYNCHESTRA_CONFIG` | `~/.synchestra/config.yaml` | Path to config file |

---

## Output Formats

All commands support `--output json` for machine-readable output, useful in agent scripts:

```bash
# Get task ID from creation output
TASK_ID=$(synchestra task new --title "My task" --output json | jq -r .id)

# Get agent status
STATUS=$(synchestra agent get $AGENT_ID --output json | jq -r .status)
```
