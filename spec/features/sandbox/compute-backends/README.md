# Compute Backends

## Overview

Synchestra supports multiple compute backends for running sandbox containers. A project's configuration specifies which backend to use. The orchestrator delegates container lifecycle operations to the selected backend through a unified interface.

## Contents

| Document | Description |
|----------|-------------|
| [cloud-serverless.md](cloud-serverless.md) | Cloud serverless backend deep-dive: three submodes, workspace persistence, cold start optimization |

This document defines the pluggable compute backend architecture: three execution modes and the abstraction layer that unifies them. The current sandbox spec assumes a single Docker host; this document extends the architecture to support multiple deployment topologies while keeping the same Orchestrator interface for upstream consumers (HTTP API, CLI).

> **Related documents:** [orchestrator](../orchestrator/README.md) (orchestrator interface and state machine), [lifecycle](../orchestrator/lifecycle.md) (container lifecycle phases), [database-schema](../orchestrator/database-schema.md) (host-side storage), [http-api](../orchestrator/http-api.md) (API endpoints), [go-types-and-signatures](../go-types-and-signatures.md) (type definitions).

---

## Backend Selection

Projects specify their compute backend in the project configuration (in the spec repo or via API). The backend is selected at container provisioning time and persists for the lifetime of the container (migration between backends is not supported in v1).

```yaml
# In project's synchestra-spec.yaml or equivalent
sandbox:
  backend: single-host          # "single-host" | "cloud-serverless" | "kubernetes" | "external"
  backend_config:
    # Backend-specific configuration (see each backend section)
```

If no backend is specified, the host's default backend is used (typically `single-host`).

---

## Compute Backend Interface

All backends implement the same Go interface. This is the abstraction boundary — the orchestrator calls this interface and does not know which backend is active.

```go
// ComputeBackend abstracts container lifecycle operations across deployment topologies.
type ComputeBackend interface {
    // Provision creates a new container for the project. Returns container metadata.
    Provision(ctx context.Context, projectID string, config ContainerConfig) (*ContainerInfo, error)

    // Start starts a stopped container.
    Start(ctx context.Context, projectID string) error

    // Stop gracefully stops a running container (SIGTERM + timeout).
    Stop(ctx context.Context, projectID string, timeout time.Duration) error

    // Pause suspends a running container (if supported by backend).
    // Returns ErrNotSupported if the backend doesn't support pause.
    Pause(ctx context.Context, projectID string) error

    // Resume resumes a paused container.
    // Returns ErrNotSupported if the backend doesn't support resume.
    Resume(ctx context.Context, projectID string) error

    // Destroy removes a container and its resources.
    Destroy(ctx context.Context, projectID string) error

    // Status returns the current state of the container.
    Status(ctx context.Context, projectID string) (*ContainerStatus, error)

    // Connect returns a gRPC client connection to the container's agent.
    // For single-host: Unix socket. For cloud/K8s: TCP/TLS or HTTPS.
    Connect(ctx context.Context, projectID string) (*grpc.ClientConn, error)

    // SupportsCapability checks if the backend supports a given capability.
    SupportsCapability(cap Capability) bool
}
```

### Capabilities

```go
// Capability represents optional backend features.
type Capability string

const (
    CapPause            Capability = "pause"             // Pause/resume without stop
    CapLiveResize       Capability = "live_resize"       // Change resource limits while running
    CapWorkspacePersist Capability = "workspace_persist" // Workspace survives destroy
    CapAutoScale        Capability = "auto_scale"        // Backend handles scaling
)
```

### Supporting Types

```go
// ContainerConfig holds the configuration for provisioning a container.
type ContainerConfig struct {
    Image        string
    MemoryMB     int
    CPULimit     float64
    DiskGB       int
    EnvVars      map[string]string
    StateRepoURL string
    Labels       map[string]string
}

// ContainerInfo is returned after provisioning.
type ContainerInfo struct {
    ContainerID string // Backend-specific container/instance ID
    Endpoint    string // How to reach the agent: unix socket path, host:port, or URL
    Host        string // Which node the container is running on (for K8s mode; empty for single-host)
}

// ContainerStatus represents the current state.
type ContainerStatus struct {
    State       string // "running", "paused", "stopped", "failed", "creating"
    ContainerID string
    Host        string
    Endpoint    string
    StartedAt   time.Time
    Resources   ResourceUsage
}
```

---

## Mode 1: Single Host

The simplest mode. One VM runs `synchestra serve --http` and all containers are local Docker containers.

### Characteristics

