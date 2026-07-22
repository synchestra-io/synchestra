# Feature: CLI Runner Dispatch

**Status:** In Progress

## Summary

Dispatches an ad-hoc prompt, plan, or task for remote execution. The Hub records a durable [Dispatch](../../../dispatch/README.md), a matching worker leases an attempt, and each attempt creates one [session](../../session/README.md). The command returns immediately with a dispatch ID; session details appear when a worker starts the attempt.

## Problem

Users need a single, one-shot command to offload work from their local machine to a registered runner. Without it, remote execution requires manual coordination — copying plans, SSHing, managing credentials, babysitting progress — which defeats the purpose of having runners at all.

The command must:

1. Accept whatever the user has on hand — an ad-hoc prompt, a plan file path/name, or a task ID/name.
2. Resolve that input unambiguously against the active project's state repository.
3. Submit durable work to a named runner or a general worker selector.
4. Return a dispatch ID immediately and enough resolved context for the caller to monitor progress.
5. Return quickly; not block on the session.

## Behavior

### Synopsis

```
synchestra runner dispatch [<target>] [--prompt <text>] [--runner <name>] [--profile fast|balanced|large] [--agent <agent>] [--model <selector>] [--format text|json]
```

### Parameters

| Parameter | Required | Description |
|---|---|---|
| [`<target>`](./_args/target.md) (positional) | Conditional | Plan/task target. Omit when `--prompt` is supplied. |
| [`--prompt`](./_args/prompt.md) | Conditional | Ad-hoc work description. Required when no target is supplied. |
| [`--runner`](./_args/runner.md) | No | Preferred registered runner name. When omitted, the scheduler matches any authorized eligible worker. |
| [`--profile`](./_args/profile.md) | No | Provider-neutral execution profile; defaults through routing rules to `balanced`. |
| [`--agent`](./_args/agent.md) | No | Override the runner's default agent. Accepted only when the runner advertises support for multiple agents; otherwise fails with invalid-arguments. |
| [`--model`](./_args/model.md) | No | Exact or adapter-specific model selector such as `sonnet`; overrides profile mapping without changing the recorded requested profile. |
| [`--format`](../../_args/format.md) | No | Output format — `text` (default) or `json`. |

### Target resolution

When supplied, the positional `<target>` argument is resolved in this order:

1. **Path on disk** — if `<target>` is a path to an existing plan file (e.g., `plans/migrate.md`), the file is dispatched as a plan.
2. **Explicit ID** — if `<target>` matches the ID format of a plan or task in the current project's state repository, it is dispatched as that resource.
3. **Fuzzy name match** — otherwise, the CLI queries the project's active plans and tasks and attempts a name match. Exactly one match dispatches it; multiple matches list candidates and fail; zero matches fails.

When `--prompt` is supplied, repository identity and immutable base revision are resolved from the current Git checkout. A prompt and target are mutually exclusive in the MVP. In non-interactive contexts (agent-driven, piped stdin, `--format json`), ambiguity always fails fast — the CLI never prompts.

### Output order

Output is printed in two blocks, **resolution first**, then session details:

```
Resolved:
  target:  plan plans/migrate-to-typescript.md  (plan-id: PLAN-42)
  runner:  hetzner-vm1
  agent:   claude-code  (default for this runner)

Dispatch:
  dispatch-id: dsp_01HXYZ…
  status:      queued
  task-id:     TASK-1024
```

Resolution is printed unconditionally — even when the target is hard-specified by ID — so the caller can verify what was understood before acting on the result. On dispatch failure after resolution, the session block is replaced by an error block; resolution stays visible.

With `--format json`, the same fields are emitted as a single JSON object: `{resolved: {...}, dispatch: {...}}` on success, `{resolved: {...}, error: {...}}` on failure.

### Session lifetime

One dispatch may have multiple attempts over retries. Each attempt creates at most one session. If the dispatched unit is a plan with subtasks, the remote agent handles those subtasks within one attempt unless a future fan-out feature explicitly decomposes it.

### Exit codes

Standard CLI exit codes apply (see [CLI exit code contract](../../README.md#exit-code-contract)). This command uses:

| Exit code | Meaning |
|---|---|
| `0` | Dispatch accepted and queued or leased |
| `1` | Conflict — targeted task was claimed by another agent before the runner could take it |
| `2` | Invalid arguments, ambiguous fuzzy match, or unsupported `--agent` override |
| `3` | Target not found |
| `4` | Target is in a non-dispatchable state (already terminal) |
| `80` | Runner not found or not registered to this user |
| `81` | No eligible runner can satisfy the explicit runner/agent/model constraints |
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

1. Dispatching with an existing plan file path creates a durable dispatch and prints both a resolution block and a dispatch block before exiting 0.
2. Dispatching with a task ID that is currently `queued` transitions the task to `claimed` via the normal task lifecycle; a conflict with another agent returns exit code 1.
3. Dispatching with a name that matches multiple plans or tasks lists the candidates on stderr and exits with code 2 without creating a session.
4. Dispatching to a runner name the caller has not registered exits with code 80; omitting `--runner` permits general scheduler matching.
5. Dispatching without valid user credentials exits with code 101 and prints no session details.
6. With `--format json`, output is a single JSON object with `resolved` always present and either `dispatch` or `error` present — never both.
7. Dispatching with `--prompt` and no target resolves the current Git repository/base revision and creates an ad-hoc dispatch without creating a Task.

## Outstanding Questions

1. Should `--watch` compose `dispatch status` and session logs, or remain outside the MVP?
2. When the runner's default agent is already correct but the user passes `--agent` with the same value, is that a no-op or an error? Leaning no-op.

---
*This document follows the https://specscore.md/feature-specification*
