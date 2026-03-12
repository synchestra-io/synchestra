# Skills

Synchestra skills for AI agents. Each skill wraps a single Synchestra CLI command with clear trigger conditions, parameters, and exit code handling.

See the [agent-skills feature spec](../spec/features/agent-skills/README.md) for design principles and the full skill format.

## Available Skills

| Skill | Description | CLI Command |
|---|---|---|
| [synchestra-task-create](synchestra-task-create/README.md) | Create a new task | [task create](../spec/features/cli/task/create/README.md) |
| [synchestra-task-enqueue](synchestra-task-enqueue/README.md) | Move a task from planning to queued | [task enqueue](../spec/features/cli/task/enqueue/README.md) |
| [synchestra-claim-task](synchestra-claim-task/README.md) | Claim a task before starting work on it | [task claim](../spec/features/cli/task/claim/README.md) |
| [synchestra-task-start](synchestra-task-start/README.md) | Begin work on a claimed task | [task start](../spec/features/cli/task/start/README.md) |
| [synchestra-task-status](synchestra-task-status/README.md) | Query or update task status | [task status](../spec/features/cli/task/status/README.md) |
| [synchestra-task-complete](synchestra-task-complete/README.md) | Mark a task as completed | [task complete](../spec/features/cli/task/complete/README.md) |
| [synchestra-task-fail](synchestra-task-fail/README.md) | Mark a task as failed with reason | [task fail](../spec/features/cli/task/fail/README.md) |
| [synchestra-task-block](synchestra-task-block/README.md) | Mark a task as blocked | [task block](../spec/features/cli/task/block/README.md) |
| [synchestra-task-unblock](synchestra-task-unblock/README.md) | Resume a blocked task | [task unblock](../spec/features/cli/task/unblock/README.md) |
| [synchestra-task-release](synchestra-task-release/README.md) | Release a claimed task back to queued | [task release](../spec/features/cli/task/release/README.md) |
| [synchestra-task-abort](synchestra-task-abort/README.md) | Request abortion of a task | [task abort](../spec/features/cli/task/abort/README.md) |
| [synchestra-task-aborted](synchestra-task-aborted/README.md) | Report a task has been aborted | [task aborted](../spec/features/cli/task/aborted/README.md) |
| [synchestra-task-list](synchestra-task-list/README.md) | List tasks with filtering | [task list](../spec/features/cli/task/list/README.md) |
| [synchestra-task-info](synchestra-task-info/README.md) | Show full task details and context | [task info](../spec/features/cli/task/info/README.md) |

## Outstanding Questions

None at this time.
