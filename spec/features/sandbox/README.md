# Sandbox Feature

Safe, isolated execution environments for running user-initiated commands from the chat interface. Each Synchestra project gets its own persistent Docker container with encrypted credential storage and user-isolated sessions.

## Contents

| Directory / Document | Description |
|---|---|
| [orchestrator/](orchestrator/README.md) | Host-side Container Orchestrator: lifecycle state machine, gRPC pool, health monitoring, routing, database schema, HTTP API |
| [agent/](agent/README.md) | Container-side gRPC agent: protocol, protobuf definitions, credential vault, implementation patterns |
| [container-image/](container-image/README.md) | Docker image build, security hardening, entrypoint, deployment procedures, CI/CD automation |
| [compute-backends/](compute-backends/README.md) | Pluggable compute backends: Single Host, Cloud Serverless, Kubernetes |
| [observability/](observability/README.md) | Monitoring, logging, alerting, distributed tracing, and testing strategy |
| [go-types-and-signatures.md](go-types-and-signatures.md) | Go type/interface definitions, function signatures, and call graphs across all packages |
| [outstanding-questions.md](outstanding-questions.md) | Consolidated outstanding questions with context and recommendations |

## Subsystem Summaries

### [orchestrator/](orchestrator/README.md)

Container Orchestrator specification: the host-side service component that manages sandbox container lifecycles through a 10-state state machine (17 transitions), maintains a gRPC connection pool, performs health monitoring with circuit breakers, handles idle detection and auto-pause/resume, enforces resource quotas and LRU eviction, and routes HTTP API requests to the appropriate container. Includes the host-side SQLite database schema (`sandbox_container_metadata`, `sandbox_user_project_access`), the full HTTP REST API specification (sandbox and admin endpoints, auth, error mapping, rate limiting), container lifecycle phases with operational runbook, and Go implementation patterns.

### [agent/](agent/README.md)

Container-side gRPC agent (`SandboxAgent` service) that runs inside each sandbox container. Defines the complete gRPC protocol over Unix sockets — command execution with streaming output, session management, credential storage and retrieval via AES256-GCM encrypted vault, task state queries against the local `.synchestra/` git repo, and health checks. Includes the protobuf 3 service definition (`agent.proto`), credential management specification (encryption architecture, vault format, injection patterns, key rotation, audit logging), and Go implementation patterns.

### [container-image/](container-image/README.md)

Multi-stage Docker image specification for the sandbox agent. Covers the Dockerfile (Alpine-based, non-root user, read-only filesystem, dropped capabilities), entrypoint script (environment validation, workspace setup, state repo clone, encryption key management, agent startup), image building and scanning (Trivy, Docker Content Trust), deployment procedures (Docker, Compose, Kubernetes), and CI/CD automation (Makefile targets, GitHub Actions workflow).

### [compute-backends/](compute-backends/README.md)

Pluggable compute backend architecture defining the `ComputeBackend` Go interface and three execution modes: Single Host (local Docker, SQLite, Unix sockets — the default), Cloud Serverless (Cloud Run/Fargate/ACI with three submodes: fully managed, delegated, external), and Kubernetes (CRD+operator, PVCs, K8s scheduler). Includes workspace persistence strategies, cold start optimization, and a comparison matrix across all modes.

### [observability/](observability/README.md)

Observability strategy spanning both host-side (orchestrator, HTTP API) and container-side (gRPC agent). Covers Prometheus metrics catalog, structured JSON logging with sensitive data policy, OpenTelemetry distributed tracing, alerting rules (critical/warning/info), dashboard specifications, and health endpoints. Also includes the integration testing strategy: unit tests, integration tests (Docker lifecycle, gRPC communication, session reconnection), end-to-end tests, test infrastructure, and security test cases.

## Outstanding Questions

> **See [outstanding-questions.md](outstanding-questions.md) for the full consolidated list with context and recommendations.**

1. Should credentials support expiry/auto-rotation? Timeline for this feature?
2. ~~Should audit logs be retained indefinitely or with retention policy?~~ **Resolved**: Retain logs only for the last N hours; default is 24 hours. The retention window is configurable per host.
3. Are there compliance requirements (HIPAA, PCI-DSS, SOC 2) affecting credential handling?
4. ~~Should container images be signed (Docker Content Trust)?~~ **Resolved**: The requirement for signed Docker images is configurable per host.
5. ~~Should containers auto-terminate after idle period (e.g., 24 hours)?~~ **Resolved**: Yes, containers auto-terminate after an idle timeout that is configurable per host.
