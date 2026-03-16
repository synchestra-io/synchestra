# Cloud Serverless Backend

Comprehensive architecture for running Synchestra sandbox containers on cloud serverless platforms (Google Cloud Run, AWS Fargate, Azure Container Instances).

> **Related documents:** [compute-backends.md](compute-backends.md) (backend selection and `ComputeBackend` interface), [orchestrator.md](orchestrator.md) (orchestrator state machine), [lifecycle.md](lifecycle.md) (container lifecycle phases), [credentials.md](credentials.md) (credential encryption), [http-api.md](http-api.md) (REST API), [outstanding-questions.md](outstanding-questions.md) (open design questions).

---

## Submodes

The cloud serverless backend supports three submodes based on who owns the cloud resources and who pays:

| Submode | Lifecycle | Cloud Account | Billing | Use Case |
|---|---|---|---|---|
| **A: Fully Managed** | Synchestra | Synchestra's account | User pays Synchestra | SaaS product, simplest UX |
| **B: User Account, Managed Lifecycle** | Synchestra | User's cloud account | Cloud provider bills user | Enterprise, cost control, data sovereignty |
| **C: User-Managed** | User | User's cloud account | Cloud provider bills user | Full control, BYOI (bring your own infra) |

```yaml
# Project configuration
sandbox:
  backend: cloud-serverless
  submode: managed           # "managed" | "delegated" | "external"
  # submode-specific config follows
```

---

## Submode A: Fully Managed

Synchestra owns and operates the cloud infrastructure. Users interact only with the Synchestra API/WebUI. Cloud costs are absorbed into the Synchestra subscription price.

### How It Works

```
User → Synchestra API → Synchestra Orchestrator → Synchestra's Cloud Account
                                                    ├── Cloud Run service (per project)
                                                    ├── Cloud Storage bucket (workspaces)
                                                    └── Cloud SQL (metadata)
```

1. User creates a project and triggers their first sandbox command.
2. Synchestra provisions a Cloud Run service in **Synchestra's own GCP/AWS/Azure account**.
3. All cloud resources are tagged with the user/org ID for internal cost attribution.
4. User never sees cloud credentials, project IDs, or resource names.

### Configuration

```yaml
sandbox:
  backend: cloud-serverless
  submode: managed
  # No backend_config needed — Synchestra manages everything.
  # Optional overrides:
  region: us-central1          # Preferred region (default: closest to user)
  tier: standard               # "standard" | "performance" (affects CPU/memory limits)
```

### Architecture

**Synchestra-side infrastructure:**

- **Multi-tenant Cloud Run deployment**: Each project gets its own Cloud Run service in a shared GCP project (or AWS account). Services are isolated by IAM and network policies.
- **Shared Cloud Storage**: One bucket per region with per-project prefixes: `gs://synchestra-workspaces-{region}/{org_id}/{project_id}/`
- **Shared metadata DB**: Cloud SQL (PostgreSQL) or equivalent, replacing SQLite.
- **Cost attribution**: Each Cloud Run service is labeled with `synchestra-org={org_id}`, `synchestra-project={project_id}`. Monthly cost reports are generated per org for internal billing.

**Isolation model:**

| Layer | Mechanism |
|---|---|
| Compute | Separate Cloud Run service per project (process-level isolation by cloud provider) |
| Storage | Per-project prefix in shared bucket; IAM policy per service account |
| Network | Cloud Run services have no inter-service network access by default |
| Secrets | Credential vault inside container (unchanged from single-host mode) |
| Data | No cross-project data access; each service account scoped to its prefix |

**Billing model:**

- Synchestra charges a flat subscription or per-minute execution fee.
- Synchestra absorbs cloud costs and manages margins.
- Users see a single invoice from Synchestra, not from GCP/AWS/Azure.
- Internal cost tracking via cloud resource labels enables per-org cost dashboards.

### Pros & Cons

