# cli/internal

Internal packages shared across CLI commands. These packages are not exported outside the `cli` module.

## Contents

| Directory | Description |
|---|---|
| [`exitcode/`](exitcode/README.md) | Typed exit-code error type used by all CLI commands |
| [`gitops/`](gitops/README.md) | Injectable `Runner` struct wrapping real git subprocess calls |
| [`globalconfig/`](globalconfig/README.md) | Loads `~/.synchestra.yaml`; provides `repos_dir` default |
| [`reporef/`](reporef/README.md) | Parses and resolves repository references to local paths and origin URLs |

## Outstanding Questions

None at this time.