| Property | Value |
|---|---|
| **Transport** | gRPC over Unix socket (`/var/run/synchestra/{project_id}.sock`) |
| **Storage** | Local Docker volumes at `SYNCHESTRA_SANDBOX_WORKSPACE_ROOT` |
| **State DB** | SQLite (local file) |
| **Scheduling** | N/A — all containers on same host |
| **Pause/Resume** | Docker `pause`/`unpause` (cgroup freeze) |
| **Use case** | Self-hosted, dev environments, small teams |

### Backend Config

```yaml
sandbox:
  backend: single-host
  backend_config:
    workspace_root: /var/synchestra/workspaces  # Optional override
    socket_dir: /var/run/synchestra             # Optional override
```

### Implementation

This is the current default implementation described in [orchestrator](../orchestrator/README.md) and [orchestrator implementation guide](../orchestrator/implementation-guide.md). The `ComputeBackend` implementation wraps the Docker client directly.

### Multi-tenant variant

A single host can serve multiple tenants (users/organizations). Isolation is provided by:

- Per-project containers with separate Docker namespaces
- User-scoped access control via the access cache (see [database-schema](../orchestrator/database-schema.md))
- Resource quotas per container

No architectural changes needed — multi-tenancy is an authorization concern, not a compute concern.

---

## Mode 2: Cloud Serverless

Per-project containers run as cloud-managed serverless instances (Google Cloud Run, AWS Fargate, Azure Container Instances). Synchestra orchestrates lifecycle via cloud provider APIs.

> **See [cloud-serverless.md](cloud-serverless.md) for the comprehensive architecture**, including three submodes (fully managed, delegated, external), workspace persistence, cold start optimization, and a cloud-vs-Kubernetes implementation comparison.

### Characteristics

