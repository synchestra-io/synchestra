# State Store Backends

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

## Outstanding Questions

- Should backends be registered via a plugin mechanism, or is compile-time selection sufficient?
- How should backend-specific configuration (connection strings, credentials) be passed — via `StoreOptions` or backend-specific option types?
