# Command Group: `synchestra state`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state?op=request-change) |
**Parent:** [CLI](../README.md)

Commands for manually synchronizing the project's state repository with its remote, plus the `state-store/topology` operator/agent controls (mirror barriers today; `status`/`verify`/`replicate` remain future work — see [state-store/topology](../../state-store/topology/README.md#health-reconciliation-and-measurement)).

`pull`/`push`/`sync` are policy-unaware — they execute immediately and unconditionally, regardless of the project's [sync policy](../../state-store/backends/git/README.md#sync-policy) — and synchronize the state repository as a whole (tasks, chats, project configuration together, not per-entity). They are currently unimplemented stubs, deliberately not registered on the CLI's root command tree (`pkg/cli/main.go`) until they have a real implementation; their own subcommand docs below still describe the intended behavior. `wait` is a real, working command targeting a different layer entirely — the backend-neutral replication journal (`pkg/state/replication`), not the markdown-file task/chat state the other three describe.

## When to use

- **`manual` sync policy:** These commands are the only way to sync with the remote.
- **Any sync policy:** Use as escape hatches for immediate sync when the automatic policy hasn't triggered yet.
- **Debugging:** Verify the state repo is in sync, force a push after local-only operations, or pull to see remote changes.

## Commands

| Command | Description |
|---|---|
| [pull](pull/README.md) | Pull latest state from origin to local main (stub, not yet registered) |
| [push](push/README.md) | Push local main to origin (stub, not yet registered) |
| [sync](sync/README.md) | Full round-trip — pull then push (stub, not yet registered) |
| [wait](wait/README.md) | Block until a named replica durably applies events up to a cursor (mirror barrier) |

### `pull`

Fetches the latest state from the remote origin, fast-forwards local main, and rebases active agent branches. Use when you need fresh state before reading (e.g., checking for new tasks or abort requests). See [pull/README.md](pull/README.md).

### `push`

Merges pending agent branch commits to local main and pushes to origin. Use when you want remote visibility of local changes (e.g., after completing tasks in `manual` mode). See [push/README.md](push/README.md).

### `sync`

Equivalent to `pull` followed by `push`, with conflict retry. The go-to command when you want to ensure full bidirectional sync. See [sync/README.md](sync/README.md).

### `wait`

Blocks until a named replica endpoint has durably applied journal events up to a given cursor — the `state-store/topology` mirror barrier. On a Git-backed replica this proves live remote durability (a fetch + ancestry check against the configured remote), not merely a local commit. See [wait/README.md](wait/README.md).

## Open Questions

- Should there be a `synchestra state info` subcommand to show current sync policy, last pull/push timestamps, and pending local commits?
- Should there be a `synchestra state status` subcommand to show sync health (e.g., "3 local commits unpushed, last pull 2m ago")? `state-store/topology` separately anticipates a `synchestra state status` for topology health (active identity, authority epoch, per-replica cursor/lag) — if/when both are built, reconcile whether they are the same command or two distinct ones before implementing either.
- `state-store/topology` also anticipates `synchestra state verify` and `synchestra state replicate` operator commands (library-level `VerifyConvergence`/`Replicate`/`DrainOutbox` already exist in `pkg/state/replication`; no CLI wiring yet). Scope for a future task.

---
*This document follows the https://specscore.md/feature-specification*
