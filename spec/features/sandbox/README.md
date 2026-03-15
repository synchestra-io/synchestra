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

## Outstanding Questions

1. Should credentials support expiry/auto-rotation? Timeline for this feature?
2. Should audit logs be retained indefinitely or with retention policy?
3. Are there compliance requirements (HIPAA, PCI-DSS, SOC 2) affecting credential handling?
4. Should container images be signed (Docker Content Trust)?
5. Should containers auto-terminate after idle period (e.g., 24 hours)?
