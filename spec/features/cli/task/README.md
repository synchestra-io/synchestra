# Command Group: `synchestra task`

**Parent:** [CLI](../README.md)

Commands for managing tasks — claiming, querying status, updating progress, and more.

## Arguments

Shared arguments for `synchestra task` subcommands are documented in the [_args](_args/README.md) directory: [`--task`](_args/task.md), [`--reason`](_args/reason.md), and [`--format`](_args/format.md).

## Commands

| Command | Description | Skill |
|---|---|---|
| [new](new/README.md) | Create a new task (in `planning` or `queued`) | [task: new](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/new.md) |
| [enqueue](enqueue/README.md) | Move a task from `planning` to `queued` | [task: enqueue](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/enqueue.md) |
| [claim](claim/README.md) | Claim a queued task for work | [task: claim](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/claim.md) |
| [start](start/README.md) | Begin work on a claimed task (claimed → in_progress) | [task: start](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/start.md) |
| [status](status/README.md) | Query or update task status | [task: status](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/status.md) |
| [complete](complete/README.md) | Mark a task as completed | [task: complete](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/complete.md) |
| [fail](fail/README.md) | Mark a task as failed with reason | [task: fail](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/fail.md) |
| [block](block/README.md) | Mark a task as blocked with reason | [task: block](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/block.md) |
| [unblock](unblock/README.md) | Resume a blocked task (blocked → in_progress) | [task: unblock](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/unblock.md) |
| [release](release/README.md) | Release a claimed task back to queued | [task: release](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/release.md) |
| [abort](abort/README.md) | Request abortion of a task (sets flag) | [task: abort](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/abort.md) |
| [aborted](aborted/README.md) | Report a task has been aborted (terminal) | [task: aborted](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/aborted.md) |
| [list](list/README.md) | List tasks with optional filtering | [task: list](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/list.md) |
| [info](info/README.md) | Show full task details and context | [task: info](https://github.com/synchestra-io/ai-plugin-synchestra/blob/main/skills/task/references/info.md) |

## Sync Behaviour

All task subcommands respect the project's [sync policy](../../state-store/backends/git/README.md#sync-policy). By default (`on_commit`), mutation commands push immediately and read commands pull first. Under deferred policies (`manual`, `on_session_end`, `on_interval`), pull and push happen according to the policy — not unconditionally.

To override the policy for a single invocation, use the global [`--sync`](../_args/sync.md) flag:

- `--sync remote` — force immediate pull+push (useful when an orchestrator needs a claim or completion to be visible to remote agents right away).
- `--sync local` — suppress all remote I/O (useful for batched reads or when the remote is unreachable).

See [`synchestra state pull/push/sync`](../state/README.md) for manual bulk sync.

## Outstanding Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
