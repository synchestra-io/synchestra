# `task status` Arguments

**Parent:** [status](../README.md)

Arguments specific to the `synchestra task status` command (update mode).

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--current`](current.md) | String | For update | Expected current status (guard) |
| [`--new`](new.md) | String | For update | Target status to transition to |

Also uses [`--project`](../../../_args/project.md), [`--task`](../../_args/task.md), and [`--reason`](../../_args/reason.md).

### `--current`

Guards against stale state by asserting the expected current status. See [current.md](current.md).

### `--new`

Specifies the target status for the transition. See [new.md](new.md).

## Outstanding Questions

None at this time.
