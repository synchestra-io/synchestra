# Task Arguments

**Parent:** [task](../README.md)

Arguments shared across `synchestra task` subcommands.

## Arguments

| Argument | Type | Required | Supported by |
|---|---|---|---|
| [`--task`](task.md) | String | Yes | All task subcommands except `list` |
| [`--reason`](reason.md) | String | Varies | `status`, `fail`, `block`, `unblock`, `release`, `abort`, `aborted` |
| [`--format`](format.md) | String | No | `list`, `info` |

### `--task`

Identifies the task to operate on using a `/`-separated path. Required by all task subcommands except `list`. See [task.md](task.md).

### `--reason`

Records why a transition is happening. Required for `fail` and `block`, optional for other supported commands. See [reason.md](reason.md).

### `--format`

Controls the output format of read commands. Supported formats vary by command. See [format.md](format.md).

## Outstanding Questions

None at this time.
