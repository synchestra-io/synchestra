# CLI

The `synchestra` command-line interface implementation. Built with [Cobra](https://github.com/spf13/cobra) and [fang](https://charm.land/fang).

## Packages

| Package | Description |
|---|---|
| [globalconfig](globalconfig/) | Global user configuration (`~/.synchestra.yaml`) |
| [gitops](gitops/) | Git operations — clone, commit-and-push, repo validation |
| [project](project/) | `synchestra project` command group |
| [reporef](reporef/) | Repository reference parsing and resolution |

### `globalconfig`

Reads the global Synchestra configuration from `~/.synchestra.yaml` and resolves the `repos_dir` setting with `~` expansion and default fallback to `~/synchestra/repos`.

### `gitops`

Thin wrapper around git CLI operations used by commands that mutate repositories. Provides clone, commit-and-push (with retry on conflict), pull, repo validation, and origin URL retrieval.

### `project`

Implements the `synchestra project` command group. Currently contains the `new` subcommand which creates a Synchestra project by linking spec, state, and target repos.

### `reporef`

Parses repository references in any of three formats (HTTPS URL, SSH URL, short `hosting/org/repo` path), resolves them to local disk paths under `repos_dir`, and provides canonical HTTPS origin URLs.

## Outstanding Questions

None at this time.
