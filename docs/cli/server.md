# CLI: `synchestra server`

Start and manage the Synchestra server process.

**See also:** [Self-Hosting Guide](../self-hosting.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [start](#start) | Start the Synchestra server |

---

## start

Start the Synchestra server. This runs the HTTP API, MCP server, and background processes (heartbeat monitoring, notification dispatch).

```
synchestra server start [flags]
```

### Flags

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--port` | `SYNCHESTRA_PORT` | `8080` | HTTP API port |
| `--mcp-port` | `SYNCHESTRA_MCP_PORT` | `8081` | MCP server port |
| `--db` | `SYNCHESTRA_DB` | `./synchestra.db` | Path to SQLite database file, or Postgres DSN |
| `--config` | `SYNCHESTRA_CONFIG` | `./synchestra.yaml` | Path to config file |
| `--allow-local-unauth` | — | false | Accept unauthenticated requests from localhost |
| `--log-level` | `SYNCHESTRA_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

### Examples

```bash
# Default — SQLite, port 8080
synchestra server start

# Custom port and DB
synchestra server start --port 9090 --db ./data/synchestra.db

# Postgres backend
synchestra server start --db "postgres://user:pass@localhost:5432/synchestra"

# With config file
synchestra server start --config /etc/synchestra/config.yaml

# Development mode (no auth required from localhost)
synchestra server start --allow-local-unauth --log-level debug

# In a systemd unit or Docker container
synchestra server start \
  --port $PORT \
  --db $DATABASE_URL \
  --log-level info
```

### Config file

Rather than passing all flags, use a config file:

```yaml
# synchestra.yaml
server:
  port: 8080
  mcp_port: 8081
  allow_local_unauth: false
  log_level: info

database:
  url: ./synchestra.db   # or postgres DSN

notifications:
  telegram:
    bot_token: "bot123:abc..."
    chat_id: "123456789"
    events:
      - task_failed
      - task_blocked
      - agent_offline
  webhooks:
    - url: "https://your-system.example.com/hooks/synchestra"
      events:
        - task_failed
        - task_complete
```

```bash
synchestra server start --config synchestra.yaml
```

### Running as a background service

```bash
# Simple background process
synchestra server start &

# Or install as a systemd service (see self-hosting guide)
```

For Docker, Kubernetes, and full production setup: [Self-Hosting Guide](../self-hosting.md)