| Pros | Cons |
|---|---|
| Simplest UX — no cloud account needed | Synchestra takes on cloud cost risk |
| Full control over infrastructure quality | Higher price to cover margins |
| Consistent experience across users | Data resides in Synchestra's account (compliance concern for some orgs) |
| Synchestra can optimize resource usage | Multi-tenant security responsibility falls on Synchestra |

---

## Submode B: User Account, Managed Lifecycle (Delegated)

Synchestra manages the container lifecycle but deploys into the **user's own cloud account**. Cloud usage is billed directly to the user by the cloud provider. Synchestra needs delegated access (service account, IAM role) to the user's account.

### How It Works

```
User → Synchestra API → Synchestra Orchestrator → User's Cloud Account (via delegated credentials)
                                                    ├── Cloud Run service (per project)
                                                    ├── Cloud Storage bucket (user-owned)
                                                    └── User's billing account
```

1. User provides cloud credentials (GCP service account key, AWS IAM role ARN, Azure service principal) to Synchestra via a one-time setup flow.
2. Synchestra stores these credentials encrypted (using the same vault pattern as sandbox credentials).
3. On sandbox creation, Synchestra uses the delegated credentials to provision Cloud Run services in the user's account.
4. Cloud provider bills the user directly for compute, storage, and network.
5. Synchestra bills the user separately for the platform (orchestration, WebUI, API).

### Configuration

```yaml
sandbox:
  backend: cloud-serverless
  submode: delegated
  backend_config:
    provider: gcp                # "gcp" | "aws" | "azure"
    
    # GCP-specific
    gcp_project: user-gcp-project-id
    gcp_region: us-central1
    gcp_service_account_key: vault://cloud-credentials/gcp-key
    # Or: gcp_workload_identity_provider for keyless auth
    workspace_bucket: gs://user-synchestra-workspaces
    
    # AWS-specific (alternative)
    # aws_account_id: "123456789012"
    # aws_region: us-east-1
    # aws_role_arn: arn:aws:iam::123456789012:role/synchestra-delegated
    # workspace_bucket: s3://user-synchestra-workspaces
```

### Setup Flow

1. **User creates a cloud service account** with the minimum required permissions (Synchestra provides a Terraform module / gcloud script / CloudFormation template).
2. **User registers the credentials** in Synchestra via the WebUI or CLI:
   ```bash
   synchestra config set-cloud-credentials --provider gcp --key-file ~/sa-key.json
   ```
3. **Synchestra validates** the credentials by listing Cloud Run services (read-only probe).
4. **Credentials are encrypted** and stored in Synchestra's vault (never logged, never displayed after initial setup).

### Required Cloud Permissions

**GCP (Cloud Run):**
```
roles/run.admin              # Create, update, delete Cloud Run services
roles/storage.objectAdmin    # Read/write workspace bucket
roles/iam.serviceAccountUser # Attach service account to Cloud Run services
```

**AWS (Fargate):**
```
ecs:CreateService, ecs:UpdateService, ecs:DeleteService
ecs:RunTask, ecs:StopTask, ecs:DescribeTasks
s3:GetObject, s3:PutObject, s3:DeleteObject  (workspace bucket)
iam:PassRole  (for task execution role)
```

### Isolation Model

Stronger than Submode A because each org uses its own cloud account:

| Layer | Mechanism |
|---|---|
| Compute | Separate Cloud Run service per project, in user's own cloud project |
| Storage | User-owned bucket with per-project prefixes |
| Network | User controls VPC, firewall rules, and egress policies |
| Billing | Direct from cloud provider to user — full cost transparency |
| Compliance | Data stays in user's account; user controls data residency |

### Pros & Cons

