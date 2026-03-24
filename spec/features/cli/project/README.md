# Command Group: `synchestra project`

**Parent:** [CLI](../README.md)

Commands for creating and managing Synchestra projects — setting up spec, state, and code repositories, viewing project configuration, and modifying project settings.

## Arguments

Shared arguments for `synchestra project` subcommands are documented in the [_args](_args/README.md) directory: [`--spec-repo`](_args/spec-repo.md), [`--state-repo`](_args/state-repo.md), and [`--code-repo`](_args/code-repo.md).

## Repo Reference Format

All repo reference arguments (`--spec-repo`, `--state-repo`, `--code-repo`) accept either:

- **Full git URL:** `https://github.com/org/repo` or `git@github.com:org/repo`
- **Short path:** `github.com/org/repo`

Both forms resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk. The `repos_dir` is configured in `~/.synchestra.yaml` (default: `~/synchestra/repos/`).

Values are stored as origin URLs in config files, regardless of the input format.

## Config Files

Each repository type has a dedicated config file written to its root:

| Repo type | Config file | Purpose |
|---|---|---|
| Spec | `synchestra-spec-repo.yaml` | Full project definition (`title`, `state_repo`, `repos`) |
| State | `synchestra-state-repo.yaml` | Points to spec repos (`spec_repos`) |
| Code | `synchestra-code-repo.yaml` | Points to spec repos (`spec_repos`) |

## Commands

| Command | Description |
|---|---|
| [init](init/README.md) | Initialize embedded state in the current repo |
| [new](new/README.md) | Create a new project with dedicated repos |
| [info](info/README.md) | Display project configuration |
| [set](set/README.md) | Update project settings |
| [code](code/README.md) | Manage code repositories |

### `init`

Initializes Synchestra embedded state in the current git repository — creates an orphan branch, sets up a git worktree at `.synchestra/`, and you're ready to go. Zero-friction alternative to `project new` for single-repo projects. See [init/README.md](init/README.md).

### `new`

Creates a new project by linking a spec repo, state repo, and one or more code repos. Clones missing repos, writes config files to each, commits and pushes. See [new/README.md](new/README.md).

### `info`

Displays the contents of the spec repo's `synchestra-spec-repo.yaml` for the current project. See [info/README.md](info/README.md).

### `set`

Updates project configuration — change the spec or state repo reference, or set config values like `--allow-proposals=true`. See [set/README.md](set/README.md).

### `code`

Sub-group for managing code repositories. Contains `add` and `remove` subcommands. See [code/README.md](code/README.md).

## Outstanding Questions

- Should `synchestra project new` auto-initialize a git repo if a resolved directory exists but is not a git repo?
- Should there be a `synchestra project delete` command to tear down a project?
