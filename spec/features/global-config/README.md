# Global Configuration

**Status:** Conceptual

## Summary

The global Synchestra configuration file (`~/.synchestra.yaml`) stores user-level settings that apply across all projects and CLI invocations. It is the single source of truth for machine-local preferences such as where repositories are stored on disk.

## Location

`~/.synchestra.yaml` — in the user's home directory.

If the file does not exist, all settings fall back to their defaults. The file is created by [`synchestra config set`](../cli/config/set/README.md) when the user first sets a value.

## Schema

```yaml
# Where cloned repositories are stored on disk.
# Repo references resolve to {repos_dir}/{hosting}/{org}/{repo}.
repos_dir: ~/synchestra/repos
```

## Go Type Definition

```go
// GlobalConfig represents the user-level Synchestra configuration
// read from ~/.synchestra.yaml.
type GlobalConfig struct {
	// ReposDir is the root directory where repositories are stored on disk.
	// Repo references resolve to {ReposDir}/{hosting}/{org}/{repo}.
	// Default: ~/synchestra/repos
	ReposDir string `yaml:"repos_dir"`
}
```

## Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `repos_dir` | String (directory path) | No | `~/synchestra/repos` | Root directory for cloned repositories |

### `repos_dir`

The root directory where Synchestra stores cloned repositories. When a repo reference like `github.com/acme/acme-api` is resolved, it maps to `{repos_dir}/github.com/acme/acme-api` on disk.

Supports `~` expansion for the home directory. Relative paths are resolved relative to the user's home directory.

### Repo resolution

Given a repo reference (full git URL or `{hosting}/{org}/{repo}` short form), the CLI resolves it to a local path:

1. Parse the reference to extract `{hosting}/{org}/{repo}` (e.g., `https://github.com/acme/acme-api` → `github.com/acme/acme-api`)
2. Join with `repos_dir`: `{repos_dir}/github.com/acme/acme-api`
3. If the directory exists, use it. If not, clone the repo there.

### Validation

- If `repos_dir` is set, it must be a valid directory path (or creatable).
- The CLI creates `repos_dir` and intermediate directories if they do not exist when cloning a repo for the first time.

## CLI Commands

The [`synchestra config`](../cli/config/README.md) command group provides `show`, `set`, and `clear` subcommands to manage this file.

## Outstanding Questions

- Should the file support additional settings beyond `repos_dir` (e.g., default `--format`, default git remote name)?
- Should `repos_dir` support environment variable interpolation (e.g., `$HOME/synchestra/repos`)?
