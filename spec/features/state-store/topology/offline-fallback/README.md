---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Git Offline Fallback

**Status:** Approved
**Parent:** [Authoritative Store and Replicas](../)

## Summary

The required Git state repository keeps agent-to-agent communication and
recoverable coordination available when the Synchestra server cannot be
reached. Agents immediately exchange immutable fallback envelopes through Git.
If authoritative task, lease, or assignment mutations must continue, Git is
promoted through the normal authority-epoch protocol rather than becoming a
second active store implicitly.

The Synchestra server continuously reconciles the Git state repository while
running and on every restart. Git-provider webhooks accelerate reconciliation;
they are hints, never the source of truth.

## Problem

Centralized low-latency coordination is useful only if a server outage does not
erase the information agents need to understand one another's work. Agents
must still be able to announce changed refs, exchange targeted messages,
acknowledge delivery, publish checkpoints, and leave enough recovery evidence
for another agent to resume.

Allowing agents to mutate a Git mirror as if it were authoritative while a
SQLite server might still accept writes would create split brain. Conversely,
requiring the server for every fallback message defeats the purpose. The
fallback therefore separates append-only communication ingress from
authoritative domain transitions.

## Fallback Modes

### Communication fallback

Communication fallback is always available when Git is reachable. Agents may
append immutable envelopes to the Git fallback inbox without changing the
active-store role. Supported v1 envelope kinds are:

- `message.sent` and `message.acknowledged`;
- `worklog.checkpointed` and `worklog.handoff_requested`;
- `repository.ref.updated`;
- `agent.attention_requested`;
- `server.unreachable_observed`.

These envelopes are durable ingress and audit records. They do not directly
change task status, lease ownership, assignment ownership, or authority epoch.
Agents may read and act on messages addressed to them, but a fallback envelope
claiming that a task changed is not authoritative evidence of that change.

### Authority fallback

Task claims, lease renewal, assignment changes, and lifecycle transitions may
continue through Git only after Git is safely promoted to active. Promotion
uses the parent Feature's authority epoch and fencing protocol.

Two policies are supported:

| Policy | Server-down behavior |
|---|---|
| `communication-only` | Git inbox remains writable; authoritative mutations wait for the server or an authorized manual promotion. |
| `leased-failover` | The server maintains a time-bounded authority lease in Git and stops accepting writes before it can no longer prove the lease. After expiry, an authorized agent may atomically promote Git. |

`communication-only` is safe without timing assumptions. `leased-failover`
enables bounded automatic recovery but intentionally makes continued server
writes depend on renewal of the Git witness lease.

## Fallback Envelope

Each envelope is one inGitDB record with a deterministic schema and immutable
ID. A representative message is:

```json
{
  "schema": "https://synchestra.io/schemas/fallback-envelope/v1",
  "id": "01J...",
  "project_id": "github.com/acme/service",
  "kind": "message.sent",
  "created_at": "2026-08-10T11:15:00Z",
  "sender": {
    "run_id": "run_01J...",
    "agent": "codex",
    "worklog_ref": "worklog://effort_01J.../run_01J..."
  },
  "recipients": ["run_01K..."],
  "thread_id": "thread_01J...",
  "correlation_id": "01J...",
  "payload": {},
  "payload_sha256": "...",
  "signature": "..."
}
```

The collection uses one record file per envelope so concurrent senders normally
touch disjoint paths. IDs are generated before retry and are idempotency keys.
Payloads obey the private-state repository's access policy and retention rules;
credentials and provider secrets are forbidden.

## Agent Behavior

### REQ: deterministic-fallback-selection

The CLI first uses the configured active endpoint with a bounded timeout. On a
transport failure, it reads the topology and fallback policy from locally
cached verified configuration and the Git state repository. It MUST NOT fall
back because a domain request was rejected, conflicted, or returned an invalid
transition; those are valid active-store results, not outages.

### REQ: append-and-fetch

In communication fallback, the CLI fetches the Git state ref, appends its
envelope through the inGitDB DALgo adapter, commits, and pushes with
expected-base protection. A push conflict triggers fetch/reapply/retry because
envelope paths are immutable. The command reports the Git commit SHA as durable
evidence.

Agents poll/fetch at the configured fallback interval and before handoff,
publish, finalize, or cleanup. A received message is acknowledged with a new
immutable envelope; no sender edits the original message.

### REQ: worklog-outbox

If neither the server nor Git is reachable, the agent retains the envelope in
the local Work Log outbox. It may continue private work that does not require a
new authoritative claim, but MUST NOT claim shared resources or represent an
unconfirmed message as delivered.

## Server Reconciliation

The server watches two logical streams in the Git repository:

1. the authoritative/mirrored state-change journal;
2. the fallback inbox of externally appended envelopes.

