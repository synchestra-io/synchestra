# Command: `synchestra task info`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-info](../../../../../skills/synchestra-task-info/README.md)

## Synopsis

```
synchestra task info --project <project_id> --task <task_path> [--format <text|json|yaml>]
```

## Description

Displays the full context for a task — everything an agent needs to understand what the task requires before starting work.

This is a **read-only** command. It pulls the latest state from the project repo but does not mutate anything.

Includes:
- **Task description** — the full task README content
- **Status** — current status fields (a superset of what `task status` returns)
- **Parent chain** — ancestor tasks for hierarchical context
- **Sibling tasks** — other tasks at the same level for awareness of related work
- **Outstanding questions** — unresolved questions that may affect approach
- **Linked feature spec** — reference to the feature specification, if any

More detailed than `task status`, which only shows status fields. Use `task info` when you need to understand the full picture of what a task involves; use `task status` when you only need to check or update the status.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../$args/project.md) | Yes | Project identifier |
| [`--task`](../$args/task.md) | Yes | Task path using `/` as separator |
| [`--format`](../$args/format.md) | No | Output format: `text` (default), `json`, `yaml` |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Task not found |

## Behaviour

1. Pull latest state from the project repo
2. Locate the task by path
3. Read the task README, status, parent chain, siblings, and linked feature spec
4. Render the assembled context in the requested format
5. Print to stdout and exit `0`

## Outstanding Questions

- Should `task info` include auto-generated minimal context (the parent chain, sibling awareness) or just the raw task README? Including assembled context is more useful for agents but adds coupling to the output format.
- How deep should the parent chain go? All the way to the project root, or limited to a fixed depth (e.g., 3 levels)?
