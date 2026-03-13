# Command: `synchestra task enqueue`

**Parent:** [task](../README.md)

**Skill:** [synchestra-task-enqueue](../../../../../skills/synchestra-task-enqueue/README.md)

## Synopsis

```
synchestra task enqueue --project <project> --task <task>
```

## Description

Transitions a task from `planning` to `queued` status. This marks the task as
fully defined and ready for an agent to discover and claim.

The command implicitly guards on `--current planning` -- it will fail if the
task is not currently in `planning` status. This prevents accidental
re-enqueuing of tasks that have already progressed past the planning stage.

Once queued, agents can discover the task via `synchestra task list --status queued`
and claim it with `synchestra task claim`.

The status change is performed as an atomic commit-and-push.

## Parameters

| Parameter   | Required | Description                          |
|-------------|----------|--------------------------------------|
| [`--project`](../../$args/project.md) | Yes      | Project identifier                   |
| [`--task`](../$args/task.md)    | Yes      | Task identifier to enqueue           |

The `--current planning` guard is applied implicitly and cannot be overridden.

## Exit codes

| Code | Meaning                  |
|------|--------------------------|
| 0    | Success                  |
| 1    | Status mismatch          |
| 2    | Invalid arguments        |
| 3    | Task not found           |
| 4    | Invalid state transition |

## Behaviour

1. Validate that `--project` and `--task` are provided; exit **2** if not.
2. Locate the task within the project; exit **3** if the task does not exist.
3. Read the current status of the task.
4. If the current status is not `planning`, exit **1** (status mismatch).
5. Transition the task status from `planning` to `queued`.
6. Atomically commit and push the change.
7. Exit **0** on success.

If the state transition is rejected for any other reason, exit **4**.

## Outstanding Questions

- Should there be a bulk enqueue command for multiple tasks at once?
