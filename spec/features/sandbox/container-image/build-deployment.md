# Container Image Build & Deployment

## Overview

> **Related documents:** [Dockerfile.spec](Dockerfile.spec) (Dockerfile), [docker-entrypoint.sh](docker-entrypoint.sh) (entrypoint script), [build-automation.md](build-automation.md) (CI/CD workflow).

The Sandbox Agent runs in a Docker container, one per Synchestra project. This document covers image building, deployment, security hardening, and operational considerations.

## Image Build

### Prerequisites

- Docker 20.10+ or compatible runtime
- Go 1.21+ (for build-time compilation only)
- This repository (synchestra)

### Building Locally

```bash
cd /path/to/synchestra

# Build image
docker build \
  -t synchestra/sandbox-agent:latest \
  -f build/Dockerfile.sandbox-agent \
  .

# Verify image
docker images | grep sandbox-agent
docker inspect synchestra/sandbox-agent:latest
```

### Building with Custom Versions

```bash
# Custom Go version
docker build \
  --build-arg GO_VERSION=1.22-alpine \
  -t synchestra/sandbox-agent:latest \
  -f build/Dockerfile.sandbox-agent \
  .

# Custom Alpine version
docker build \
  --build-arg ALPINE_VERSION=3.20 \
  -t synchestra/sandbox-agent:latest \
  -f build/Dockerfile.sandbox-agent \
  .

# With build labels for versioning
docker build \
  --label "version=1.0.0" \
  --label "commit=$(git rev-parse HEAD)" \
  --label "build_date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
  -t synchestra/sandbox-agent:1.0.0 \
  -f build/Dockerfile.sandbox-agent \
  .
```

### Multi-Platform Builds (Buildx)

```bash
# Enable buildx (if not already enabled)
docker buildx create --name sandbox-builder
docker buildx use sandbox-builder

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t synchestra/sandbox-agent:latest \
  --push \
  -f build/Dockerfile.sandbox-agent \
  .
```

### Container Registry Push

```bash
# Login to registry (Azure, Docker Hub, ECR, etc.)
docker login myregistry.azurecr.io

# Tag for registry
docker tag synchestra/sandbox-agent:latest \
  myregistry.azurecr.io/synchestra/sandbox-agent:latest

# Push
docker push myregistry.azurecr.io/synchestra/sandbox-agent:latest

# Verify push
docker pull myregistry.azurecr.io/synchestra/sandbox-agent:latest
```

## Image Scanning & Security

### Vulnerability Scanning (Trivy)

```bash
# Scan image for vulnerabilities
trivy image synchestra/sandbox-agent:latest

# Fail build if critical vulnerabilities found
trivy image --severity CRITICAL --exit-code 1 synchestra/sandbox-agent:latest

# Generate SBOM (Software Bill of Materials)
trivy image --format json synchestra/sandbox-agent:latest > sbom.json
```

### Verify Image Provenance

```bash
# Use Docker Content Trust (DCT) to sign image
export DOCKER_CONTENT_TRUST=1
docker tag synchestra/sandbox-agent:latest \
  myregistry.azurecr.io/synchestra/sandbox-agent:latest
docker push myregistry.azurecr.io/synchestra/sandbox-agent:latest

# Verify signature
docker pull --disable-content-trust=false myregistry.azurecr.io/synchestra/sandbox-agent:latest
```

## Container Runtime Deployment

### Docker (Local/Single Host)

