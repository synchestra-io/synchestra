# Dockerfile for Synchestra Sandbox Agent

## Overview

Multi-stage Docker image that builds the gRPC agent and runtime environment. Optimized for security (minimal base, non-root user, read-only filesystem where possible) and performance.

## Build Arguments

- `GO_VERSION`: Go runtime version (default: 1.21-alpine)

## Build & Push

```bash
# Build locally
docker build -t synchestra/sandbox-agent:latest -f Dockerfile .

# Build with custom Go version
docker build --build-arg GO_VERSION=1.22-alpine -t synchestra/sandbox-agent:1.0.0 -f Dockerfile .

# Push to registry
docker tag synchestra/sandbox-agent:latest myregistry.azurecr.io/synchestra/sandbox-agent:latest
docker push myregistry.azurecr.io/synchestra/sandbox-agent:latest
```

## Security Considerations

- **Base Image**: Alpine Linux (minimal, ~40MB)
- **Non-root User**: Runs as UID 1000 (unprivileged)
- **Dropped Capabilities**: All capabilities dropped (Unix sockets don't require special capabilities)
- **Read-only Root FS**: Except /tmp and /workspace
- **Minimal Package Manager**: Only essential runtime packages installed; consider removing apk in production builds for additional hardening (`RUN rm -rf /sbin/apk /etc/apk /lib/apk /usr/share/apk`)
- **Signed Binaries**: Optional: GPG verification of Go binary and git

## Runtime Configuration

### Environment Variables

```
SYNCHESTRA_PROJECT_ID       # Required: project identifier
SYNCHESTRA_STATE_REPO_URL   # Required: git URL for state repository
SYNCHESTRA_STATE_REPO_TOKEN # Optional: git authentication token
SYNCHESTRA_ENCRYPTION_KEY   # Optional: encryption key (auto-generated if omitted)
SYNCHESTRA_LOG_LEVEL        # Optional: debug, info, warn, error (default: info)
SYNCHESTRA_SOCKET_PATH      # Optional: Unix socket path (default: /var/run/synchestra-{PROJECT_ID}.sock)
SYNCHESTRA_WORK_DIR         # Optional: workspace directory (default: /workspace/{PROJECT_ID})
SYNCHESTRA_SESSION_RETENTION_HOURS # Optional: session log retention (default: 24)
SYNCHESTRA_MEMORY_LIMIT_MB  # Optional: per-session memory limit (default: 100)
SYNCHESTRA_CPU_LIMIT        # Optional: per-session CPU limit (default: 0.5)
```

### Volumes

```bash
# Required: project workspace (bind mount from host)
-v /var/lib/synchestra/workspaces/{project_id}:/workspace/{project_id}

# Optional: host docker socket (for docker-in-docker)
-v /var/run/docker.sock:/var/run/docker.sock:ro

# Optional: credentials from host (for read-only mount)
-v /etc/synchestra/credentials/{project_id}:/secure/credentials:ro
```

### Ports

- **Unix Socket**: `/var/run/synchestra-{project_id}.sock` (no network port by default)
- **TCP (if enabled)**: `50051` (localhost only, use --network host if needed)

## Example Runtime

```bash
# Note: --cap-drop=ALL is sufficient; no --cap-add needed since the agent uses Unix sockets.
docker run \
  --name synchestra-sandbox-${PROJECT_ID} \
  --rm \
  --cap-drop=ALL \
  --read-only \
  --tmpfs /run:noexec,nosuid \
  --tmpfs /tmp:noexec,nosuid \
  --volume /var/lib/synchestra/workspaces/${PROJECT_ID}:/workspace/${PROJECT_ID}:rw \
  --volume /var/run:/var/run:rw \
  --user 1000:1000 \
  --memory 512m \
  --cpus 2 \
  --pids-limit 256 \
  -e SYNCHESTRA_PROJECT_ID=${PROJECT_ID} \
  -e SYNCHESTRA_STATE_REPO_URL=${STATE_REPO_URL} \
  -e SYNCHESTRA_LOG_LEVEL=info \
  synchestra/sandbox-agent:latest
```

## Health Check

```bash
# Inside container or from host
synchestra-sandbox-agent health \
  --socket /var/run/synchestra-${PROJECT_ID}.sock \
  --timeout 5s
```

Returns exit code 0 if healthy, non-zero otherwise.