| Property | Value |
|---|---|
| **Transport** | gRPC over HTTPS (cloud-assigned URL) |
| **Storage** | Cloud storage (GCS, S3, Azure Blob) mounted or synced into container |
| **State DB** | Cloud-managed (Cloud SQL, RDS, or Synchestra's shared PostgreSQL) |
| **Scheduling** | Cloud provider handles placement |
| **Pause/Resume** | **Not supported** — containers are started/stopped. Cold start latency is the trade-off. |
| **Use case** | Elastic scaling, pay-per-use, no infrastructure management |

### Backend Config

```yaml
sandbox:
  backend: cloud-run    # or "fargate", "aci"
  backend_config:
    provider: gcp       # "gcp" | "aws" | "azure"
    project: my-gcp-project
    region: us-central1
    service_account: synchestra@my-gcp-project.iam.gserviceaccount.com
    workspace_bucket: gs://synchestra-workspaces
```

### Cloud Run Example

- `Provision()` → `gcloud run services create synchestra-sandbox-{project_id} ...`
- `Start()` → scale to 1 instance (or update min-instances=1)
- `Stop()` → scale to 0 instances (or delete revision)
- `Pause()` → returns `ErrNotSupported` (Cloud Run does not support pause)
- `Connect()` → returns gRPC connection to `https://{service-url}:443`
- `Destroy()` → `gcloud run services delete ...`

### User-Provided Endpoint

As an alternative to Synchestra managing the cloud lifecycle, users can provide a pre-running endpoint:

```yaml
sandbox:
  backend: external
  backend_config:
    endpoint: https://my-sandbox.run.app:443
    # Synchestra skips Provision/Start/Stop — assumes endpoint is always ready
    # Only Connect() is used
```

This is useful for:

- Teams that manage their own infrastructure
- Environments where Synchestra does not have cloud provider credentials
- Custom runtimes that are not Docker-based

### Key Differences

- No pause/resume — `SupportsCapability(CapPause)` returns false
- Workspace must be synced to/from cloud storage on start/stop
- Higher cold-start latency (seconds vs milliseconds for Docker unpause)
- Networking is HTTPS, not Unix sockets or plain TCP
- Cost model changes: pay per execution time, not per VM

---

## Mode 3: Kubernetes

Containers run as Kubernetes Pods managed by a Synchestra controller (CRD + operator pattern).

### Characteristics

| Property | Value |
|---|---|
| **Transport** | gRPC over TCP/TLS (in-cluster service or pod IP) |
| **Storage** | PersistentVolumeClaim (PVC) per project |
| **State DB** | PostgreSQL (in-cluster or external), or CRD status fields |
| **Scheduling** | K8s scheduler with optional node affinity |
| **Pause/Resume** | Scale Deployment replicas 0↔1 (not true pause, but similar effect) |
| **Use case** | Teams already running K8s, enterprise deployments |

### Architecture

Synchestra runs as a K8s operator that watches `SandboxContainer` custom resources:

```yaml
apiVersion: synchestra.io/v1
kind: SandboxContainer
metadata:
  name: sandbox-my-project
  namespace: synchestra
spec:
  projectID: my-project
  image: synchestra/sandbox-agent:latest
  resources:
    memory: 512Mi
    cpu: "2"
    disk: 50Gi
  stateRepoURL: https://github.com/org/my-project-synchestra.git
status:
  phase: Running
  podName: sandbox-my-project-7f8d9
  nodeName: worker-3
  endpoint: 10.244.1.15:50051
```

### Backend Config

```yaml
sandbox:
  backend: kubernetes
  backend_config:
    namespace: synchestra       # K8s namespace for sandbox pods
    storage_class: fast-ssd     # StorageClass for PVCs
    node_selector:
      synchestra.io/role: sandbox
```

### Implementation

- `Provision()` → create `SandboxContainer` CR + PVC
- `Start()` → set `spec.replicas: 1` on the managed Deployment
- `Stop()` → set `spec.replicas: 0` (pod terminated, PVC retained)
- `Pause()` → same as Stop (K8s does not have true pause; returns `ErrNotSupported` or emulates via scale-to-zero)
- `Connect()` → connect to pod IP or K8s Service
- `Destroy()` → delete CR + PVC

### Key Differences

- Leverages K8s scheduler, health checks (liveness/readiness probes), and rolling updates
- PVCs provide durable workspace storage without shared filesystem
- Synchestra operator handles reconciliation (desired state → actual state)
- Networking uses K8s Services or direct pod IPs
- Monitoring integrates with existing K8s observability (Prometheus operator, etc.)

---

## Comparison Matrix

| Feature | Single Host | Cloud Serverless | Kubernetes |
|---|---|---|---|
| **Setup complexity** | Minimal | Medium | High (needs K8s) |
| **Scaling** | Vertical only | Elastic (auto) | Horizontal (auto) |
| **Pause/Resume** | ✅ Native | ❌ Start/Stop only | ⚠️ Scale 0/1 |
| **Cold start** | ~1s (unpause) | 2–10s (cold start) | 5–30s (pod scheduling) |
| **Workspace persistence** | Local volume | Cloud storage | PVC |
| **Cost model** | Fixed (VM) | Pay-per-use | Fixed (cluster) |
| **Transport** | Unix socket | HTTPS | TCP/TLS |
| **State DB** | SQLite | Cloud DB | PostgreSQL / CRD |
| **Multi-tenant** | ✅ (authz) | ✅ (isolation by service) | ✅ (namespace isolation) |
| **Infra management** | Self | Cloud provider | Self (K8s) |

---

## Impact on Existing Spec

Adopting the `ComputeBackend` interface requires the following changes to existing documents:

1. **[orchestrator](../orchestrator/README.md)**: The `DockerClient` dependency becomes the `ComputeBackend` interface. The state machine, health checks, and idle management remain the same — they call backend methods instead of Docker API directly.

2. **[database-schema](../orchestrator/database-schema.md)**: For Kubernetes mode, the `sandbox_container_metadata` table needs an `endpoint` column (replacing `socket_path`) to support TCP/HTTPS endpoints, and a `node` column for placement tracking. SQLite remains the default for single-host; shared PostgreSQL for cloud/K8s.

3. **[lifecycle](../orchestrator/lifecycle.md)**: Pause/resume phases become conditional on `SupportsCapability(CapPause)`. Backends that do not support pause skip directly to stop/start.

4. **[protocol](../agent/README.md)**: Transport-agnostic — gRPC works over Unix sockets, TCP/TLS, and HTTPS. No protocol changes needed.

5. **[credentials](../agent/credentials.md)**: No changes — credential encryption remains container-internal regardless of backend.

6. **[http-api](../orchestrator/http-api.md)**: Status responses gain a `node` field for K8s mode. A new admin endpoint `GET /api/v1/admin/sandbox/{project_id}/placement` reports backend and node info.

---

## Outstanding Questions

1. For cloud serverless mode, should Synchestra pre-warm containers (keep min-instances=1) or accept cold-start latency?
2. Should the `external` backend (user-provided endpoint) support any lifecycle operations beyond `Connect()`?
3. Should backend selection be immutable per-project, or should projects be able to migrate between backends?
4. For Kubernetes mode, should Synchestra use a CRD+operator pattern or a simpler Deployment-per-project approach?
5. Should cloud serverless workspace sync (to/from cloud storage) happen on every start/stop, or only on explicit save points?
