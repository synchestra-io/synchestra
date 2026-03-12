# Feature: CLI

**Status:** In Progress

## Summary

The Synchestra CLI (`synchestra`) is the primary interface for agents and humans to interact with Synchestra-managed projects. It validates inputs, enforces state transitions, and handles the git commit-and-push mechanics so callers don't have to.

## Design Principles

### Command hierarchy

Commands follow a `synchestra <resource> <action>` pattern:

```
synchestra task claim
synchestra task status
synchestra task release
synchestra task list
synchestra skills list
synchestra skills show
...
```

### Exit code contract

All commands share a consistent exit code contract:

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `1` | Conflict (e.g., another agent claimed first, status changed since last read) |
| `2` | Invalid arguments |
| `3` | Resource not found |
| `4` | Invalid state transition |
| `10+` | Unexpected errors |

On non-zero exit, a human-readable explanation is written to stderr.

### Git mechanics

Commands that mutate state (claim, status change, release) perform an atomic commit-and-push. If the push fails due to a remote conflict, the command pulls, checks whether the intended operation is still valid, and either retries or fails with an appropriate exit code.

Commands that only read state (list, status query) do a pull first to ensure freshness.

## Task Statuses

### Status values

| Status | Description |
|---|---|
| `planning` | Task is being defined, requirements are being gathered |
| `queued` | Task is fully defined and ready for an agent to claim |
| `claimed` | An agent has claimed the task but not yet started work |
| `in_progress` | Agent is actively working on the task |
| `completed` | Task finished successfully |
| `failed` | Task failed (reason recorded) |
| `blocked` | Task is blocked on a dependency or decision |
| `aborted` | Task was aborted (terminal) |

### Status transitions

```
planning → queued → claimed → in_progress → completed
                                           → failed
                                           → blocked → in_progress (when unblocked)
                                           → aborted
                             → aborted (claimed but aborted before starting)
                   → blocked (queued but blocked on a dependency)
```

### The `abort_requested` flag

`abort_requested` is a flag, not a status. It can be set on a task that is `claimed` or `in_progress` — the task retains its current status while the flag signals that the agent should stop work and transition to `aborted`.

Why a flag and not a status:
- The agent needs to know the task's actual state (`in_progress`) to clean up properly
- A status change would lose the previous state
- The agent is the one that transitions to `aborted` after seeing the flag — it's a request, not a command

The `synchestra task status` command includes the `abort_requested` flag in its output when set.

## Command Groups

| Group | Description |
|---|---|
| [task](task/README.md) | Task management — claiming, status, progress |

See each command group for its subcommands and linked skills.

## Outstanding Questions

- How does the CLI discover which project repo to operate on — current directory, explicit flag, or config file?
- Should the CLI support `--dry-run` for mutation commands?
