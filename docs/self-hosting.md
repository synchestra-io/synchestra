# Self-Hosting

Run Synchestra on your own infrastructure. It's a single binary or a Docker container. No managed services required.

---

## Quick Start

### Binary

```bash
# Download
curl -sSL https://synchestra.io/install.sh | sh

# Run
synchestra server start --port 8080 --db ./synchestra.db
```

### Docker

```bash
docker run -d \
  --name synchestra \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  synchestra/synchestra:latest
```

---

## Configuration

Synchestra is configured via:
1. A YAML config file (recommended for production)
2. Environment variables
3. CLI flags (override everything)

Priority: CLI flags > environment variables > config file > defaults.

### Full config file reference

```yaml
# synchestra.yaml

server:
  port: 8080                   # HTTP API port
  mcp_port: 8081               # MCP server port (set 0 to disable)
  log_level: info              # debug | info | warn | error
  allow_local_unauth: false    # Accept unauthed requests from 127.0.0.1

database:
  url: ./synchestra.db         # SQLite file path, or Postgres DSN

auth:
  # Initial admin token created on first run (if set)
  # After first run, manage tokens via CLI or API
  bootstrap_token: ""

notifications:
  telegram:
    bot_token: ""              # Telegram bot token
    chat_id: ""                # Telegram chat/channel ID
    events:
      - task_failed
      - task_blocked
      - agent_offline
      # - task_complete        # Uncomment if you want completion notifications
      # - task_created

  webhooks:
    - url: "https://your-system.example.com/hooks/synchestra"
      secret: ""               # HMAC secret for request signing (optional)
      events:
        - task_failed
        - task_complete
      headers:
        Authorization: "Bearer your-webhook-secret"

heartbeat:
  timeout_seconds: 120         # Mark agent offline after this many seconds without heartbeat
```

### Environment variables

All config options map to env vars with the `SYNCHESTRA_` prefix:

| Env var | Equivalent config | Default |
|---|---|---|
| `SYNCHESTRA_PORT` | `server.port` | `8080` |
| `SYNCHESTRA_MCP_PORT` | `server.mcp_port` | `8081` |
| `SYNCHESTRA_DB` | `database.url` | `./synchestra.db` |
| `SYNCHESTRA_LOG_LEVEL` | `server.log_level` | `info` |
| `SYNCHESTRA_BOOTSTRAP_TOKEN` | `auth.bootstrap_token` | — |
| `SYNCHESTRA_CONFIG` | — | `./synchestra.yaml` |

---

## Storage

### SQLite (default)

Zero configuration. Data is stored in a single file. Suitable for single-server self-hosted setups and local development.

```bash
synchestra server start --db ./data/synchestra.db
```

### PostgreSQL

For multi-instance setups or when you need concurrent write performance:

```bash
synchestra server start --db "postgres://user:password@localhost:5432/synchestra?sslmode=require"
```

Or via environment:

```bash
export SYNCHESTRA_DB="postgres://user:password@localhost:5432/synchestra"
synchestra server start
```

Synchestra runs migrations automatically on startup. No manual schema management required.

---

## Authentication

### First Run

On first run, create an admin token:

```bash
# Option 1: Bootstrap token via config (created on first server start)
# synchestra.yaml:
# auth:
#   bootstrap_token: "your-initial-token"

# Option 2: Create after start (if allow_local_unauth: true)
synchestra server start --allow-local-unauth
synchestra auth token create --name "admin" --scopes admin
# Returns: tok_abc123...
# Disable allow_local_unauth now
```

### Token Management

```bash
# Create agent tokens with minimal scopes
synchestra auth token create \
  --name "coder-agent" \
  --scopes "tasks:read,tasks:write,agents:write"

# List all tokens
synchestra auth token list

# Revoke a token
synchestra auth token revoke tok_abc123
```

See: [Auth CLI](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md) | [Auth API](api/auth.md)

---

## Docker

### docker-compose.yml

```yaml
version: "3.8"
services:
  synchestra:
    image: synchestra/synchestra:latest
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - ./data:/data
      - ./synchestra.yaml:/etc/synchestra/config.yaml
    environment:
      SYNCHESTRA_CONFIG: /etc/synchestra/config.yaml
    restart: unless-stopped
```

```bash
docker-compose up -d
```

### With PostgreSQL

```yaml
version: "3.8"
services:
  synchestra:
    image: synchestra/synchestra:latest
    ports:
      - "8080:8080"
    environment:
      SYNCHESTRA_DB: "postgres://synchestra:secret@postgres:5432/synchestra"
      SYNCHESTRA_PORT: "8080"
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: synchestra
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: synchestra
    volumes:
      - pgdata:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  pgdata:
```

---

## Systemd Service

```ini
# /etc/systemd/system/synchestra.service
[Unit]
Description=Synchestra Server
After=network.target

[Service]
Type=simple
User=synchestra
WorkingDirectory=/opt/synchestra
ExecStart=/usr/local/bin/synchestra server start --config /etc/synchestra/config.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable synchestra
sudo systemctl start synchestra
sudo journalctl -u synchestra -f
```

---

## Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl;
    server_name synchestra.your-domain.com;

    ssl_certificate     /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## Backups

### SQLite

```bash
# Simple file copy (safe while server is running — SQLite WAL mode)
cp /data/synchestra.db /backups/synchestra-$(date +%Y%m%d).db

# Or use sqlite3 online backup
sqlite3 /data/synchestra.db ".backup /backups/synchestra-$(date +%Y%m%d).db"
```

### PostgreSQL

```bash
pg_dump synchestra > /backups/synchestra-$(date +%Y%m%d).sql
```

---

## Upgrading

```bash
# Docker
docker pull synchestra/synchestra:latest
docker-compose up -d

# Binary
curl -sSL https://synchestra.io/install.sh | sh
sudo systemctl restart synchestra
```

Synchestra runs database migrations automatically on startup. Always back up your database before upgrading.

---

## Health Check

```bash
curl http://localhost:8080/api/v1/status
# {"server": {"status": "ok", ...}}
```

Use this endpoint for load balancer and container health checks.

---

## Related

- [Server CLI reference](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/server.md)
- [Auth CLI reference](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md)
- [API Reference](api/README.md)
