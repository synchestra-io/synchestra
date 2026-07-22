# ADR-0006: Synchestra owns queued remote dispatch

**Status:** Accepted
**Date:** 2026-07-22

## Context

Synchestra already has Git-backed tasks, registered hosts, runner/session APIs, agent processes, model fields, credentials, and container execution. SpecStudio separately implements a single-project specification lifecycle and within-session subagent orchestration. A requested user experience makes the missing boundary concrete:

```text
/dispatch @sonnet Update dal-go deps to latest
```

The command must work for an ad-hoc repository prompt, use a general scheduler, and be proven on an existing registered VM. Placing it in SpecStudio would require a SpecScore lifecycle artifact for a general execution operation and would contradict SpecStudio's explicit exclusion of remote, multi-machine, and across-run scheduling.

The existing runtime also contains multiple execution paths. Adding an SSH-only shortcut or another worker binary would produce a fourth path rather than completing the existing control-plane-to-worker design.

## Decision

1. **Synchestra owns general dispatch.** Queueing, scheduling, matching, leases, attempts, retries, cancellation, and results are Synchestra capabilities. SpecStudio may optionally delegate approved Plan tasks to them but is not a required dependency.
2. **The Synchestra plugin owns the human command.** `ai-plugin-synchestra` provides a visible `/dispatch` command that delegates to the deterministic Synchestra CLI. The existing resource-level skill structure remains intact: the command invokes the runner dispatch workflow rather than duplicating queue logic in prose.
3. **Hub is the durable scheduler.** Dispatch records and attempts live in the Hub control plane. Workers authenticate using the existing host identity and claim eligible leases through outbound long-polling. A VM does not require a public inbound endpoint.
4. **The existing host is the worker.** `synchestra-host` gains the lease loop and coordinates the existing `synchestra-agent`/container session machinery. No additional worker executable is introduced.
5. **Task, Dispatch, and Session are separate.** Task expresses desired work, Dispatch expresses operational scheduling and attempts, and Session is one agent process/transcript. An ad-hoc Dispatch need not create a Task; a Task may have multiple attempts and sessions.
6. **Both ad-hoc and SpecScore targets are supported.** The CLI resolves repository context for raw prompts and may also bind to a SpecScore Plan or Task. This supersedes the earlier MVP idea's deferral of raw prompts.
7. **Execution profiles are stable and provider-neutral.** `fast`, `balanced`, and `large` are the user-level profiles. Provider selectors such as `@sonnet` may request a concrete mapping. Requested selector, resolved agent, resolved model, effort, mapping version, and routing reason are persisted.
8. **The first delivery result is a branch.** The worker creates an isolated worktree and task branch, validates the change, commits, pushes, and reports the branch and commit. It never merges or deploys automatically.
9. **The existing personal VM is the first acceptance worker.** The registered user-scoped `synchestra-host` installation is activated under the `ai` account for the proof. System-wide packaging, managed pools, and additional runtimes follow the stable protocol later.

## Consequences

**Easier**

- The first proof reuses host registration, authentication, sessions, credentials, Docker, Claude Code, and Hub persistence.
- `/dispatch` works without requiring a pre-existing SpecScore artifact.
- SpecStudio remains standalone and can integrate through a narrow optional adapter.
- Multiple workers and runtimes can share one queue and lease contract.
- Per-task model selection is auditable and does not pin the scheduler to a vendor.

**Harder**

- Cross-repository protocol compatibility must be tested explicitly rather than relying on repository-local fakes.
- The current session, repository initialization, and worktree paths must be reconciled.
- Worker crash recovery and lease expiry become correctness requirements.
- The personal proof can reuse user credentials, but broader use requires short-lived, scoped Git and model credentials.

## Alternatives considered

1. **Put `/dispatch` in SpecStudio.** Rejected because general ad-hoc execution is not a specification-lifecycle phase and SpecStudio assigns remote/across-run orchestration to a separate layer.
2. **Use SSH as the queue transport.** Rejected as a useful diagnostic shortcut but not a general scheduler. It bypasses runner identity, durable state, retries, cancellation, logs, and multi-worker matching.
3. **Have workers poll Git task state directly.** Rejected for the MVP because repository discovery, credentials, contention, leases, retries, and ad-hoc prompts do not fit cleanly into the task board. Git remains task truth; Hub owns operational leases.
4. **Use sessions as queue records.** Rejected because one task can have retries and multiple attempts. A session is an execution instance, not a durable scheduling request.
5. **Build a new worker daemon.** Rejected because `synchestra-host` already owns registration, heartbeat, provisioning, and agent coordination.
6. **Start with Cloud Run or multiple providers.** Rejected because the registered long-lived VM is the shortest real proof and establishes the protocol other runtimes can later implement.
