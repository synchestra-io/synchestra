# --sync

Overrides the project's configured sync policy for a single command invocation.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | — (use configured policy) |

## Supported by

All `synchestra` commands that interact with the state repository (task mutations, task reads, project queries, etc.).
Commands that never touch the state repository (e.g., `config show`, `spec lint`) ignore this flag.

## Values

| Value | Behaviour |
|---|---|
| `remote` | Force immediate pull before read and push after write, regardless of the project's configured [sync policy](../../state-store/backends/git/README.md#sync-policy). Equivalent to behaving as if both pull and push policies are `on_commit` for this invocation only. |
| `local` | Suppress all remote I/O for this invocation. Reads operate on the local working tree; writes commit and merge locally but do not push. Equivalent to behaving as if both pull and push policies are `manual` for this invocation only. |

## Description

The state store's [SyncConfig](../../state-store/README.md#construction) determines when the CLI automatically pulls from and pushes to the remote. The `--sync` flag lets a user or orchestrator override that configuration for a single command without changing the project's persistent settings.

When omitted, the command follows the project's configured sync policy as normal.

### When to use `--sync remote`

- An orchestrator needs a contended operation (e.g., `task claim`) to be immediately visible to agents on other hosts, but the project uses a deferred-push policy like `manual` or `on_interval`.
- A human operator wants to publish local state changes on demand rather than waiting for the next sync cycle.
- Debugging or auditing requires the freshest remote state before reading.

### When to use `--sync local`

- Running a batch of read commands locally where network latency is unacceptable.
- The remote is temporarily unreachable and the caller accepts stale data.
- Testing or scripting against a local clone without affecting the remote.

### Interaction with conflict detection

When `--sync remote` is used with a mutation command, the full conflict-detection protocol applies: pull → validate → commit → merge → push → retry on conflict. When `--sync local` is used with a mutation command, push is skipped and conflict detection is limited to the local host. The caller assumes responsibility for eventual sync and conflict resolution (e.g., via `synchestra state push`).

## Examples

```bash
# Force a claim to sync with remote even though project uses manual policy
synchestra task claim --project acme --task fix-bug --run 42 --model sonnet --sync remote

# Read task list from local state without hitting the network
synchestra task list --project acme --sync local

# Force push after completing a task under on_session_end policy
synchestra task complete --project acme --task fix-bug --sync remote
```

## Outstanding Questions

None at this time.
