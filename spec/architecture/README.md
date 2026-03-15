# Architecture

Architectural documents describing Synchestra's foundational design decisions and structural concepts.

These are cross-cutting concerns that features and specifications build upon.

## Contents

| Document | Description |
|---|---|
| [repository-types.md](repository-types.md) | The three repository types (spec, state, code) — what they hold, why they're separate, how they connect |
| [spec-to-execution.md](spec-to-execution.md) | How features, development plans, and tasks connect across repository boundaries |
| [sandbox-architecture.md](sandbox-architecture.md) | Sandbox feature design: stateless host, autonomous containers per project, gRPC communication, state management |
| [sandbox-security.md](sandbox-security.md) | Security model for sandbox: credential encryption, container hardening, user isolation, threat analysis |

### repository-types.md

Defines the three kinds of repositories Synchestra operates with: the spec repository (requirements and documentation), the state repository (tasks and coordination), and the code repository (implementation). Explains why the state repo must be separate, naming conventions, how repos connect through `synchestra-spec.yaml`, and when spec+code repos can be combined.

### spec-to-execution.md

The end-to-end pipeline from product intent to running work. Shows how features (what), development plans (how), and tasks (who/when) relate across the spec and state repositories. Covers the three-layer architecture, the complete lifecycle sequence, artifact relationships, mutability profiles, derived status without duplication, and repository boundaries.

### sandbox-architecture.md

Comprehensive design specification for the Sandbox feature. Covers: stateless host architecture, autonomous containers per project, gRPC communication protocol, container file structure, concurrency and isolation model, container lifecycle management (startup, running, paused, resume, cleanup), state management (container-authoritative, no host sync), HTTP API surface, deployment infrastructure, and design rationale.

### sandbox-security.md

Detailed security model and threat analysis for Sandbox. Covers: threat model with seven attack vectors (host compromise, container escape, credential theft, cross-user leakage, DoS, state tampering, network interception), mitigations for each threat, security mechanisms (AES256 encryption, container hardening, user isolation, resource limits, gRPC security), validation testing checklist, and future enhancements (HSM, signed commits, audit trail, credential rotation, per-user encryption).

## Outstanding Questions

None at this time.
