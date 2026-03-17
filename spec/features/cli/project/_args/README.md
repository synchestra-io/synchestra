# Project Arguments

**Parent:** [project](../README.md)

Arguments shared across `synchestra project` subcommands.

## Arguments

| Argument | Type | Required | Supported by |
|---|---|---|---|
| [`--spec-repo`](spec-repo.md) | String (repo reference) | Varies | `new`, `set` |
| [`--state-repo`](state-repo.md) | String (repo reference) | Varies | `new`, `set` |
| [`--code-repo`](code-repo.md) | String (repo reference) | Varies | `new`, `code add`, `code remove` |

### `--spec-repo`

Reference to the project's spec repository. Required for `new`, optional for `set`. See [spec-repo.md](spec-repo.md).

### `--state-repo`

Reference to the project's state repository. Required for `new`, optional for `set`. See [state-repo.md](state-repo.md).

### `--code-repo`

Reference to a code repository. Repeatable for multiple code repos. Required for `new` and `code add/remove`. See [code-repo.md](code-repo.md).

## Outstanding Questions

None at this time.