```bash
PROJECT_ID="my-project"
STATE_REPO_URL="https://github.com/user/my-project-synchestra.git"

docker run \
  --name synchestra-sandbox-${PROJECT_ID} \
  --rm \
  --detach \
  \
  # Security options
  --cap-drop=ALL \
  # No capabilities needed — agent uses Unix socket, not privileged ports
  --read-only \
  --tmpfs /run:noexec,nosuid \
  --tmpfs /tmp:noexec,nosuid \
  --security-opt=seccomp=default \
  \
  # Volume mounts
  -v /var/lib/synchestra/workspaces/${PROJECT_ID}:/workspace/${PROJECT_ID}:rw \
  -v /var/run:/var/run:rw \
  \
  # Resource limits
  --memory 512m \
  --memory-swap 512m \
  --cpus 2.0 \
  --pids-limit 256 \
  \
  # Networking
  --network bridge \
  \
  # User
  --user 1000:1000 \
  \
  # Environment
  -e SYNCHESTRA_PROJECT_ID=${PROJECT_ID} \
  -e SYNCHESTRA_STATE_REPO_URL=${STATE_REPO_URL} \
  -e SYNCHESTRA_LOG_LEVEL=info \
  -e SYNCHESTRA_SESSION_RETENTION_HOURS=24 \
  \
  # Image
  synchestra/sandbox-agent:latest
```

### Docker Compose (Multi-Project)

```yaml
# docker-compose.sandbox.yml
version: '3.9'

services:
  sandbox-project-a:
    image: synchestra/sandbox-agent:latest
    container_name: synchestra-sandbox-project-a
    restart: unless-stopped
    
    cap_drop:
      - ALL
    # No capabilities needed — agent uses Unix socket, not privileged ports
    
    read_only: true
    tmpfs:
      - /run:noexec,nosuid
      - /tmp:noexec,nosuid
    
    volumes:
      - /var/lib/synchestra/workspaces/project-a:/workspace/project-a:rw
      - /var/run:/var/run:rw
    
    environment:
      SYNCHESTRA_PROJECT_ID: project-a
      SYNCHESTRA_STATE_REPO_URL: https://github.com/user/project-a-synchestra.git
      SYNCHESTRA_LOG_LEVEL: info
    
    mem_limit: 512m
    cpus: 2.0
    pids_limit: 256
    
    healthcheck:
      test: ['CMD', 'synchestra-sandbox-agent', 'health']
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 5s
  
  sandbox-project-b:
    image: synchestra/sandbox-agent:latest
    container_name: synchestra-sandbox-project-b
    # ... similar configuration for project-b
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: synchestra-sandbox-project-a
  namespace: synchestra
  labels:
    app: synchestra-sandbox
    project: project-a
spec:
  replicas: 1
  selector:
    matchLabels:
      app: synchestra-sandbox
      project: project-a
  
  template:
    metadata:
      labels:
        app: synchestra-sandbox
        project: project-a
    
    spec:
      serviceAccountName: synchestra-sandbox
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      
      containers:
      - name: sandbox-agent
        image: myregistry.azurecr.io/synchestra/sandbox-agent:latest
        imagePullPolicy: IfNotPresent
        
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
            # No capabilities needed — agent uses Unix socket, not privileged ports
          readOnlyRootFilesystem: true
        
        env:
        - name: SYNCHESTRA_PROJECT_ID
          value: "project-a"
        - name: SYNCHESTRA_STATE_REPO_URL
          valueFrom:
            secretKeyRef:
              name: project-a-config
              key: state-repo-url
        - name: SYNCHESTRA_STATE_REPO_TOKEN
          valueFrom:
            secretKeyRef:
              name: project-a-credentials
              key: git-token
        - name: SYNCHESTRA_LOG_LEVEL
          value: "info"
        
        ports:
        - containerPort: 50051
          name: grpc
          protocol: TCP
        
        volumeMounts:
        - name: workspace
          mountPath: /workspace/project-a
        - name: run
          mountPath: /var/run
        - name: tmp
          mountPath: /tmp
        
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "2000m"
        
        livenessProbe:
          exec:
            command:
            - synchestra-sandbox-agent
            - health
          initialDelaySeconds: 10
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 3
        
        readinessProbe:
          exec:
            command:
            - synchestra-sandbox-agent
            - health
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 2
      
      volumes:
      - name: workspace
        persistentVolumeClaim:
          claimName: synchestra-workspace-project-a
      - name: run
        emptyDir:
          sizeLimit: 100Mi
      - name: tmp
        emptyDir:
          sizeLimit: 200Mi

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: synchestra-workspace-project-a
  namespace: synchestra
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: fast
  resources:
    requests:
      storage: 50Gi
```

