# Command Group: `synchestra project code`

**Parent:** [project](../README.md)

Manages code repositories for a project. Code repos are where agents create branches and push implementation changes.

## Commands

| Command | Description |
|---|---|
| [add](add/README.md) | Add code repo(s) to the project |
| [remove](remove/README.md) | Remove code repo(s) from the project |

### `add`

Adds one or more code repos to the project's `repos` list in `synchestra-spec.yaml`. Clones repos if not on disk and writes `synchestra-code.yaml` to each. See [add/README.md](add/README.md).

### `remove`

Removes one or more code repos from the project's `repos` list in `synchestra-spec.yaml`. Does not delete `synchestra-code.yaml` from the code repos. See [remove/README.md](remove/README.md).

## Outstanding Questions

None at this time.