| Pros | Cons |
|---|---|
| User controls data residency and compliance | Setup friction (one-time credential delegation) |
| Direct cloud billing — cost transparency | Synchestra needs secure credential storage |
| Stronger isolation (user's own account) | Debugging harder (Synchestra can't access user's logs without explicit grants) |
| User can audit all cloud resources | User must maintain cloud account and permissions |

---

## Submode C: User-Managed (External)

The user manages the entire cloud infrastructure. Synchestra provides only the WebUI, CLI, and API — it connects to a user-provided endpoint but does not manage lifecycle.

### How It Works

```
User → Synchestra API → Synchestra Orchestrator → User's Pre-Running Endpoint
                         (Connect() only)           ├── User manages Cloud Run / Fargate / VM / anything
                                                     ├── User manages storage
                                                     └── User manages billing
```

1. User deploys the `synchestra/sandbox-agent` container image (or a compatible custom image) on their own infrastructure.
2. User provides the endpoint URL to Synchestra.
3. Synchestra calls `Connect()` to establish a gRPC connection — all other lifecycle operations are no-ops or return the user-reported status.
4. User is responsible for keeping the endpoint running, scaling, and paying for it.

### Configuration

```yaml
sandbox:
  backend: cloud-serverless
  submode: external
  backend_config:
    endpoint: https://my-sandbox.run.app:443
    # Optional: health check URL (Synchestra will poll to detect outages)
    health_url: https://my-sandbox.run.app/healthz
    # Optional: auth token for mTLS or bearer auth
    auth_token: vault://cloud-credentials/endpoint-token
```

### ComputeBackend Behavior

| Method | Behavior |
|---|---|
| `Provision()` | No-op (returns info from config) |
| `Start()` | No-op (assumes always running) |
| `Stop()` | No-op (user manages lifecycle) |
| `Pause()` | Returns `ErrNotSupported` |
| `Resume()` | Returns `ErrNotSupported` |
| `Destroy()` | No-op (user manages cleanup) |
| `Status()` | Calls health endpoint; returns `running` if reachable, `failed` if not |
| `Connect()` | Establishes gRPC connection to configured endpoint |

### Container Image Compatibility

The user's endpoint must run a gRPC server compatible with [agent.proto](agent.proto). Options:

1. **Use the official `synchestra/sandbox-agent` image** — deploy it on any platform that runs Docker containers.
2. **Build a custom agent** — implement the `SandboxAgent` gRPC service (see [protocol.md](protocol.md)). Must support at least `ExecuteCommand`, `GetStatus`, `Ping`.
3. **Use a sidecar pattern** — run the official agent as a sidecar alongside custom tooling.

### Pros & Cons

| Pros | Cons |
|---|---|
| Maximum flexibility — any platform, any runtime | User handles all ops (scaling, monitoring, updates) |
| No cloud credentials shared with Synchestra | Synchestra can't auto-heal or auto-restart on failure |
| Works with existing infrastructure | Cold starts and availability are user's problem |
| No Synchestra lock-in for compute | No workspace persistence managed by Synchestra |

---

## Shared Architecture (All Submodes)

### Transport: gRPC over HTTPS

All cloud serverless submodes use gRPC over HTTPS (port 443) with TLS. This differs from single-host mode (Unix sockets).

```go
// Cloud serverless Connect() implementation
func (b *CloudBackend) Connect(ctx context.Context, projectID string) (*grpc.ClientConn, error) {
    endpoint := b.getEndpoint(projectID) // e.g., "my-service-abc123.run.app:443"
    
    creds := credentials.NewTLS(&tls.Config{})
    conn, err := grpc.DialContext(ctx, endpoint,
        grpc.WithTransportCredentials(creds),
        grpc.WithPerRPCCredentials(b.authProvider), // Bearer token or mTLS
    )
    return conn, err
}
```

### Authentication Between Orchestrator and Agent

The orchestrator must authenticate to the cloud-hosted agent. Unlike single-host mode (where Unix socket permissions provide auth), cloud mode requires explicit authentication:

| Method | How | When |
|---|---|---|
| **Cloud IAM** | Cloud Run invoker role / Fargate task role | Submode A & B (cloud-native auth, no tokens to manage) |
| **Bearer token** | Token injected via env var, validated by agent middleware | Submode C (external endpoints) |
| **mTLS** | Client certificate issued by Synchestra CA | High-security deployments |

### Workspace Persistence

Cloud serverless containers are ephemeral — local filesystem is lost on stop. Workspaces must be synced to/from cloud storage.

**Sync strategy:**

```
Container Start:
  1. Download workspace snapshot from cloud storage → /workspace/{project_id}/
  2. Clone/pull state repo
  3. Container ready

Container Stop (graceful):
  1. Agent receives SIGTERM
  2. Upload workspace snapshot to cloud storage
  3. Exit

Periodic sync (while running):
  1. Every N minutes, upload changed files to cloud storage (incremental)
  2. Prevents data loss on crash
```

**Storage layout:**

```
gs://synchestra-workspaces-{region}/{org_id}/{project_id}/
  ├── workspace.tar.zst           # Compressed workspace snapshot
  ├── credentials.enc             # Encrypted credential vault
  └── metadata.json               # Last sync time, container version
```

### Cold Start Optimization

Cloud serverless containers have cold start latency (2-10s). Strategies to minimize:

1. **Min instances = 1**: Keep one instance warm (costs money, eliminates cold start).
   - Submode A: Synchestra decides based on project activity.
   - Submode B: User configures via `backend_config.min_instances`.
   - Submode C: User's responsibility.

2. **Workspace pre-warming**: Start workspace download before the first RPC arrives (triggered by HTTP health check).

3. **Slim container image**: Keep the sandbox agent image small (< 200MB). Use multi-stage builds (already defined in [Dockerfile.spec](Dockerfile.spec)).

4. **Lazy workspace loading**: Start accepting commands immediately; download workspace in the background. Commands that need workspace files wait for sync to complete.

### Lifecycle Mapping

How the 10-state lifecycle (from [orchestrator.md](orchestrator.md)) maps to cloud serverless:

| Orchestrator State | Cloud Serverless Equivalent |
|---|---|
| `unprovisioned` | No Cloud Run service exists |
| `creating` | Cloud Run service being deployed |
| `starting` | Instance starting (cold start) |
| `running` | Instance serving requests |
| `paused` | **Not supported** — skip to `stopped` |
| `resuming` | **Not supported** — use `starting` |
| `stopping` | Instance draining (workspace sync in progress) |
| `stopped` | Service scaled to 0 (or min-instances=0, no active instances) |
| `failed` | Instance crash loop or health check failures |
| `terminated` | Cloud Run service deleted |

The orchestrator's state machine remains the same, but transitions through `paused`/`resuming` are skipped (the `SupportsCapability(CapPause)` check handles this).

### Monitoring

Cloud providers offer built-in monitoring that complements Synchestra's own:

| Metric Source | What It Provides |
|---|---|
| **Cloud provider** (Cloud Run metrics, CloudWatch) | Request count, latency, instance count, CPU/memory, cold starts |
| **Synchestra agent** (inside container) | Command execution count, session metrics, credential operations |
| **Synchestra orchestrator** (host side) | End-to-end latency, error rates, backend health |

For Submode A, Synchestra aggregates all three. For Submode B, the user sees cloud metrics in their own console; Synchestra shows agent + orchestrator metrics. For Submode C, Synchestra shows only orchestrator-side metrics.

---

## Cloud vs Kubernetes: Implementation Comparison

Which backend is easier and cheaper to implement first?

### Implementation Effort

| Aspect | Cloud Serverless | Kubernetes |
|---|---|---|
| **Cloud provider SDKs** | 3 SDKs (GCP, AWS, Azure) or start with 1 | 1 SDK (`client-go`) |
| **Core implementation** | REST API calls to cloud provider | CRD definition + controller/reconciler loop |
| **State management** | Cloud provider manages container state | Synchestra must implement K8s operator pattern |
| **Networking** | Cloud-assigned URLs (simple) | Service/Ingress configuration (complex) |
| **Storage** | Cloud storage SDKs (well-documented) | PVC provisioning + StorageClass setup |
| **Auth** | Cloud IAM (built-in) | ServiceAccount + RBAC configuration |
| **Health checks** | Cloud provider built-in | Must configure liveness/readiness probes |
| **Scaling** | Built-in (scale to zero) | Must configure HPA or manual replica management |
| **Testing** | Can mock cloud APIs; emulators available | Needs a real K8s cluster or kind/minikube for integration tests |
| **Time to MVP** | **2-3 weeks** (single provider) | **4-6 weeks** (operator + CRD + RBAC) |

### Cost to Run

| Aspect | Cloud Serverless | Kubernetes |
|---|---|---|
| **Idle cost** | $0 (scale to zero) | $50-200/mo minimum (cluster nodes) |
| **Per-execution cost** | ~$0.00004/vCPU-second | Amortized across cluster |
| **Storage** | ~$0.02/GB/month (cloud storage) | ~$0.10/GB/month (PVC/SSD) |
| **Fixed overhead** | None | K8s control plane ($72/mo on GKE, free on EKS) |
| **Break-even** | Cheaper below ~500 concurrent containers | Cheaper above ~500 concurrent containers |

### Recommendation

**Implement Cloud Serverless first (Submode A with GCP Cloud Run).**

Reasoning:
1. **Lower fixed cost**: No cluster to maintain. Scale-to-zero means you pay nothing for idle projects.
2. **Faster to implement**: Cloud Run has a simple API — `create service`, `update service`, `delete service`. No operator pattern needed.
3. **Better for early-stage product**: Most initial users will have < 50 projects. K8s overhead isn't justified until scale demands it.
4. **Submode A first**: Start with Synchestra-managed (simplest UX, no credential delegation). Add Submode B when enterprise customers need it. Submode C is nearly free (it's just `Connect()` to a user URL).
5. **GCP first**: Cloud Run has the best gRPC support, simplest API, and generous free tier (2M requests/month). Add AWS Fargate as a second provider when there's demand.

K8s backend should come later, when:
- Customers already have K8s clusters and want to use them
- Scale exceeds what's cost-effective on serverless
- Self-hosted on-prem deployments are needed (no cloud provider)

---

## Implementation Roadmap

### Phase 1: Submode C (External Endpoint)
**Effort: 1-2 days.** Nearly trivial — `Connect()` to a user-provided URL. All lifecycle operations are no-ops. This is useful immediately for power users and testing.

### Phase 2: Submode A (Fully Managed, GCP Cloud Run)
**Effort: 2-3 weeks.**
- Implement GCP Cloud Run backend (create/start/stop/destroy/connect)
- Workspace sync to/from GCS
- Cloud IAM authentication for orchestrator→agent
- Cost attribution labels
- Integration tests with Cloud Run emulator

### Phase 3: Submode B (Delegated, GCP)
**Effort: 1-2 weeks** (incremental on Phase 2).
- Credential delegation setup flow (WebUI + CLI)
- Secure storage of user's GCP service account key
- Permission validation probe
- Per-user Cloud Run service deployment

### Phase 4: AWS Fargate Support
**Effort: 2-3 weeks.**
- Fargate backend implementation (parallel to Cloud Run, same interface)
- S3 workspace sync
- IAM role delegation for Submode B
- Cross-provider abstraction layer (if patterns diverge)

### Phase 5: Kubernetes Backend (separate from this doc)
**Effort: 4-6 weeks.** See [compute-backends.md](compute-backends.md) Mode 3.

---

## Outstanding Questions

1. For Submode A, what pricing model works best: per-minute execution, flat monthly fee, or tiered?
2. For Submode B, should Synchestra support GCP Workload Identity Federation (keyless auth) as an alternative to service account keys?
3. Should workspace sync be full snapshots or incremental (rsync-style diffs)?
4. What is the maximum acceptable cold start latency before auto-warming is triggered?
5. For Submode C, should Synchestra verify agent compatibility (protocol version check) on first connection?
6. Should there be a Submode B setup wizard in the WebUI that generates the Terraform/gcloud commands?