## Health Checks

### Health Check Command

```bash
# From host (check container health)
docker exec synchestra-sandbox-${PROJECT_ID} \
  synchestra-sandbox-agent health \
  --socket /var/run/synchestra-${PROJECT_ID}.sock \
  --timeout 5s

# Returns exit code 0 if healthy, 1+ if degraded/unhealthy
```

### Metrics & Observability

```bash
# Export metrics (Prometheus format)
curl -s http://localhost:9090/metrics | grep sandbox

# Example metrics:
# sandbox_container_uptime_seconds
# sandbox_active_sessions
# sandbox_commands_executed_total
# sandbox_credential_operations_total
# sandbox_disk_usage_bytes
# sandbox_memory_usage_bytes
```

## Debugging

### View Container Logs

```bash
# Stream logs in real-time
docker logs -f synchestra-sandbox-${PROJECT_ID}

# Last 100 lines
docker logs --tail 100 synchestra-sandbox-${PROJECT_ID}

# Timestamps enabled
docker logs -t synchestra-sandbox-${PROJECT_ID}
```

### Inspect Container

```bash
# Check running processes
docker top synchestra-sandbox-${PROJECT_ID}

# Check resource usage
docker stats synchestra-sandbox-${PROJECT_ID}

# Inspect config
docker inspect synchestra-sandbox-${PROJECT_ID}
```

### Debug Shell (if needed)

```bash
# Run debug container with same image
docker run -it --rm \
  --volumes-from synchestra-sandbox-${PROJECT_ID} \
  synchestra/sandbox-agent:latest \
  /bin/bash
```

## Troubleshooting

### Common Issues

**1. State Repository Clone Fails**
```
ERROR: Failed to clone state repository
URL: https://github.com/user/project-synchestra.git
```
- Verify URL is correct: `git ls-remote {URL}`
- Verify credentials: check `SYNCHESTRA_STATE_REPO_TOKEN`
- Check network connectivity from container

**2. Socket Permission Denied**
```
ERROR: Cannot listen on socket: /var/run/synchestra-project-a.sock
```
- Verify `/var/run` is writable: check volume mount permissions
- Check user permissions: container runs as UID 1000
- Verify socket path has no parent directory issues

**3. Memory Limit Exceeded**
```
Container killed (OOMKilled): exceeded memory limit
```
- Increase `--memory` limit
- Check if session resource limits are too high
- Review command execution output size (streaming helps)

**4. Encryption Key Issues**
```
ERROR: Encryption key validation failed
```
- If providing custom key: ensure it's 32 bytes (base64 encoded)
- If auto-generating: check `/dev/urandom` availability
- Verify key is consistent across container restarts (if persisting)

## Maintenance

### Updating Agent Binary

```bash
# Rebuild image with new code
git pull origin main
docker build -t synchestra/sandbox-agent:v1.1.0 -f build/Dockerfile.sandbox-agent .

# Push to registry
docker push myregistry.azurecr.io/synchestra/sandbox-agent:v1.1.0

# Restart containers with new image
docker stop synchestra-sandbox-*
docker run ... myregistry.azurecr.io/synchestra/sandbox-agent:v1.1.0
```

### Log Rotation

Configure docker daemon for log rotation:
```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

### Cleanup

```bash
# Remove stopped containers
docker container prune

# Remove dangling images
docker image prune

# Full cleanup (be careful!)
docker system prune --all
```

## Outstanding Questions

1. Should container images be automatically scanned and re-built on alpine/go updates?
2. ~~Should we support container image registry signing (Notary/DCT)?~~ **Resolved**: The requirement for signed Docker images is configurable per host.
3. What's the preferred container registry for production (Docker Hub, ECR, ACR)?
4. Should containers support GPU/accelerator access (NVIDIA CUDA)?
5. ~~Should we implement container metrics export to external monitoring system?~~ **Resolved**: Yes, eventually. Can plan for it but not first priority.
