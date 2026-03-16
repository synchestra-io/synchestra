# Sandbox Feature

Safe, isolated execution environments for running user-initiated commands from the chat interface. Each Synchestra project gets its own persistent Docker container with encrypted credential storage and user-isolated sessions.

## Contents

| Document | Description |
|---|---|
| [database-schema.md](database-schema.md) | Host database schema: minimal, access mappings only |
| [protocol.md](protocol.md) | gRPC service protocol and message definitions |
| [agent.proto](agent.proto) | Protobuf 3 service definition |
| [agent-implementation-guide.md](agent-implementation-guide.md) | Detailed Go implementation patterns for gRPC agent |
| [Dockerfile.md](Dockerfile.md) | Container image overview, build arguments, runtime configuration |
| [Dockerfile.spec](Dockerfile.spec) | Complete multi-stage Dockerfile specification |
| [docker-entrypoint.sh](docker-entrypoint.sh) | Container entrypoint script for initialization |
| [container-build-deployment.md](container-build-deployment.md) | Image building, scanning, deployment (Docker, Compose, K8s) |
| [container-build-automation.md](container-build-automation.md) | Build scripts, Makefile targets, CI/CD pipeline (GitHub Actions) |
| [orchestrator.md](orchestrator.md) | Container Orchestrator: lifecycle state machine, gRPC pool, health monitoring, routing |
| [orchestrator-implementation-guide.md](orchestrator-implementation-guide.md) | Go implementation patterns for Container Orchestrator |
| [credentials.md](credentials.md) | Credential encryption, vault format, injection patterns, rotation, audit |
| [lifecycle.md](lifecycle.md) | Container lifecycle phases, workspace cache, timing parameters, runbook |
| [http-api.md](http-api.md) | HTTP REST API specification: sandbox and admin endpoints, auth, error mapping, rate limiting |
| [testing.md](testing.md) | Integration testing strategy: unit, integration, and end-to-end test specifications |
| [monitoring.md](monitoring.md) | Monitoring, logging, alerting: Prometheus metrics, structured logging, tracing, dashboards |
| [outstanding-questions.md](outstanding-questions.md) | Consolidated outstanding questions with context and recommendations |
| [go-types-and-signatures.md](go-types-and-signatures.md) | Go type/interface definitions, function signatures, and call graphs |
| [compute-backends.md](compute-backends.md) | Pluggable compute backends: Single Host, Cloud Serverless, Kubernetes |
| [cloud-serverless.md](cloud-serverless.md) | Cloud Serverless deep-dive: 3 submodes (managed, delegated, external), workspace sync, cold starts, cloud-vs-K8s comparison |

## Document Summaries

### [database-schema.md](database-schema.md)
Host-side database schema for container metadata and access control.

### [protocol.md](protocol.md)
Complete gRPC service specification for host↔container communication.

### [agent.proto](agent.proto)
Protobuf 3 service definition for code generation.

### [agent-implementation-guide.md](agent-implementation-guide.md)
Go implementation patterns for the gRPC agent.

### [Dockerfile.spec](Dockerfile.spec)
Multi-stage Dockerfile for sandbox container image.

### [Dockerfile.md](Dockerfile.md)
Container build arguments and runtime configuration documentation.

### [docker-entrypoint.sh](docker-entrypoint.sh)
Container initialization and setup script.

### [container-build-deployment.md](container-build-deployment.md)
Build, scan, and deployment procedures.

### [container-build-automation.md](container-build-automation.md)
Makefile targets, build scripts, and CI/CD pipeline.

### [orchestrator.md](orchestrator.md)
Container Orchestrator specification: lifecycle state machine, gRPC connection pool, health monitoring, idle detection, circuit breaker, request routing, and resource quota enforcement.

### [orchestrator-implementation-guide.md](orchestrator-implementation-guide.md)
Go implementation patterns for the Container Orchestrator — interfaces, state machine, connection pool, health manager, circuit breaker, and graceful shutdown.

