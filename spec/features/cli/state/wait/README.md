---
format: https://specscore.md/feature-specification
---

# Command: `synchestra state wait`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state/wait?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state/wait?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state/wait?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/state/wait?op=request-change) |
**Parent:** [state](../README.md)
**Environment:** Coordination (Agent, Operator)

## Synopsis

```
synchestra state wait --project <project_id> --replica <replica_id> \
  (--cursor <epoch:sequence> | --source-dir <path>) \
  [--replica-dir <path>] [--remote <name>] [--branch <name>] \
  [--timeout <duration>] [--poll-interval <duration>] [--format text|json]
```

## Description

Implements the mirror barrier from
[`state-store/topology`](../../../state-store/topology/README.md#write-acknowledgement-and-durability):
"a request to wait until selected replicas acknowledge a specified state
change." Blocks until the named replica has durably applied journal events up
to a given cursor, then returns.

On the required Git mirror, satisfying the barrier proves **remote**
durability — a live fetch and ancestry check against the configured Git
remote, never merely a local commit — before reporting success
(`state-store/topology#ac:mirror-barrier-proves-git-durability`,
`state-store/backends/sqlite#ac:git-barrier-proves-portable-durability`). A
timeout never rolls back or hides the active write that already succeeded:
it reports the exact cursor observed plus an explicit "barrier unsatisfied"
outcome.

Unlike `pull`/`push`/`sync` in this command group (currently unimplemented
stubs — see [state/README.md](../README.md)), `wait` is a real, backend-aware
capability. It targets the `state-store/topology` replication journal
directly (via `pkg/state/replication`), not the markdown-file task/chat state
`pull`/`push`/`sync` describe.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `--project` | Yes | Project ID the journal is scoped to. Unlike most `synchestra` commands' [`--project`](../../_args/project.md), this is **not autodetected** — no `synchestra.yaml`/`synchestra-spec-repo.yaml` field currently carries the topology-scoped project ID this journal is keyed on. See Open Questions. |
| `--replica` | Yes | Replica endpoint ID being waited on. Used for reporting and to open the journal; role/epoch fencing is not exercised by this read-only command. |
| `--replica-dir` | No (autodetected) | Path to the replica's local Git state repository. Defaults to the current project's resolved state repo path (same resolution [`pull`](../pull/README.md) uses). |
| `--cursor` | One of `--cursor`/`--source-dir` | Target cursor as `epoch:sequence` (e.g. `1:42`) — typically the cursor a prior terminal command returned. |
| `--source-dir` | One of `--cursor`/`--source-dir` | Resolve the target cursor from a second local Git journal's current head, instead of passing `--cursor` explicitly. |
| `--remote` | No (default `origin`) | Git remote name to prove durability against. |
| `--branch` | No (default `main`) | Git branch to prove durability against. |
| `--timeout` | No (default `30s`) | How long to wait before reporting the barrier unsatisfied. |
| `--poll-interval` | No (default `200ms`) | How often to re-check the replica while waiting. |
| `--format` | No (default `text`) | Output format: `text` (human-readable) or `json` (machine-readable; see Output below). |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — the replica durably reached the target cursor (remote-proven, for a Git-backed replica) |
| `1` | Conflict — the barrier timed out before the replica satisfied the target cursor; the active write itself already succeeded |
| `2` | Invalid arguments — missing `--project`/`--replica`, neither or both of `--cursor`/`--source-dir` given, malformed `--cursor`, or unsupported `--format` |
| `3` | Not found — `--source-dir`'s journal has no events yet, so there is no head cursor to resolve the target from |
| `10` | Unexpected error — journal open failure, Git subprocess failure, etc. |

## Output

**`--format text`** (default): one human-readable line, e.g.:

```
mirror barrier satisfied: replica git-mirror reached cursor 1:42 (durably recorded on the Git remote at commit 4f2c1a9...) in 340ms
```

or, on timeout:

```
mirror barrier unsatisfied: replica git-mirror requested 1:42, observed 1:40 after 30.01s
```

**`--format json`**: one JSON object matching `waitOutput`
(`pkg/cli/state/wait.go`):

```json
{
  "replica": "git-mirror",
  "requested_cursor": "1:42",
  "observed_cursor": "1:42",
  "satisfied": true,
  "remote_proven": true,
  "commit_sha": "4f2c1a9...",
  "elapsed_ms": 340
}
```

## Behaviour

1. Resolve `--replica-dir` (explicit, or autodetected from the current
   directory the same way `pull`/`push`/`sync` do).
2. Open a read-only view of the replica's Git-backed journal at that
   directory (`Head`/`After` only — this command never appends).
3. Resolve the target cursor from `--cursor`, or by reading `--source-dir`'s
   current head.
4. Poll the replica's `Head` cursor at `--poll-interval` until it reaches or
   passes the target, or `--timeout` elapses.
5. Once the target cursor is locally reached, additionally prove Git remote
   durability via a live fetch + ancestry check (`--remote`/`--branch`)
   before reporting success.
6. Print the result (text or JSON) and exit with the corresponding code.

## Open Questions

1. `--project` has no autodetection source today. A future `repo-config`
   increment that declares topology endpoints (project ID, backend, role,
   purpose) in `synchestra.yaml`/`synchestra-spec-repo.yaml` should let this
   command (and its siblings) autodetect `--project` the same way
   [`--project`](../../_args/project.md) works elsewhere.
2. This command only supports Git-backed replicas today (the only backend
   with a physical `Journal` implementation reachable directly from the CLI
   without a running server). A SQLite mirror target (via `sqlitestore`,
   task-4's scope) needs its own `--replica-dir`-equivalent connection
   parameters once that package exists.

---
*This document follows the https://specscore.md/feature-specification*
