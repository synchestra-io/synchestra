# --repos-dir

Root directory where repositories are stored on disk.

| Detail | Value |
|---|---|
| Type | String (directory path) |
| Required | No |
| Default | `~/synchestra/repos` |

## Supported by

| Command |
|---|
| [`config set`](../set/README.md) |
| [`config clear`](../clear/README.md) |

## Description

Maps to the `repos_dir` field in [`~/.synchestra.yaml`](../../../global-config/README.md). Repo references resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk.

For `config set`, provides the new value. Empty values are not allowed — use `config clear` instead to revert to the default.

For `config clear`, removes the `repos_dir` field from `~/.synchestra.yaml`, causing it to fall back to the default (`~/synchestra/repos`).

## Examples

```bash
# Set a custom repos directory
synchestra config set --repos-dir /data/synchestra/repos

# Clear back to default (~/synchestra/repos)
synchestra config clear --repos-dir
```

## Outstanding Questions

None at this time.
