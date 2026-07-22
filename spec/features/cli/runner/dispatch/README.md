# Feature: CLI Runner Dispatch

**Status:** In Progress

## Summary

Dispatches a plan or task to a named runner for remote execution. One dispatch creates exactly one [session](../../session/README.md); the runner executes the dispatched unit within that session and reports status back through the existing state sync.

## Problem

Users need a single, one-shot command to offload work from their local machine to a registered runner. Without it, remote execution requires manual coordination — copying plans, SSHing, managing credentials, babysitting progress — which defeats the purpose of having runners at all.

The command must:

1. Accept whatever the user has on hand — a plan file path, a plan name, a task ID, or a task name.
2. Resolve that input unambiguously against the active project's state repository.
3. Tell a specific runner (by name) to execute it.
4. Return enough identifiers (session ID, task ID) for the caller to monitor progress.
5. Return quickly; not block on the session.

## Behavior

### Synopsis

```
synchestra runner dispatch <target> --runner <name> [--agent <agent>] [--format text|json]
```

### Parameters

| Parameter | Required | Description |
|---|---|---|
| `<target>` (positional) | Yes | What to dispatch — resolved per [Target resolution](#target-resolution). |
| `--runner` | Yes | Registered runner name. |
| `--agent` | No | Override the runner's default agent. Accepted only when the runner advertises support for multiple agents; otherwise fails with invalid-arguments. |
| [`--format`](../../_args/format.md) | No | Output format — `text` (default) or `json`. |

### Target resolution

The positional `<target>` argument is resolved in this order:

1. **Path on disk** — if `<target>` is a path to an existing plan file (e.g., `plans/migrate.md`), the file is dispatched as a plan.
2. **Explicit ID** — if `<target>` matches the ID format of a plan or task in the current project's state repository, it is dispatched as that resource.
3. **Fuzzy name match** — otherwise, the CLI queries the project's active plans and tasks and attempts a name match. Exactly one match dispatches it; multiple matches list candidates and fail; zero matches fails.

In non-interactive contexts (agent-driven, piped stdin, `--format json`), ambiguity always fails fast — the CLI never prompts.

### Output order

Output is printed in two blocks, **resolution first**, then session details:

```
Resolved:
  target:  plan plans/migrate-to-typescript.md  (plan-id: PLAN-42)
  runner:  hetzner-vm1
  agent:   claude-code  (default for this runner)

Session:
  session-id:  sess_01HXYZ…
  task-id:     TASK-1024
  started-at:  2026-04-18T10:22:14Z
```

Resolution is printed unconditionally — even when the target is hard-specified by ID — so the caller can verify what was understood before acting on the result. On dispatch failure after resolution, the session block is replaced by an error block; resolution stays visible.

With `--format json`, the same fields are emitted as a single JSON object: `{resolved: {...}, session: {...}}` on success, `{resolved: {...}, error: {...}}` on failure.

### Session lifetime

One dispatch creates one session. If the dispatched unit is a plan with subtasks, the remote agent handles those subtasks within the same session — the decomposition is opaque to the CLI. Fan-out across multiple runners requires multiple dispatch calls.

### Exit codes

Standard CLI exit codes apply (see [CLI exit code contract](../../README.md#exit-code-contract)). This command uses:

| Exit code | Meaning |
|---|---|
| `0` | Dispatched successfully; session started |
| `1` | Conflict — targeted task was claimed by another agent before the runner could take it |
| `2` | Invalid arguments, ambiguous fuzzy match, or unsupported `--agent` override |
| `3` | Target not found |
| `4` | Target is in a non-dispatchable state (already terminal) |
| `80` | Runner not found or not registered to this user |
| `81` | Runner rejected the dispatch (capacity, incompatible agent, or Hub-side error) |
| `101` | Unauthenticated — run `synchestra auth login` first |

Exit codes `80`–`89` are reserved for the CLI Runner feature; `101`–`109` for CLI Auth. These ranges are registered in the [CLI](../../README.md#command-group-reserved-ranges) parent feature.

## Dependencies

- [cli/runner](../README.md) — parent feature; defines runner identity and sync behaviour
- [cli/session](../../session/README.md) — every successful dispatch produces a session observable via this feature
- [cli/auth](../../auth/README.md) — unauthenticated dispatch fails with `101`
- [runner](../../../runner/README.md) — product-level runner lifecycle and registration
- [plan](https://github.com/specscore/specscore/blob/main/spec/features/plan/README.md) — SpecScore plan format; plans are one valid target
- [task](../../task/README.md) — tasks are the other valid target; dispatching a task triggers a claim

## Acceptance Criteria

1. Dispatching with an existing plan file path creates a session and prints both a resolution block and a session block before exiting 0.
2. Dispatching with a task ID that is currently `queued` transitions the task to `claimed` via the normal task lifecycle; a conflict with another agent returns exit code 1.
3. Dispatching with a name that matches multiple plans or tasks lists the candidates on stderr and exits with code 2 without creating a session.
4. Dispatching to a runner name the caller has not registered exits with code 80.
5. Dispatching without valid user credentials exits with code 101 and prints no session details.
6. With `--format json`, output is a single JSON object with `resolved` always present and either `session` or `error` present — never both.

## Outstanding Questions

1. Should `--watch` be added for synchronous UX, or is `session logs --follow` sufficient composition?
2. Should the Hub queue dispatches when the runner is at capacity, or reject them (current spec: reject with 81)?
3. When the runner's default agent is already correct but the user passes `--agent` with the same value, is that a no-op or an error? Leaning no-op.

---
*This document follows the https://specscore.md/feature-specification*
