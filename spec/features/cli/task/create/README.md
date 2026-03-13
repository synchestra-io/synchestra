# Command: `synchestra task create`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-create](../../../../../skills/synchestra-task-create/README.md)

## Synopsis

```
synchestra task create --project <project_id> --task <task_path> --title <title> [--description <description>] [--depends-on <deps>] [--enqueue]
```

## Description

Creates a new task in `planning` status by default. The task directory and a `README.md` containing the task description are created, and the parent's task board is updated with a new row.

If the `--enqueue` flag is passed, the task is created directly in `queued` status instead of `planning`, skipping the planning phase.

The `--task` parameter accepts nested paths using `/` as a separator (e.g., `parent-task/new-subtask`), which creates the task as a subtask of the specified parent. The parent task must already exist; if it does not, the command fails with exit code `3`.

Like all mutation commands, `task create` is atomic: the CLI commits the new task files and pushes to the project repo. If the push fails due to a remote conflict, the CLI pulls and checks whether the task can still be created. If yes, it retries. If no, it exits with code `1`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier |
| [`--task`](../$args/task.md) | Yes | Task path using `/` as separator (e.g., `new-task` or `parent-task/new-subtask`) |
| [`--title`]($args/title.md) | Yes | Human-readable title for the task |
| [`--description`]($args/description.md) | No | Task description; included in the generated `README.md` |
| [`--depends-on`]($args/depends-on.md) | No | Comma-separated list of task paths this task depends on |
| [`--enqueue`]($args/enqueue.md) | No | Flag; if passed, creates the task in `queued` status instead of `planning` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Task created successfully |
| `1` | Conflict — remote state changed during push |
| `2` | Invalid arguments |
| `3` | Parent task not found |
| `4` | Task already exists |

## Behaviour

1. Pull latest state from the project repo
2. Validate that the parent task exists (if the task path is nested)
3. Verify that no task with the given path already exists
4. Create the task directory and `README.md` with the title and description
5. Set the initial status to `planning` (or `queued` if `--enqueue` is passed)
6. Record dependencies if `--depends-on` is provided
7. Update the parent's task board with a new row for the created task
8. Commit and push
9. On push conflict: pull, re-check, retry or fail

## Outstanding Questions

- Should there be a `--assignee` / `--requester` parameter?
- Should the description be read from stdin if not provided as a flag?
