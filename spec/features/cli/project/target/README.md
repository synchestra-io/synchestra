# Command Group: `synchestra project target`

**Parent:** [project](../README.md)

Manages target (code) repositories for a project. Target repos are where agents create branches and push implementation changes.

## Commands

| Command | Description |
|---|---|
| [add](add/README.md) | Add target repo(s) to the project |
| [remove](remove/README.md) | Remove target repo(s) from the project |

### `add`

Adds one or more target repos to the project's `repos` list in `synchestra-spec.yaml`. Clones repos if not on disk and writes `synchestra-target.yaml` to each. See [add/README.md](add/README.md).

### `remove`

Removes one or more target repos from the project's `repos` list in `synchestra-spec.yaml`. Does not delete `synchestra-target.yaml` from the target repos. See [remove/README.md](remove/README.md).

## Outstanding Questions

None at this time.
