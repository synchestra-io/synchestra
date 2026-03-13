# Skill: synchestra-task-create

Create a new task in a project. The task is created in `planning` status by default, with its own directory and `README.md`. The parent's task board is updated automatically.

**CLI reference:** [synchestra task create](../../spec/features/cli/task/create/README.md)

## When to use

- You need to add a new task to a project's task board
- You are breaking down a parent task into subtasks
- You want to queue a task for immediate pickup by passing `--enqueue`

## Command

```bash
synchestra task create \
  --project <project_id> \
  --task <task_path> \
  --title <title> \
  [--description <description>] \
  [--depends-on <deps>] \
  [--enqueue]
```

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--project`](../../spec/features/cli/_args/project.md) | Yes | Project identifier (e.g., `synchestra`, `my-service`) |
| [`--task`](../../spec/features/cli/task/_args/task.md) | Yes | Task path using `/` as separator (e.g., `new-task`, `parent-task/new-subtask`) |
| [`--title`](../../spec/features/cli/task/create/_args/title.md) | Yes | Human-readable title for the task |
| [`--description`](../../spec/features/cli/task/create/_args/description.md) | No | Task description; written into the task's `README.md` |
| [`--depends-on`](../../spec/features/cli/task/create/_args/depends-on.md) | No | Comma-separated list of task paths this task depends on (e.g., `setup-db,create-schema`) |
| [`--enqueue`](../../spec/features/cli/task/create/_args/enqueue.md) | No | Flag; creates the task in `queued` status instead of `planning` |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Task created successfully | Proceed with next steps (e.g., claim the task, create subtasks) |
| `1` | Conflict — remote state changed during push | Re-pull and retry the creation |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Parent task not found | Verify the parent task path exists before creating a subtask |
| `4` | Task already exists | Use a different task path, or check the existing task's status |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Basic task creation

```bash
synchestra task create \
  --project synchestra \
  --task implement-cli \
  --title "Implement CLI framework"
```

### Create a task and enqueue it immediately

```bash
synchestra task create \
  --project synchestra \
  --task fix-auth-bug \
  --title "Fix authentication bypass bug" \
  --description "Users can bypass auth by sending an empty token header" \
  --enqueue
```

### Create a nested subtask

```bash
synchestra task create \
  --project synchestra \
  --task implement-cli/parse-arguments \
  --title "Parse CLI arguments" \
  --description "Implement argument parsing for all task subcommands"
```

### Create a task with dependencies

```bash
synchestra task create \
  --project my-service \
  --task run-migrations \
  --title "Run database migrations" \
  --depends-on setup-db,create-schema
```

## Notes

- Creation is atomic -- it commits the new task files and pushes to the project repo. If the push fails due to a conflict, the creation fails.
- The parent task must already exist when creating a nested subtask. If it does not, the command exits with code `3`.
- The default status is `planning`. Use `--enqueue` to skip planning and place the task directly in `queued` status.
- The task directory and `README.md` are created automatically; do not create them manually before running this command.

## Outstanding Questions

- Should there be a `--assignee` / `--requester` parameter?
- Should the description be read from stdin if not provided as a flag?
