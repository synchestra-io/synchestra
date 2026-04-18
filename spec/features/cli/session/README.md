# Feature: CLI Session

**Status:** In Progress

## Summary

Command group under the `synchestra` CLI for inspecting and controlling **sessions** — the runtime execution contexts created by [`synchestra runner dispatch`](../runner/dispatch/README.md). Sessions are orthogonal to tasks: a task answers *"where is this work in its SpecScore lifecycle?"*; a session answers *"is the runtime alive, and what is it doing right now?"*

## Problem

Once a user dispatches a plan or task to a runner, they need to observe and sometimes intervene. The existing [`task status`](../task/README.md) command reports SpecScore state (queued, claimed, in_progress, completed), but it cannot answer:

- Is the runtime process still alive on the runner?
- When did the agent last emit output?
- What is in the session log?
- Can I cancel this execution?

Without a session command group, these questions route through the web UI or ad-hoc SSH — incompatible with CLI-first and agent-driven workflows.

## Behavior

### Session concept

A session is the runtime state of a single dispatch. One `runner dispatch` creates one session. The session lives on a runner, carries one agent, executes one plan or task (including its subtasks), and terminates when the dispatched unit reaches a terminal state or the caller stops it.

A session is identified by a stable, hub-issued ID (e.g., `sess_01HXYZ…`).

### Command group structure

```
synchestra session <verb>
```

All verbs require authentication via [`synchestra auth login`](../auth/README.md); unauthenticated calls fail with exit code `101`.

### Sync behaviour

Session verbs read from the Hub and do not touch the state repository, with one exception: `session stop` sets the `abort_requested` flag on the underlying task (per the existing [abort flag semantics](../task/README.md#the-abort_requested-flag)), which follows the normal state-repo sync policy.

## Contents

Each verb is a subfeature. Detailed specs are planned; this README documents the surface.

| Verb | Description |
|---|---|
| `list` | List sessions visible to the authenticated user, with optional runner/status filters |
| `status` | Report current runtime status of a session by ID |
| `logs` | Fetch or follow a session's output stream |
| `stop` | Request termination of a running session |

### list

```
synchestra session list [--runner <name>] [--status running|completed|failed|stopped] [--format text|json]
```

Defaults to the authenticated user's sessions across all runners. Terminated sessions older than the Hub's retention window are omitted unless `--status` selects them explicitly.

### status

```
synchestra session status <session-id> [--format text|json]
```

Reports: runner name, dispatched target, agent, runtime status, start time, last activity, and linked task ID (if applicable).

### logs

```
synchestra session logs <session-id> [--follow] [--since <duration>]
```

Streams the session's combined stdout/stderr. `--follow` blocks and tails new output; it exits 0 when the session reaches a terminal state or on SIGINT. `--since 10m` limits the window.

### stop

```
synchestra session stop <session-id> [--reason <text>]
```

Signals the runner to terminate the session. The command returns exit 0 when the stop request is accepted by the Hub — not when the session actually ends. The underlying task (if any) receives `abort_requested`, giving the agent a chance to clean up.

### Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments |
| `3` | Session not found |
| `4` | Session is not in a stoppable state (already terminal) |
| `90` | Hub unreachable |
| `91` | Session state inconsistent (Hub and runner disagree — details on stderr) |
| `101` | Unauthenticated |

Exit codes `90`–`99` are reserved for the CLI Session feature. This range is registered in the [CLI](../README.md#command-group-reserved-ranges) parent feature.

## Dependencies

- [cli](../README.md) — parent feature
- [cli/runner](../runner/README.md) — `runner dispatch` is the only way sessions are created in MVP
- [cli/auth](../auth/README.md) — authentication required by all verbs
- [task](../task/README.md) — `session stop` interacts with the abort flag on the underlying task
- [runner](../../runner/README.md) — product-level session concept (chat-UI sessions); CLI sessions are a specialization of this for dispatch-initiated executions

## Acceptance Criteria

1. `session list` with no flags returns all sessions the authenticated user has visibility into, ordered by recency.
2. `session status <id>` returns runtime status even when the underlying task is already `completed` — the two are independently queryable.
3. `session logs <id> --follow` exits 0 when the session transitions to a terminal state; SIGINT also exits 0.
4. `session stop <id>` returns 0 on request acceptance; the session's actual termination is observable only via `session status` or `session logs --follow`.
5. Every session verb exits with code 101 when the caller is unauthenticated.

## Outstanding Questions

1. What is the Hub's retention window for terminated sessions? Defer to the [Runner](../../runner/README.md) or Hub API feature.
2. Should `session logs` support pagination for very long outputs (offset/limit), or is `--since` the only slicing primitive in MVP?
3. Should `session stop` accept a list of session IDs for batch cancellation, or is one-at-a-time the MVP contract?
4. Is there a scenario where a CLI-initiated session and a UI chat session need distinct verbs, or are they unified under this feature?
