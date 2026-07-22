---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Dispatch Worker

**Status:** Approved
**Source Ideas:** [Remote Dispatch](https://github.com/synchestra-io/synchestra-marketing/blob/main/ideas/remote-dispatch.md)

## Summary

The Dispatch Worker is a role of the existing `synchestra-host`. It advertises agent/model capabilities, claims eligible dispatch leases through an outbound connection, prepares an isolated repository worktree, launches the existing agent/session runtime, validates and publishes a branch, and reports durable results.

## Problem

The VM host already registers, heartbeats, provisions containers, and starts sessions, but it does not pull scheduled work or complete a repository change lifecycle. Repository initialization and worktree paths also disagree. A new daemon would duplicate existing host responsibilities rather than fix the missing coordination.

## Behavior

### Worker loop

On startup the host authenticates using existing registration credentials, advertises its capabilities, and repeatedly performs a bounded long-poll claim. It runs no more attempts than its configured concurrency. For every active attempt it heartbeats the lease, observes cancellation, and emits terminal state exactly once.

### Workspace preparation

The worker:

1. Resolves the repository URL and immutable base revision from the dispatch snapshot.
2. Clones or fetches a persistent repository cache using scoped credentials.
3. Creates an isolated worktree and `synchestra/<dispatch-id>` branch from the base revision.
4. Verifies the agent process starts in that worktree.
5. Removes or retains the worktree according to explicit cleanup policy after terminal reporting.

The canonical repository root is passed explicitly through provisioning and session start; code must not infer two incompatible layouts.

### Agent execution

The worker passes the resolved agent, model, effort, prompt/target context, repository instructions, and scoped environment to the session runtime. The initial production adapter is Claude Code. Adapter interfaces remain provider-neutral and fail clearly when a requested selector is unavailable.

### Finalization

On successful agent exit, the worker runs or confirms validation, refuses unrelated or secret-bearing changes, creates a commit, pushes the task branch, and reports result metadata. A failed validation or push records failure with actionable logs. The worker never merges or deploys.

### VM proof service

For the personal proof, the existing XDG-installed host runs as an `ai` user service because that user owns the installation and existing Git/Claude credentials. The unit follows the `current` symlink so a release promotion is an atomic symlink update. A dedicated system user and `/opt/synchestra` installation are deferred.

## Acceptance Criteria

### AC: outbound-claim-loop

**Given** a connected registered host with available capacity
**When** an eligible dispatch is queued
**Then** the host claims it through the outbound scheduler API without requiring an inbound public endpoint.

### AC: repository-root-is-worktree

**Given** a dispatch for a Git repository
**When** the session starts
**Then** its working directory is an isolated worktree of the requested repository and base revision, not an empty project directory.

### AC: resolved-model-reaches-agent

**Given** an attempt resolved to a concrete agent, model, and effort
**When** the host starts the session
**Then** those exact values reach the adapter invocation and appear in attempt metadata.

### AC: heartbeat-protects-ownership

**Given** a running attempt
**When** its heartbeat succeeds
**Then** its lease remains active; when heartbeat reports cancellation or ownership loss, the worker stops without publishing success.

### AC: branch-only-publication

**Given** a successful change and validation
**When** the worker finalizes
**Then** it commits and pushes only the dedicated dispatch branch and reports its commit without merging or deploying.

### AC: failed-run-is-durable

**Given** agent execution, validation, or Git publication fails
**When** the worker finalizes
**Then** it reports a terminal failed attempt with stage, reason, and log reference, and does not leave the dispatch falsely running.

### AC: service-survives-restart

**Given** the personal VM restarts
**When** the `ai` user service starts again
**Then** the host reconnects with its existing registration, resumes claiming, and does not resurrect expired ownership.

## Dependencies

- [dispatch](../README.md) — parent lifecycle and publication contract
- [scheduler](../scheduler/README.md) — claim, lease, heartbeat, cancellation, and result API
- [runner](../../runner/README.md) — host/runner capabilities and concurrency
- [sandbox](../../sandbox/README.md) — container and session execution
- [channels](../../channels/README.md) — logs and session transport

## Outstanding Questions

1. Repository cache retention and disk limits should be measured on the personal VM before choosing automatic garbage collection defaults.

## Implementation Evidence

[`synchestra-vm` PR #5](https://github.com/synchestra-io/synchestra-vm/pull/5) implements the outbound claim loop, bounded backoff, capability advertisement, concurrency, heartbeat/cancellation/ownership-loss handling, graceful shutdown, token refresh, and the repository-executor adapter. Environment checkpoint `f307c149d6ea45db36b5964df99bc3a3d7a6cf20` prevents worker registration and arbitrary host state from crossing into child processes. Contract reconciliation `fa325acc9161eeee5e469b9a116d03775dcee299` pins Synchestra `a14007d...` and executor `cf7ae4a...`.

Integration checkpoint `84a8e6de275713baeabdb9d3c5b0eb76e17da923` passes the real local Hub -> two worker processes -> default executor/fake Claude -> Git branch flow, including heartbeat, cancellation, SIGKILL/lease recovery, retry history, durable logs, exact model arguments, and caller-checkout immutability. User-service restart on the registered personal VM is not yet certified because the required external control machine is unavailable in the current environment.

---
*This document follows the https://specscore.md/feature-specification*
