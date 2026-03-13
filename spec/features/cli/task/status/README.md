# Command: `synchestra task status`

**Parent:** [task](../README.md)
**Skill:** [synchestra-task-status](../../../../../skills/synchestra-task-status/README.md)

## Synopsis

```
# Query current status
synchestra task status --project <project_id> --task <task_path>

# Update status
synchestra task status --project <project_id> --task <task_path> --current <status> --new <status> [--reason <text>]
```

## Description

Queries or updates the status of a task.

**Query mode** (no `--current`/`--new`): Pulls latest state and prints the task's current status, including the `abort_requested` flag if set. Exits `0` on success.

**Update mode** (with `--current` and `--new`): Transitions the task from one status to another. The `--current` parameter acts as a guard — the command fails with exit code `1` if the task's actual status doesn't match `--current`. This prevents agents from blindly overwriting a status that changed since they last checked.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../_args/project.md) | Yes | Project identifier |
| [`--task`](../_args/task.md) | Yes | Task path using `/` as separator |
| [`--current`](_args/current.md) | For update | Expected current status (guard against stale state) |
| [`--new`](_args/new.md) | For update | Target status to transition to |
| [`--reason`](../_args/reason.md) | No | Reason for the transition (required for `failed` and `blocked`, optional otherwise) |

## Valid status values

`planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, `aborted`

See [CLI feature spec](../../README.md#task-statuses) for the full transition diagram.

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success (status queried or updated) |
| `1` | Status mismatch — actual status doesn't match `--current` (someone changed it) |
| `2` | Invalid arguments |
| `3` | Task not found |
| `4` | Invalid state transition (e.g., `planning` → `completed`) |

## Query output

```
status: in_progress
run: 4821
model: sonnet
claimed_at: 2026-03-12T10:32:00Z
updated_at: 2026-03-12T10:45:00Z
abort_requested: false
```

When `abort_requested` is `true`, agents should wrap up current work and transition to `aborted`.

## Behaviour

### Query mode

1. Pull latest state from the state repository
2. Read and print task status

### Update mode

1. Pull latest state from the state repository
2. Read current task status
3. If actual status != `--current`, exit `1` with message showing actual status
4. Validate the transition (e.g., `planning` → `completed` is invalid)
5. Update status, record reason and timestamp
6. Commit and push
7. On push conflict: pull, re-check `--current` guard, retry or fail

## Outstanding Questions

- Should query output be YAML, JSON, or a simple key-value format? Should there be a `--format` flag?
- Should `--reason` be strictly required for `failed`/`blocked`, or just strongly encouraged?
