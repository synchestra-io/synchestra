# Command Group: `synchestra project`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/project?op=request-change) |
**Parent:** [CLI](../README.md)

Read-only and lifecycle-management commands for an existing Synchestra-managed project — viewing configuration, updating settings, and managing the project's code repositories list.

> **Note.** Project bootstrap moved to the top-level [`synchestra init`](../init/README.md) command (per the [`unified-project-definition`](../../../ideas/unified-project-definition.md) Idea and the [`cli/init`](../init/README.md) Feature). The legacy `synchestra project init` and `synchestra project new` subcommands have been removed entirely. The subcommands listed below are designs awaiting implementation against the unified specscore.yaml + synchestra.yaml model.

## Project definition

A Synchestra-managed project is defined by two files at the repository root:

- [`specscore.yaml`](https://github.com/specscore/specscore/blob/main/spec/features/repo-config/README.md) — project identity (`title`, `host`, `org`, `repo`, role-tagged `repositories`).
- [`synchestra.yaml`](../../repo-config/README.md) — orchestration metadata (`state` block with mode + sync, optional `hub` registration).

State locations carry a single self-identifier file ([`synchestra-state.yaml`](../../state-repo-config/README.md)) — embedded mode places it on the orphan branch worktree; separate-repo mode places it at the state repo root.

## Commands (planned, not yet implemented)

| Command | Description |
|---|---|
| [info](info/README.md) | Display project configuration |
| [set](set/README.md) | Update project settings |
| [code](code/README.md) | Manage code repositories |

### `info`

Displays the project's effective configuration by reading both `specscore.yaml` and `synchestra.yaml` and presenting the composed view. See [info/README.md](info/README.md).

### `set`

Updates project configuration — identity fields land in `specscore.yaml`, orchestration fields in `synchestra.yaml`. See [set/README.md](set/README.md).

### `code`

Sub-group for managing code repositories — operates on `specscore.yaml#project.repositories` (role-tagged entries per the SpecScore [Repo Config](https://github.com/specscore/specscore/blob/main/spec/features/repo-config/README.md) Feature). Contains `add` and `remove` subcommands. See [code/README.md](code/README.md).

## Open Questions

- The implementation of `info`, `set`, and `code` commands against the new two-file model is tracked separately; this Feature documents only the design.
- Whether `set` should ever write to `specscore.yaml` (project identity) or limit itself to the `synchestra.yaml` orchestration fields — to be decided when `set` is specified.

---
*This document follows the https://specscore.md/feature-specification*
