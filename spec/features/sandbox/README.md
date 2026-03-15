# Sandbox Feature

Safe, isolated execution environments for running user-initiated commands from the chat interface. Each Synchestra project gets its own persistent Docker container with encrypted credential storage and user-isolated sessions.

## Contents

| Document | Description |
|---|---|
| [overview.md](overview.md) | Feature summary and user-facing benefits |
| [database-schema.md](database-schema.md) | Host database schema: minimal, access mappings only |
| [protocol.md](protocol.md) | gRPC service protocol and message definitions |
| [container-operations.md](container-operations.md) | Container lifecycle, file structure, initialization |
| [credential-management.md](credential-management.md) | AES256 encryption, storage, decryption, lifecycle |
| [user-isolation.md](user-isolation.md) | Session isolation, filesystem permissions, resource limits |

## Outstanding Questions

1. Should credentials support expiry/auto-rotation? Timeline for this feature?
2. Should audit logs be retained indefinitely or with retention policy?
3. Are there compliance requirements (HIPAA, PCI-DSS, SOC 2) affecting credential handling?
4. Should container images be signed (Docker Content Trust)?
5. Should containers auto-terminate after idle period (e.g., 24 hours)?
