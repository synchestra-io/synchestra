## Contents

| Child | Description |
|---|---|
| [git](git/README.md) | TODO: Add description. |
| [sqlite](sqlite/README.md) | TODO: Add description. |

# State Store Backends

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/backends?op=request-change) |
**Parent:** [State Store](../)

Implementations of the `state.Store` interface. Each backend satisfies the full interface using its native storage and concurrency mechanisms.

| Backend | Directory | Use Case | Status |
|---|---|---|---|
| [Git](git/) | `pkg/state/gitstore/` | Default, works everywhere | Default implementation |
| SQLite | `pkg/state/sqlitestore/` | Single-host, high performance | Future |
| PostgreSQL | `pkg/state/pgstore/` | Multi-host, K8s clusters | Future |
| Cloud DB | TBD | Managed cloud deployments | Future |

### Git

The default backend. Maps every `state.Store` method to file operations, markdown rendering, and atomic commit-and-push in the [state repository](../../../architecture/repository-types.md#state-repository). See [Git Backend](git/).

## Open Questions

- Should backends be registered via a plugin mechanism, or is compile-time selection sufficient?
- How should backend-specific configuration (connection strings, credentials) be passed — via `StoreOptions` or backend-specific option types?

---
*This document follows the https://specscore.md/feature-specification*
