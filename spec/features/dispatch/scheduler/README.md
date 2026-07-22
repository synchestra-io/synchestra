---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Dispatch Scheduler

**Status:** Approved
**Source Ideas:** [Remote Dispatch](https://github.com/synchestra-io/synchestra-marketing/blob/main/ideas/remote-dispatch.md)

## Summary

The Dispatch Scheduler is the Hub-side durable queue and matching engine. It stores dispatches and attempts, selects work eligible for a registered worker, grants expiring leases transactionally, and records heartbeat, retry, cancellation, and terminal results.

## Problem

The existing cloud message queue redelivers chat messages but cannot safely schedule work. A scheduler needs durable intent, idempotent submission, worker constraints, future start time, priority, exactly one active lease, attempt history, and recovery after worker failure.

## Behavior

### Records

A Dispatch contains source snapshot, prompt/target, requested execution profile, worker constraints, priority, optional `not_before`, aggregate status, idempotency key, creator, and timestamps. An Attempt contains lease owner and expiry, session ID, resolved execution configuration, heartbeat, result, and terminal reason.

### API

The initial contract provides operations equivalent to:

- Create and inspect a dispatch.
- List dispatches visible to the authenticated user.
- Cancel or explicitly retry a dispatch.
- Claim the next eligible dispatch using registered host identity and advertised capabilities.
- Heartbeat, start, complete, fail, or acknowledge cancellation for the active attempt.
- Read attempt logs/result metadata through the existing session/message surface or stable links to it.

Exact route spelling is implementation-owned, but user and worker authentication domains remain separate.

### Matching

A worker can claim a dispatch only when:

- the worker is registered, authorized, online, and not draining;
- `not_before` has passed;
- repository/project access matches;
- requested agent/profile/capabilities are advertised;
- concurrency and quota allow another attempt; and
- no unexpired attempt currently owns the dispatch.

The MVP sorts eligible work by priority, creation time, and stable ID. More advanced fairness is deferred.

### Lease correctness

Lease assignment uses a Firestore transaction or equivalent compare-and-set. Claim responses are idempotent for the same worker request. Heartbeats extend only the caller's active lease. A stale owner cannot complete, fail, or publish a result after expiry or reassignment.

## Acceptance Criteria

### AC: idempotent-create

**Given** a caller retries dispatch creation with the same idempotency key and equivalent payload
**When** the API receives the retry
**Then** it returns the original dispatch and creates no duplicate work.

### AC: transactional-single-lease

**Given** concurrent eligible claim requests
**When** they contend for one queued dispatch
**Then** exactly one attempt receives an active lease.

### AC: capability-filtering

**Given** queued dispatches with different agent, profile, project, or runner constraints
**When** a worker claims work
**Then** it receives only a dispatch matching its authenticated identity and advertised capabilities.

### AC: not-before-honored

**Given** a dispatch scheduled for a future `not_before`
**When** an eligible worker claims before that time
**Then** the dispatch is not returned; after that time it becomes eligible.

### AC: stale-owner-rejected

**Given** an attempt lease expired or was reassigned
**When** the former worker reports heartbeat or completion
**Then** the scheduler rejects the mutation and preserves the current owner/result.

### AC: abandoned-attempt-recoverable

**Given** an attempt stops heartbeating
**When** its lease expires
**Then** it is recorded as abandoned and retry policy can make the dispatch claimable again.

### AC: cancellation-visible-to-worker

**Given** a leased or running attempt
**When** the user cancels the dispatch
**Then** the next heartbeat response tells the owning worker to stop and no subsequent success is accepted.

## Dependencies

- [dispatch](../README.md) — parent lifecycle and result contract
- [runner](../../runner/README.md) — identities, ACLs, capabilities, status, and quotas
- [host-auth](../../host-auth/README.md) — authenticated worker mutations
- [channels](../../channels/README.md) — session/message logs

## Outstanding Questions

1. Whether logs are embedded under attempts or referenced exclusively through Sessions should be settled during contract implementation; no log text should be duplicated in two stores.

## Implementation Evidence

[`synchestra-cloud` PR #1](https://github.com/synchestra-io/synchestra-cloud/pull/1) implements Firestore-backed idempotent creation, capability matching, transactional claims, owner-bound mutations, heartbeats, cancellation, retry/history, expiry recovery, and terminal session/log/result persistence. Reconciliation commit `96b7b7e65c70ab1bde982635a4a867264f190da5` pins the canonical v1 contract and `ordered-capabilities-v1` model mapping. Firestore emulator concurrency, lost-response replay, stale-owner rejection, expiry, cancellation, unsupported-selector, and durable-restart tests pass. The integration-only Hub helper is published at `ba9c4bc2eed0f4d031c1da258e128fc85405d344`; it binds loopback only and production deploy jobs remain skipped. Hardening checkpoint `1acc52f896f453f42bcdfe44967f01d3ab3346d8` adds bounded lifecycle and lease-expiry observations, persistence/rollback failure injection, redacted retryable infrastructure failures, indefinite MVP retention defaults, and common diagnosis without changing the v1 protocol.

---
*This document follows the https://specscore.md/feature-specification*
