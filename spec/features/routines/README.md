# Feature: Routines

**Status:** draft

## Summary

Cross-platform scheduled and triggered agent workflows. A routine is a spec-native, runtime-portable unit of recurring or event-driven work that runs on a chosen runner and produces reviewable git artifacts.

## Problem

AI agent runtimes are converging on a "recurring task" primitive — Claude Code has [Routines](https://code.claude.com/docs/en/routines), GitHub Copilot has background agents, Cursor has background tasks. Each is scoped to a single runtime and a single execution environment. Teams that coordinate work across multiple runtimes and multiple compute targets hit three walls:

1. **Runtime lock-in.** A routine authored for Claude Code cannot be executed by Copilot CLI without rewriting the configuration, secrets, and output contract.
2. **Compute lock-in.** Vendor-hosted routines run on the vendor's infrastructure. Teams with compliance, data-residency, or cost constraints need the same routine to run on a personal laptop, an in-house VM, or a self-hosted cloud environment (Azure, GCP, AWS) interchangeably.
3. **Coordination blindness.** Routines scheduled in external tools do not share state with the project's work graph. They produce side effects (commits, issues, messages) without appearing as first-class nodes in the task tree — making it impossible to see, from one place, what is running, what ran, and why.

Synchestra already models compute endpoints ([runner](../runner/README.md)), pre/post workflow chains ([micro-tasks](../micro-tasks/README.md)), and reusable agent invocations ([agent-skills](../agent-skills/README.md)). What is missing is the envelope that binds them: a trigger, a runtime choice, and a runner target, captured as a versioned spec.

## Behavior

### Routine Definition

A routine is a spec directory at `spec/routines/<slug>/` with a `README.md` describing intent and YAML frontmatter describing execution. A routine has four declarative components:

- **Trigger** — when the routine should fire.
- **Runtime** — which agent runtime executes the body.
- **Runner** — which registered [runner](../runner/README.md) the runtime runs on.
- **Body** — the prompt, skill reference, or task reference to execute, optionally wrapped in a [micro-tasks](../micro-tasks/README.md) chain.

```yaml
---
trigger:
  - cron: "0 2 * * *"          # nightly at 02:00 local to runner
  - manual: true               # also invocable on demand via CLI/API
runtime: claude-code-headless  # or: copilot-cli, anthropic-api, openai-api, ...
runner: local                  # any registered runner name
body:
  skill: triage-stale-prs      # ref an agent-skill, a task, or inline prompt
  params:
    max_age_days: 14
artifacts:
  commit_to: branch:routines/triage-stale-prs
  open_pr: true                # HITL by default — result is reviewable
---
```

### Trigger Types

Four trigger types are supported at v1. Multiple triggers may be combined; any one firing invokes the routine.

| Trigger | Fires When | Notes |
|---|---|---|
| `cron` | A cron expression matches the runner's clock | Standard 5-field cron. Timezone defaults to the runner's timezone; overridable per trigger. |
| `git-event` | A repository event occurs (push, PR opened, issue labeled, file changed) | Requires a source of events — local daemon polling or webhook ingestion via [api](../api/README.md). |
| `task-state` | A [task](../cli/task/README.md) transitions to a named status | Example: `task-state: { status: queued, label: routine-dispatch }`. Lets routines react to the work graph. |
| `manual` | Invoked via `synchestra routine run <name>` or API | Makes the routine a reusable callable unit independent of schedule. |

### Runtime Adapters

A routine's `runtime` field names an adapter that knows how to invoke a specific agent runtime in headless mode, pass parameters, and capture output. Each adapter is a thin wrapper around an external tool or API.

v1 adapters:
- `claude-code-headless` — Claude Code in non-interactive mode.
- One additional adapter (`copilot-cli` or a raw `anthropic-api`/`openai-api` adapter) — chosen during implementation to prove runtime portability.

Future adapters (out of scope for v1): `cursor-background`, `aider`, `opencode`, custom shell scripts.

### Runner Targeting

A routine targets a registered [runner](../runner/README.md) by name. At v1, only `runner: local` is required — the local daemon acts as the runner. As the runner feature matures to support remote VMs and cloud targets (Azure/GCP/AWS/Synchestra Cloud), the same routine spec targets any of them by changing one field. No routine rewrite is needed to move compute.

### Execution Flow

```mermaid
sequenceDiagram
    participant T as Trigger Source
    participant D as Synchestra Daemon
    participant R as Runner
    participant A as Runtime Adapter
    participant G as Git (state repo)
    T->>D: Fire (cron tick / event / task state / manual)
    D->>D: Resolve routine spec & parameters
    D->>R: Dispatch (runtime, body, params, secrets)
    R->>A: Invoke adapter with body
    A->>A: Execute agent runtime headlessly
    A-->>R: Output (transcript, artifacts)
    R-->>D: Result
    D->>G: Commit transcript + artifact deltas
    D->>D: Open PR / update task / notify (per artifacts config)
```

### Human-in-the-Loop by Default

Every routine run produces a reviewable git artifact — a branch, a PR, or a task update. Routines never silently mutate main branches or external state. A routine that intends to auto-merge must declare `artifacts.auto_merge: true` explicitly; the default is a human review gate.

### Spec-Native Coordination

Because routines live in the spec tree and commit through Synchestra's state repository, each run is:
- Attributable (signed/co-authored commits via [host-auth](../host-auth/README.md)).
- Queryable (routine runs appear alongside tasks in the work graph).
- Composable (a routine body may reference a [task](../cli/task/README.md), an [agent-skill](../agent-skills/README.md), or a [micro-tasks](../micro-tasks/README.md) chain).

### Routine Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Defined: spec created
    Defined --> Enabled: trigger registered
    Enabled --> Running: trigger fires
    Running --> Completed: adapter succeeds
    Running --> Failed: adapter errors
    Completed --> Enabled: next trigger
    Failed --> Enabled: next trigger (or quarantined after N failures)
    Enabled --> Disabled: synchestra routine disable
    Disabled --> Enabled: synchestra routine enable
    Enabled --> [*]: spec removed
    Disabled --> [*]: spec removed
```

## Dependencies

- [runner](../runner/README.md) — Routines target registered runners for execution; runner feature owns compute endpoints and auth.
- [micro-tasks](../micro-tasks/README.md) — A routine body may be wrapped in a pre/post/background micro-task chain.
- [agent-skills](../agent-skills/README.md) — A routine body may reference a skill as its unit of work.
- [cli](../cli/README.md) — `synchestra routine new|run|list|enable|disable|logs` commands surface routine operations.

## Acceptance Criteria

1. A user can scaffold a routine with `synchestra routine new --title <name>` producing a valid spec directory under `spec/routines/`.
2. The Synchestra daemon fires routines with `cron` and `manual` triggers on the local runner.
3. At least two runtime adapters are implemented and routable by the `runtime` field.
4. A routine run commits its transcript and any artifact deltas to the state repository (or a designated branch in the code repo), never to `main` by default.
5. `synchestra routine list` shows defined routines, their triggers, last run status, and next scheduled fire.
6. Switching a routine's `runner` field from `local` to any other registered runner requires no other spec changes.
7. Changing a routine's `runtime` field between supported adapters requires no other spec changes (beyond adapter-specific `params`).
8. A routine spec validates via `synchestra spec lint`.

## Outstanding Questions

1. Should routines be a top-level directory (`spec/routines/`) or a sub-feature (`spec/features/routines/`) of the feature tree? This spec assumes `spec/routines/` because routines are instances, not features — but the decision affects CLI command shape and navigation.
2. What is the secret-injection model for routines on remote runners — inherited from the runner's credential store, declared per-routine, or both? Coordinate with [host-auth](../host-auth/README.md).
3. How should `git-event` triggers be sourced when the user has no webhook endpoint — daemon polling, GitHub App installation ([github-app](../github-app/README.md)), or both?
4. When a routine body references a task and the task is already in-flight (claimed by another agent), does the routine no-op, queue, or fail?
5. Should the daemon quarantine a routine after N consecutive failures, and if so what is the default N and reset policy?
6. Do we need a `routine` status primitive distinct from `task` status, or can routine runs be modeled as tasks with a synthetic parent? Relates to the deferred Direction C (routines-as-tasks) evolution.
7. Should routine definitions support parameter sets / matrices (fan-out a single routine over a list of values, e.g., per-repo in a multi-repo fleet)?
