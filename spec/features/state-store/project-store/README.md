# Sub-Feature: Project Store

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/project-store?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/project-store?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/project-store?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/state-store/project-store?op=request-change) |
**Parent:** [State Store](../)

**Status:** Conceptual

## Summary

The `ProjectStore` interface manages project-level state — the configuration back-reference and the auto-generated README.

## Interface

```go
package state

type ProjectStore interface {
    Config(ctx context.Context) (ProjectConfig, error)
    UpdateConfig(ctx context.Context, config ProjectConfig) error
    RebuildREADME(ctx context.Context) error
}
```

## Operations

### `Config`

Returns the project configuration stored in the state store. In the git backend, this reads `synchestra-state-repo.yaml` which contains the back-reference to the spec repository.

### `UpdateConfig`

Writes updated project configuration. Used during project initialization and when the spec repo location changes.

### `RebuildREADME`

Regenerates the auto-generated project overview README from current state. In the git backend, this rewrites the root `README.md` of the state repository.

See [Repo Config](https://github.com/specscore/specscore/blob/main/spec/features/repo-config/README.md) for the full configuration schema.

## Types

```go
type ProjectConfig struct {
    Title    string
    SpecRepos []string
}
```

## Open Questions

- Should `ProjectConfig` include additional fields like project description, owner, or creation timestamp?
- Should `RebuildREADME` be triggered automatically after state-mutating operations, or always explicitly?

---
*This document follows the https://specscore.md/feature-specification*
