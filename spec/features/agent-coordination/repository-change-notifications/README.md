---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Repository Change Notifications

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/repository-change-notifications?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/repository-change-notifications?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/repository-change-notifications?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/repository-change-notifications?op=request-change) |
**Status:** Approved
**Source Ideas:** —

## Summary

Normalizes agent, Workbench, Git-provider, and reconciliation signals into verified ref-update notifications for affected active agents.

## Problem

An agent can work for hours on a base that has changed without seeing it.
Polling alone is slow and cannot explain why a refresh is needed; provider
webhooks are fast but may be delayed, duplicated, reordered, or dropped. A
server must transform these heterogeneous signals into one verified,
auditable notice without treating an untrusted hint as repository truth or
forcing a dirty worktree to rebase.

## Behavior

### Typed input and verified event

Workbench, agents, Git providers, and scheduled reconciliation MAY submit a
typed input signal. The normalized event emitted by Synchestra is
`repository.ref.updated`:

```json
{
  "schema": "https://synchestra.io/schemas/repository-ref-updated/v1",
  "event_id": "01J...",
  "project_id": "github.com/acme/service",
  "repository_id": "github.com/acme/service",
  "ref": "refs/heads/main",
  "previous_sha": "a1b2...",
  "head_sha": "c3d4...",
  "observed_at": "2026-08-10T12:00:00Z",
  "verified_at": "2026-08-10T12:00:02Z",
  "source": "workbench|agent|github_webhook|reconciler",
  "evidence": {"remote": "origin", "commit": "c3d4..."}
}
```

`head_sha` becomes verified only after Synchestra fetches the configured remote
and confirms that the named ref resolves to it. A signal with an unknown
repository/ref, failed signature, inaccessible commit, non-fast-forward change,
or already-seen delivery does not produce a trusted update; it creates
diagnostic/audit state as appropriate. Events are idempotent by delivery/event
ID plus `(repository, ref, head_sha)` and retain source evidence.

### Affected claims and notifications

For every active Worktree Claim whose target/base ref matches the verified ref,
Synchestra records and delivers a `worktree.base_advanced` notification. It
contains claim/run/effort IDs, prior and new target SHA, the recipient's last
fetched/integrated SHA, policy trigger, and exact safe next action. It never
modifies the Git checkout itself.

If the recipient worktree is clean and its policy permits, the Workbench/agent
may fetch and assess immediately. If it is dirty, unpublished, or otherwise
unsafe, Synchestra sets `refresh_required` and reports it in project/repository
views. Acknowledgement records only that the run observed the notification;
integration evidence is a separate checkpoint showing method, target SHA, and
resulting branch head.

### Sources and reconciliation

The server continuously watches/reconciles registered Git state and code
repositories while running, and repeats a complete fetch/scan on startup and
at configured intervals. Reconciliation is the correctness path.

The optional GitHub App subscribes to push events for authorized repositories.
Its receiver verifies signature, installation scope, and delivery ID, then wakes
the relevant reconciliation worker. GitHub webhook payloads are latency hints
only: no payload directly updates a target head or worktree claim, and lost
webhooks are repaired by periodic/startup reconciliation. A local file-system
watcher is likewise only a hint; it cannot substitute for fetching the remote.

### Refresh configuration and delivery guarantees

Default policy is `interval: 60m`, plus verified target movement and the
pre-commit/push/handoff/finalize/merge assessments defined by Agent
Coordination. Configuration can narrow/extend intervals per repository,
but cannot disable assessment before merge. Delivery prefers server stream/
polling. During server outage it uses the Git fallback envelope and is later
reconciled idempotently.

All notifications show authority epoch, active/replica transport, and cursor.
They expose no local file contents, credentials, or raw prompts.

## Acceptance Criteria

### AC: verified-ref-update-notifies-affected-runs

**Given** two active claims target `main` and another targets `release`
**When** a verified `main` head advances
**Then** exactly the two `main` owners receive one `worktree.base_advanced`
notice with old/new SHAs; the `release` owner receives none.

### AC: dirty-worktree-is-never-auto-integrated

**Given** a target update and a claim with uncommitted files
**When** its notification is delivered
**Then** the claim is marked `refresh_required` with evidence, the owner is
notified, and no rebase/merge/reset is attempted by Synchestra or Workbench.

### AC: webhook-is-a-hint-not-truth

**Given** duplicated, delayed, or missing GitHub push webhooks
**When** reconciliation runs
**Then** each reachable ref head is fetched and verified once, affected agents
receive the same result as with a perfect webhook stream, and no state change
depends on the webhook body alone.

### AC: fallback-notification-reconciles-once

**Given** a server outage while Git is reachable
**When** Workbench publishes a `repository.ref.updated` fallback envelope and
the server resumes
**Then** the server verifies the remote ref, creates/deduplicates the canonical
update and affected notices, and retains Git commit evidence for the fallback
delivery.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