It MUST distinguish its own mirror commits from fallback ingress, validate
schema, signatures, project scope, and referenced run identities, then import
each envelope idempotently into the active store. Imported envelopes retain
their original ID, Git commit SHA, and timestamps. The server emits normal
delivery notifications and acknowledgements without rewriting the envelope.

Reconciliation runs:

- at server startup before it reports the project healthy;
- after local filesystem notifications for a checked-out state repository;
- after a verified Git-provider webhook;
- on a configurable periodic fetch, even when notifications appear healthy;
- on explicit `synchestra state reconcile`.

Missed or duplicated wake-ups therefore affect latency, not correctness.

## GitHub App Integration

The Synchestra GitHub App SHOULD subscribe to push events for registered state
repositories. The webhook receiver:

1. verifies the GitHub signature and installation/repository scope;
2. records the delivery ID for deduplication;
3. normalizes the push into a repository-ref hint;
4. wakes the relevant reconciliation worker;
5. fetches and verifies the referenced commit before importing any state.

Webhook bodies are not applied as database mutations. Push events can be
duplicated, delayed, reordered, or absent, and the receiving service may be
down. The Git repository plus startup/periodic reconciliation remain the
durable truth. A separately hosted webhook receiver may retain delivery hints
while a project server is down, but this is an optimization rather than a
requirement for recovery.

## Safe Git Promotion

Under `leased-failover`, the current server active periodically renews an
authority lease in Git containing project ID, active endpoint ID, authority
epoch, expiry, and a fencing token. It stops accepting authoritative writes
before local time reaches the lease expiry minus the configured safety margin
if renewal cannot be confirmed.

After the lease expires, an authorized agent may propose a higher epoch by an
expected-base Git commit/push. Exactly one competing promotion wins. The winner
replays any fallback envelopes needed for state, verifies checksums, and starts
Git-active operations. Other agents fetch the new epoch and route writes to
Git.

A restarted SQLite server MUST fetch Git before enabling writes. If Git has a
higher epoch, the server remains a replica, imports the Git-active journal, and
requires an explicit promotion to become active again. It MUST NOT merge two
independent histories or silently resume its former epoch.

## Visibility

During fallback, active/recent views show:

- `transport: git-fallback`;
- last verified server contact;
- current authority role and epoch;
- fallback inbox head commit;
- each agent's last fetched commit and unacknowledged messages;
- whether authoritative mutations are available;
- pending local Work Log outbox count.

This state is available through the CLI directly from Git. The UI may display
the last synchronized snapshot but must label it stale while its server is
unreachable.

## Acceptance Criteria

### AC: agents-communicate-with-server-down

**Given** SQLite active, a Git mirror, and a stopped Synchestra server
**When** agent A sends a targeted message and agent B polls the fallback inbox
**Then** B reads the exact immutable envelope, writes an acknowledgement, and A
observes it; both commands report Git commit evidence and no task/lease state is
mutated.

### AC: offline-local-outbox-replays

**Given** both server and Git are unreachable
**When** an agent sends a message and later regains Git access
**Then** the message remains explicitly pending in its Work Log, replays with
the original ID exactly once, and is never reported delivered before its Git
commit or server acknowledgement exists.

### AC: server-imports-fallback-idempotently

**Given** fallback envelopes written while the server was stopped
**When** the server starts and receives duplicate webhook and filesystem
notifications
**Then** it imports each valid envelope once, retains its Git evidence, delivers
it once per recipient, and reaches the same result after explicit periodic
reconciliation.

### AC: webhook-loss-does-not-lose-state

**Given** Git accepts pushes while every corresponding webhook is dropped
**When** periodic or startup reconciliation runs
**Then** all contiguous journal changes and fallback envelopes are discovered
from Git and imported; correctness does not depend on webhook delivery.

### AC: communication-fallback-does-not-split-authority

**Given** an unreachable server whose authority lease has not expired
**When** an agent enters communication fallback
**Then** messages and checkpoints can be appended, but claims and lifecycle
mutations are rejected with the active endpoint, epoch, and safe retry/promotion
instructions.

### AC: leased-failover-fences-restarted-server

**Given** a server that stopped renewing, an expired Git authority lease, and
two agents attempting failover
**When** both push a promotion
**Then** exactly one wins expected-base protection, Git becomes active at the
next epoch, and the restarted old server refuses writes until explicitly
promoted.

## Dependencies

- [Authoritative Store and Replicas](../) — authority epochs, replica cursors, promotion, and mirror barriers.
- [Git Backend](../../backends/git/) — inGitDB/DALgo commit and expected-base semantics.
- [Agent Coordination](../../../agent-coordination/) — message, run, acknowledgement, and activity records.
- [Repository Change Notifications](../../../agent-coordination/repository-change-notifications/) — normalized verified ref updates.
- [GitHub App](../../../github-app/) — optional webhook wake-up path.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
