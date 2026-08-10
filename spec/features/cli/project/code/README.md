# Command Group: `synchestra project code`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project/code?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project/code?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project/code?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project/code?op=request-change) |
**Parent:** [project](../README.md)

Manages code repositories for a project. Code repos are where agents create branches and push implementation changes.

## Commands

| Command | Description |
|---|---|
| [add](add/README.md) | Add code repo(s) to the project |
| [remove](remove/README.md) | Remove code repo(s) from the project |

### `add`

Adds one or more code repos to the project's `repos` list in `synchestra-spec-repo.yaml`. Clones repos if not on disk and writes `synchestra-code-repo.yaml` to each. See [add/README.md](add/README.md).

### `remove`

Removes one or more code repos from the project's `repos` list in `synchestra-spec-repo.yaml`. Does not delete `synchestra-code-repo.yaml` from the code repos. See [remove/README.md](remove/README.md).

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
