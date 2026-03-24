# Command Group: `synchestra state`

**Parent:** [CLI](../README.md)

Commands for manually synchronizing the project's state repository with its remote. These commands are policy-unaware — they execute immediately and unconditionally, regardless of the project's [sync policy](../../state-store/backends/git/README.md#sync-policy).

All state store entities (tasks, chats, project configuration) are synchronized together — sync operates on the state repository as a whole, not on individual entities.

## When to use

- **`manual` sync policy:** These commands are the only way to sync with the remote.
- **Any sync policy:** Use as escape hatches for immediate sync when the automatic policy hasn't triggered yet.
- **Debugging:** Verify the state repo is in sync, force a push after local-only operations, or pull to see remote changes.

## Commands

| Command | Description |
|---|---|
| [pull](pull/README.md) | Pull latest state from origin to local main |
| [push](push/README.md) | Push local main to origin |
| [sync](sync/README.md) | Full round-trip — pull then push |

### `pull`

Fetches the latest state from the remote origin, fast-forwards local main, and rebases active agent branches. Use when you need fresh state before reading (e.g., checking for new tasks or abort requests). See [pull/README.md](pull/README.md).

### `push`

Merges pending agent branch commits to local main and pushes to origin. Use when you want remote visibility of local changes (e.g., after completing tasks in `manual` mode). See [push/README.md](push/README.md).

### `sync`

Equivalent to `pull` followed by `push`, with conflict retry. The go-to command when you want to ensure full bidirectional sync. See [sync/README.md](sync/README.md).

## Outstanding Questions

- Should there be a `synchestra state info` subcommand to show current sync policy, last pull/push timestamps, and pending local commits?
- Should there be a `synchestra state status` subcommand to show sync health (e.g., "3 local commits unpushed, last pull 2m ago")?
