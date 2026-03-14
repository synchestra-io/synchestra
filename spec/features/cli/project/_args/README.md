# Project Arguments

**Parent:** [project](../README.md)

Arguments shared across `synchestra project` subcommands.

## Arguments

| Argument | Type | Required | Supported by |
|---|---|---|---|
| [`--spec-repo`](spec-repo.md) | String (repo reference) | Varies | `new`, `set` |
| [`--state-repo`](state-repo.md) | String (repo reference) | Varies | `new`, `set` |
| [`--target-repo`](target-repo.md) | String (repo reference) | Varies | `new`, `target add`, `target remove` |

### `--spec-repo`

Reference to the project's spec repository. Required for `new`, optional for `set`. See [spec-repo.md](spec-repo.md).

### `--state-repo`

Reference to the project's state repository. Required for `new`, optional for `set`. See [state-repo.md](state-repo.md).

### `--target-repo`

Reference to a target (code) repository. Repeatable for multiple targets. Required for `new` and `target add/remove`. See [target-repo.md](target-repo.md).

## Outstanding Questions

None at this time.
