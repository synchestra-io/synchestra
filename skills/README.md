# Skills

Synchestra skills for AI agents. Each skill wraps a single Synchestra CLI command with clear trigger conditions, parameters, and exit code handling.

See the [agent-skills feature spec](../spec/features/agent-skills/README.md) for design principles and the full skill format.

## Skill File Format

Every `README.md` inside a skill directory **MUST** begin with a YAML frontmatter header containing `name` and `description` fields. This is required by the [Claude Code skills format](https://code.claude.com/docs/en/skills.md).

```yaml
---
name: synchestra-feature-list
description: Lists all features in a project. Use when listing features, exploring feature structure, or checking what features exist.
---
```

- **`name`** — the skill identifier (must match the directory name).
- **`description`** — a concise, action-oriented sentence describing what the skill does and when to invoke it. Include trigger phrases like "Use when…" so agents can match user intent to the right skill.

The rest of the file follows the standard skill body format (heading, context, parameters, exit codes, etc.).

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
| [synchestra-feature-list](synchestra-feature-list/README.md) | List all features in a project | [feature list](../spec/features/cli/feature/list/README.md) |
| [synchestra-feature-tree](synchestra-feature-tree/README.md) | Display feature hierarchy as a tree | [feature tree](../spec/features/cli/feature/tree/README.md) |
| [synchestra-feature-deps](synchestra-feature-deps/README.md) | Show features a feature depends on | [feature deps](../spec/features/cli/feature/deps/README.md) |
| [synchestra-feature-refs](synchestra-feature-refs/README.md) | Show features that reference a feature | [feature refs](../spec/features/cli/feature/refs/README.md) |

## Outstanding Questions

None at this time.
