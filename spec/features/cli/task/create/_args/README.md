# `task create` Arguments

**Parent:** [create](../README.md)

Arguments specific to the `synchestra task create` command.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--title`](title.md) | String | Yes | Human-readable title for the task |
| [`--description`](description.md) | String | No | Task description for the README.md |
| [`--depends-on`](depends-on.md) | String | No | Comma-separated list of dependency task paths |
| [`--enqueue`](enqueue.md) | Flag | No | Create in `queued` status instead of `planning` |

Also uses [`--project`](../../../_args/project.md) and [`--task`](../../_args/task.md).

### `--title`

Human-readable name for the task. See [title.md](title.md).

### `--description`

Body text written into the task's generated `README.md`. See [description.md](description.md).

### `--depends-on`

Declares dependencies on other tasks. See [depends-on.md](depends-on.md).

### `--enqueue`

Skips the `planning` phase and creates the task directly in `queued`. See [enqueue.md](enqueue.md).

## Outstanding Questions

None at this time.
