# Command Group: `synchestra task`

**Parent:** [CLI](../README.md)

Commands for managing tasks — claiming, querying status, updating progress, and more.

## Arguments

Shared arguments for `synchestra task` subcommands are documented in the [_args](_args/README.md) directory: [`--task`](_args/task.md), [`--reason`](_args/reason.md), and [`--format`](_args/format.md).

## Commands

| Command | Description | Skill |
|---|---|---|
| [new](new/README.md) | Create a new task (in `planning` or `queued`) | [synchestra-task-new](../../../../skills/synchestra-task-new/README.md) |
| [enqueue](enqueue/README.md) | Move a task from `planning` to `queued` | [synchestra-task-enqueue](../../../../skills/synchestra-task-enqueue/README.md) |
| [claim](claim/README.md) | Claim a queued task for work | [synchestra-claim-task](../../../../skills/synchestra-claim-task/README.md) |
| [start](start/README.md) | Begin work on a claimed task (claimed → in_progress) | [synchestra-task-start](../../../../skills/synchestra-task-start/README.md) |
| [status](status/README.md) | Query or update task status | [synchestra-task-status](../../../../skills/synchestra-task-status/README.md) |
| [complete](complete/README.md) | Mark a task as completed | [synchestra-task-complete](../../../../skills/synchestra-task-complete/README.md) |
| [fail](fail/README.md) | Mark a task as failed with reason | [synchestra-task-fail](../../../../skills/synchestra-task-fail/README.md) |
| [block](block/README.md) | Mark a task as blocked with reason | [synchestra-task-block](../../../../skills/synchestra-task-block/README.md) |
| [unblock](unblock/README.md) | Resume a blocked task (blocked → in_progress) | [synchestra-task-unblock](../../../../skills/synchestra-task-unblock/README.md) |
| [release](release/README.md) | Release a claimed task back to queued | [synchestra-task-release](../../../../skills/synchestra-task-release/README.md) |
| [abort](abort/README.md) | Request abortion of a task (sets flag) | [synchestra-task-abort](../../../../skills/synchestra-task-abort/README.md) |
| [aborted](aborted/README.md) | Report a task has been aborted (terminal) | [synchestra-task-aborted](../../../../skills/synchestra-task-aborted/README.md) |
| [list](list/README.md) | List tasks with optional filtering | [synchestra-task-list](../../../../skills/synchestra-task-list/README.md) |
| [info](info/README.md) | Show full task details and context | [synchestra-task-info](../../../../skills/synchestra-task-info/README.md) |

## Sync Behaviour

All task subcommands respect the project's [sync policy](../../state-store/backends/git/README.md#sync-policy). By default (`on_commit`), mutation commands push immediately and read commands pull first. Under deferred policies (`manual`, `on_session_end`, `on_interval`), pull and push happen according to the policy — not unconditionally.

To override the policy for a single invocation, use the global [`--sync`](../_args/sync.md) flag:

- `--sync remote` — force immediate pull+push (useful when an orchestrator needs a claim or completion to be visible to remote agents right away).
- `--sync local` — suppress all remote I/O (useful for batched reads or when the remote is unreachable).

See [`synchestra state pull/push/sync`](../state/README.md) for manual bulk sync.

## Outstanding Questions

None at this time.
