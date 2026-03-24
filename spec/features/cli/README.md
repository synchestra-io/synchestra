# Feature: CLI

**Status:** In Progress

## Summary

The Synchestra CLI (`synchestra`) is the primary interface for agents and humans to interact with Synchestra-managed projects. It validates inputs, enforces state transitions, and handles the git commit-and-push mechanics so callers don't have to.

## Design Principles

### Command hierarchy

Commands follow a `synchestra <resource> <action>` pattern using **singular nouns** and **verb subcommands**:

```
synchestra config show
synchestra config set
synchestra config clear
synchestra project new
synchestra project info
synchestra project set
synchestra project code add
synchestra task claim
synchestra task status
synchestra task release
synchestra task list
synchestra skill list
synchestra skill show
synchestra code deps
synchestra server project list
synchestra server project add
```

**Singular nouns** — resource names are always singular (`task`, `skill`, `project`), never plural. This matches the convention used by `gh` (`gh repo list`, `gh issue new`), `kubectl` (`kubectl pod list`), and most modern CLIs. The resource name identifies the *type*, not a collection.

**Verb subcommands** — every action is an explicit subcommand. A bare resource name (e.g., `synchestra task`) shows help, never performs an implicit action like listing. Common verbs: `list`, `new`, `show`, `delete`, `update`.

**Prefer `new` over `create`** — use `new` as the subcommand for creating resources (e.g., `task new`, `project new`, `skill new`). The word `create` is intentionally avoided to stay consistent across all Synchestra commands and to align with tools like `gh` (`gh issue new`, `gh pr new`).

**Nesting** — sub-resources nest under their parent: `synchestra server project add`. Limit nesting to three levels (`<group> <resource> <action>`) to keep commands ergonomic.

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

Commands that mutate state (claim, status change, release) commit on the agent's branch and merge to local main. Whether the merge is pushed to the remote depends on the project's [sync policy](../state-store/backends/git/README.md#sync-policy) — by default (`on_commit`), every mutation pushes immediately. If the push fails due to a remote conflict, the command pulls, checks whether the intended operation is still valid, and either retries or fails with an appropriate exit code.

Commands that only read state (list, status query) pull first to ensure freshness when the sync policy calls for it.

**Contended operations** (`task claim`) always force an immediate pull+push round-trip regardless of the sync policy, to preserve optimistic locking.

Manual sync is available via [`synchestra state pull/push/sync`](state/README.md).

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

## The `_args` Directory Convention

CLI arguments are documented in `_args/` directories at the level where the argument applies:

- **Global arguments** — `spec/features/cli/_args/` — available to all commands (e.g., `--project`)
- **Command-group arguments** — `spec/features/cli/task/_args/` — shared across subcommands (e.g., `--task`, `--reason`, `--format`)
- **Command-specific arguments** — `spec/features/cli/task/new/_args/` — unique to one command (e.g., `--title`, `--enqueue`)

### File format

Each argument has its own `.md` file named after the flag (without `--`). For example, `--project` is documented in `project.md`. Every `_args/` directory also has a `README.md` with an argument index table and a brief summary per argument.

Each argument document contains:

1. **Heading** — `# --<flag-name>`
2. **Summary line** — one-sentence description
3. **Details table** — Type, Required, Default
4. **Supported by** — which commands use this argument
5. **Description** — full explanation
6. **Examples** — usage examples
7. **Outstanding Questions** — required section (use "None at this time." if empty)

### Placement rules

Place an argument at the **highest level where it is consistently meaningful**:

- If used by all CLI commands → `spec/features/cli/_args/`
- If used by multiple subcommands in a group → `spec/features/cli/task/_args/`
- If unique to one command → `spec/features/cli/task/<command>/_args/`

Arguments used by several (but not all) subcommands still go at the group level with a "Supported by" table listing the applicable commands.

### Linking

All CLI command READMEs and skill READMEs link to the canonical `_args` document when mentioning an argument. This ensures a single source of truth per argument.

## Command Groups

For an overview of which commands run in which environments (host, agent container, CI), see [Command Environments](command-environments.md).

| Entry | Description |
|---|---|
| [_args](_args/README.md) | Global CLI arguments |
| [config](config/README.md) | Global user configuration management |
| [project](project/README.md) | Project creation and management |
| [task](task/README.md) | Task management — claiming, status, progress |
| [state](state/README.md) | State repository synchronization — pull, push, sync |
| [feature](feature/README.md) | Feature queries — listing, hierarchy, dependencies, references |
| [code](code/README.md) | Code queries — source file dependencies on Synchestra resources |
| [serve](serve/README.md) | Foreground dev server — HTTP, HTTPS, MCP |
| [server](server/README.md) | Background daemon management |
| [mcp](mcp/README.md) | stdio MCP server for AI agents |
| [test](test/README.md) | Run and list markdown-native test scenarios |

### `_args`

Global arguments available to all `synchestra` commands. Contains `--project`, `--path`, and `--format`.

### `config`

Manages the global user configuration at `~/.synchestra.yaml` — show effective config with defaults, set values, and clear values back to defaults. See [config/README.md](config/README.md).

### `project`

Project creation and management — creating projects, viewing configuration, updating settings, and managing code repositories. See [project/README.md](project/README.md).

### `task`

See each command group for its subcommands and linked skills.

### `state`

Manual state repository synchronization — pull from origin, push to origin, or full bidirectional sync. These commands are policy-unaware escape hatches that execute immediately regardless of the project's sync policy. Operates on the state repo as a whole (tasks, chats, project config). See [state/README.md](state/README.md).

### `feature`

Read-only commands for querying features — flat listing, hierarchical tree, dependency lookup (`deps`), and reverse-reference lookup (`refs`). Features are identified by their directory path under the features directory using `/` as separator. See [feature/README.md](feature/README.md).

### `code`

Read-only commands for querying source code relationships to Synchestra resources. Where `feature` operates on the spec → spec graph, `code` operates on the code → spec graph — scanning source files for [source references](../source-references/README.md) (`synchestra:` annotations and expanded URLs). See [code/README.md](code/README.md).

### `spec`

Commands for validating and searching Synchestra specification repositories. Includes structural convention checking (`lint`) and keyword search with spec-aware scoping (`search`). See [spec/README.md](spec/README.md).

### `serve`

Starts a foreground Synchestra server exposing API over HTTP, HTTPS, and/or MCP. Designed for interactive development — logs stream to stdout, Ctrl+C to stop. See [serve/README.md](serve/README.md).

### `server`

Background daemon management — start, stop, restart, status, and project configuration. All settings come from `synchestra-server.yaml`. See [server/README.md](server/README.md).

### `mcp`

Starts a stdio-based MCP server for AI agent tools (Claude Code, Cursor, etc.). Designed to be launched as a subprocess. See [mcp/README.md](mcp/README.md).

### `test`

Runs and lists markdown-native test scenarios powered by [Rehearse](https://github.com/synchestra-io/rehearse). Delegates to the test runner for scenario parsing, AC resolution, and execution. See [test/README.md](test/README.md).

## Outstanding Questions

- Should the CLI support `--dry-run` for mutation commands?