### [credentials.md](credentials.md)
Credential management specification: AES256-GCM encryption architecture, vault format, credential injection patterns (git tokens, SSH keys, env vars), key rotation, and audit logging. Authoritative reference for all credential-related behavior.

### [lifecycle.md](lifecycle.md)
Container lifecycle specification: 8 lifecycle phases (provision through terminate), workspace cache with GitHub Actions-style persistence, resource management by state, timing parameters, and operational runbook for common scenarios.

### [http-api.md](http-api.md)
HTTP REST API specification served by `synchestra serve --http`. Defines all sandbox endpoints (execute, status, sessions, WebSocket logs, credentials, destroy) and admin endpoints (stop, restart, evict, config, image, container listing). Covers authentication, authorization matrix, gRPC-to-HTTP error mapping, rate limiting, and Go package structure.

### [testing.md](testing.md)
Integration testing strategy covering three tiers: unit tests (orchestrator state machine, credential vault, HTTP handlers), integration tests (Docker lifecycle, gRPC communication, session reconnection), and end-to-end tests (full HTTP→orchestrator→gRPC→container flows). Includes test infrastructure (mock Docker client, test container image), CI/CD integration, and security test cases.

### [monitoring.md](monitoring.md)
Monitoring, logging, and alerting specification: Prometheus metrics catalog (host-side and container-side), structured JSON logging with sensitive data policy, OpenTelemetry distributed tracing, alerting rules (critical/warning/info), dashboard specifications (overview, per-project, operations), and health endpoints.

### [outstanding-questions.md](outstanding-questions.md)
Consolidated summary of all unresolved design questions from across the sandbox spec. Grouped by theme (credential management, container lifecycle, API & protocol, observability, database, testing, security) with brief context and 1-2 recommended resolutions per question.

### [go-types-and-signatures.md](go-types-and-signatures.md)
All Go type definitions, interface definitions, and function signatures for the sandbox feature. Organized by package (orchestrator, agent, API, observability) with detailed call graphs for the main execution flows (command execution, credential management, health checking, idle detection, auto-resume, graceful shutdown).

### [compute-backends.md](compute-backends.md)
Pluggable compute backend architecture for running sandbox containers across different deployment topologies. Defines the `ComputeBackend` Go interface and three execution modes: Single Host (local Docker, SQLite, Unix sockets — the default), Cloud Serverless (Cloud Run/Fargate/ACI, HTTPS, pay-per-use), and Kubernetes (CRD+operator, PVCs, K8s scheduler). Includes a user-provided external endpoint variant and a comparison matrix across all modes.

### [cloud-serverless.md](cloud-serverless.md)
Comprehensive architecture for the Cloud Serverless backend. Defines three submodes: Fully Managed (Synchestra owns cloud infra, user pays Synchestra), Delegated (Synchestra manages lifecycle in user's cloud account, cloud bills user directly), and External (user provides a pre-running endpoint, Synchestra connects only). Covers gRPC-over-HTTPS transport, workspace persistence to cloud storage, cold start optimization, orchestrator-to-agent authentication, and a detailed cloud-vs-Kubernetes implementation cost/effort comparison with a phased roadmap.

## Outstanding Questions

> **See [outstanding-questions.md](outstanding-questions.md) for the full consolidated list with context and recommendations.**

1. Should credentials support expiry/auto-rotation? Timeline for this feature?
2. ~~Should audit logs be retained indefinitely or with retention policy?~~ **Resolved**: Retain logs only for the last N hours; default is 24 hours. The retention window is configurable per host.
3. Are there compliance requirements (HIPAA, PCI-DSS, SOC 2) affecting credential handling?
4. ~~Should container images be signed (Docker Content Trust)?~~ **Resolved**: The requirement for signed Docker images is configurable per host.
5. ~~Should containers auto-terminate after idle period (e.g., 24 hours)?~~ **Resolved**: Yes, containers auto-terminate after an idle timeout that is configurable per host.
