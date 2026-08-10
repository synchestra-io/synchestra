---
format: https://specscore.md/plan-specification
status: Draft
---
# Plan: Synchestra Coordination Foundation

**Status:** Draft
**Source:** none
**Date:** 2026-08-10
**Owner:** codex
**Supersedes:** —

## Summary

Implement the first safe, observable coordination path for a founder now and a
team later: every active effort/run/worktree claim is visible per repository;
Codex and Claude runs coordinate over an audited channel; one writer owns each
branch/worktree; state is fast on a SQLite server yet portable to required Git;
and server outage leaves communication/recovery working through Git rather than
creating two authorities.

SpecScore remains generic. Its extension/event boundary may supply context, but
the Synchestra CLI/server owns operational state and Workbench owns local
checkout/worktree actions and Hybrid Work Log projection.

## End-to-End Journey

> “I open a repository and can see active and recent agents, their worktrees,
> their relationships, and conflicts. Two agents about to modify the same
> library coordinate through an auditable thread. If one dies, I can resume its
> exact effort; if it lands or is aborted, I can tell whether its worktree and
> branch are safe to remove. This remains understandable when the server is
> down.”

| Stage | Observable good result | E2E evidence |
|---|---|---|
| Start | Creating an effort/run and worktree claim returns one ID and owner; repository view shows it. | CLI/API create then independent project/repository query. |
| Coordinate | An overlapping claim is visible and a targeted message is acknowledged by the other run. | Two isolated agent fixtures; inspect their shared thread/audit. |
| Base moves | A verified main update reaches exactly affected claims; dirty worktree is marked, not rewritten. | Temporary remote + dirty fixture; assert notification and unchanged index. |
| Server fails | Agents exchange/ack Git envelopes and preserve checkpoints. | Stop server; inspect Git state and local Work Log; restart/reconcile. |
| Recover | Successor learns origin/model/prompt reference/branch/base/tip and can take only an expired or handed-off claim. | Kill a fixture run, start successor, prove lease fencing. |
| Close | Landed and aborted records remain distinct in seven-day view; cleanup accepts only verified eligible state. | Forge/temporary direct-push evidence plus cleanup dry-run. |

**Divergent epilogues.** The landed branch/worktree becomes cleanup-eligible only
after verified delivery and required Git durability. An aborted/unlanded effort
stays recent, resumable, and explicitly non-eligible until handed off or
authorized for discard. Both leave an audit record.

## Approach

Freeze backend-neutral records, journal/outbox, authority fencing, and CLI
contracts before implementation. First deliver local Git-active behavior so
recovery works without a server. Then add DALgo SQLite active mode and Git
mirror mode, server reconciliation, and notifications. Workbench integrates
only after Synchestra can return deterministic claims/health/cleanup decisions.

DALgo and inGitDB are upstream prerequisites, not workarounds in Synchestra:
inGitDB supplies the Git DALgo adapter; `dalgo2sqlite` supplies SQLite. The
first release validates the identical lifecycle workload in both Git-active →
SQLite-mirror and SQLite-active → Git-mirror modes, measuring correctness and
performance before increasing scale.

## Completed Work

| Completed item | Evidence | Result |
|---|---|---|
| Authoritative-store topology and offline fallback design | `f4b2fd5` | Exactly one active endpoint, Git-required replica, journal/outbox, barriers, fence epochs, Git fallback, reconciliation, and two-direction validation defined. |
| SpecScore initialization and deterministic standard lint fixes | `1463c23` | Synchestra is initialized for SpecScore; `Outstanding Questions` was mechanically migrated to `Open Questions` wherever the current linter could fix it. Legacy lint debt remains visible. |
| SQLite, agent coordination, and repository-update contracts | this plan's accompanying specification changes | Defined records, one-writer/handoff tradeoffs, active/recent views, refresh policy, verified signals, and implementation acceptance criteria. |
| Cross-harness conformance scenario selected | `agent-coordination/cross-harness-conformance` | Defined the Fair Split Relay, typed negotiation sequence, deterministic grader, CLI-first adapter order, outage variant, and zero-backlog terminal proof. |
| Existing Claude Code merger role reviewed | `agent-coordination/portable-merger-agent` | Preserved its refresh, batching, staged validation, tested-tree, conflict, and CI strengths; identified main-only, harness-specific, raw-worktree-fallback, and missing audited-cleanup gaps. |

These are approved design/preparatory artifacts, not a claim that the runtime
is implemented or delivered.

## Tasks

### Task 1: Extend the backend-neutral state model and topology validator

**Id:** task-1
**Verifies:** state-store/topology#ac:topology-rejects-zero-or-multiple-active, state-store/topology#ac:promotion-fences-former-active
**Depends-On:** —
**Status:** planning

Extend `state.Store` with effort/run/worktree claim, message, activity,
replication journal, cursor, authority lease, and health contracts. Implement
configuration validation for exactly one active, one-or-more replicas, and
Git-required topology; add deterministic schema and migration ownership.

### Task 2: Add DALgo/inGitDB Git endpoint and append-only fallback ingress

**Id:** task-2
**Verifies:** state-store/topology#ac:git-active-replicates-to-sqlite, state-store/topology/offline-fallback#ac:agents-communicate-with-server-down, state-store/topology/offline-fallback#ac:communication-fallback-does-not-split-authority
**Depends-On:** 1
**Status:** planning

Extend inGitDB/DALgo upstream for transactional buffering, deterministic
serialization, expected-base/CAS, and commit SHA evidence. Implement Git-active
journal records, immutable fallback envelopes, recipient acknowledgements, and
direct Git active/recent queries with conflict-retry semantics.

### Task 3: Implement fenced effort/run/worktree claims and audited messaging

**Id:** task-3
**Verifies:** agent-coordination#ac:one-writer-claim-is-fenced, agent-coordination#ac:messages-survive-transport-switch, agent-coordination#ac:abandoned-run-is-resumable
**Depends-On:** 1, 2
**Status:** planning

Implement lifecycle transitions, lease/fence enforcement, explicit handoff,
scope declarations, message threads, acknowledgements, and recovery records.
Integrate the Workbench Hybrid Work Log through stable IDs/references rather
than duplicating private prompt content in the Git state repository.

### Task 4: Implement DALgo SQLite active endpoint, journal, and outbox

**Id:** task-4
**Verifies:** state-store/backends/sqlite#ac:sqlite-active-uses-one-transaction, state-store/backends/sqlite#ac:sqlite-restart-obeys-fencing, state-store/topology#ac:sqlite-active-commits-outbox-atomically
**Depends-On:** 1
**Status:** planning

Create `sqlitestore` using `dalgo2sqlite`, migrations, conditional claim
writes, one transaction for projection/journal/outbox, idempotent receipts,
and labelled query responses. Add crash-injection tests at each transaction
boundary and restart/promotion fence coverage.

### Task 5: Implement replica workers, barriers, verification, and promotion

**Id:** task-5
**Verifies:** state-store/topology#ac:replica-outage-does-not-create-dual-write, state-store/topology#ac:mirror-barrier-proves-git-durability, state-store/backends/sqlite#ac:git-barrier-proves-portable-durability
**Depends-On:** 2, 4
**Status:** planning

Deliver ordered outbox apply, checksum/cursor verification, health/lag metrics,
mirror barriers, backup checkpoints, and explicit lease/epoch promotion.
Exercise Git-active→SQLite and SQLite-active→Git workloads and record the
required performance/recovery comparison without exposing private payloads.

### Task 6: Implement server reconciliation and verified ref notifications

**Id:** task-6
**Verifies:** agent-coordination/repository-change-notifications#ac:verified-ref-update-notifies-affected-runs, agent-coordination/repository-change-notifications#ac:webhook-is-a-hint-not-truth, state-store/topology/offline-fallback#ac:webhook-loss-does-not-lose-state
**Depends-On:** 3, 5
**Status:** planning

Add server startup/periodic Git reconciliation and optional GitHub App webhook
ingress as a signed, deduplicated wake-up hint. Emit verified
`repository.ref.updated`, derive `worktree.base_advanced`, and preserve dirty
checkout safety. Update serve/server/API and repository configuration contracts.

### Task 7: Integrate Workbench lifecycle, refresh policy, and safe cleanup/recycle

**Id:** task-7
**Verifies:** agent-coordination#ac:completed-and-aborted-are-distinguishable, agent-coordination#ac:delivery-requires-merge-and-asset-disposition, agent-coordination#ac:recycle-is-explicit-and-isolated, agent-coordination#ac:offline-archive-does-not-block-default-cleanup, agent-coordination/repository-change-notifications#ac:dirty-worktree-is-never-auto-integrated, agent-coordination#ac:overlap-is-visible-and-coordinated
**Depends-On:** 3, 6
**Status:** planning

Make WB create all managed worktrees under canonical owner/repository paths,
write/update local Work Log projections, renew/release claims, apply the
60-minute/pre-critical-operation refresh policy, surface overlap, and call
Synchestra before cleanup. Completion requires merge and immediate verified
push to the declared remote task or feature target plus removal/recycle of
every claimed asset. A local target merge is `awaiting_push` and blocks cleanup. Default cleanup
removes. Optional recycle seals to a mandatory local archive, queues optional
remote archival, then resets/rebinds a new Work Log, claim, and base while
retaining only configured caches. Landing verification must accept forge PR
evidence and verified direct-to-target landing, never graph-shape guesses.

### Task 8: Deliver project/repository views and prove the full outage/recovery journey

**Id:** task-8
**Verifies:** agent-coordination#ac:abandoned-run-is-resumable, agent-coordination#ac:completed-and-aborted-are-distinguishable, agent-coordination/repository-change-notifications#ac:fallback-notification-reconciles-once, state-store/topology#ac:backend-comparison-is-equivalent
**Depends-On:** 5, 6, 7
**Status:** planning

Add CLI JSON/table/graph views for active and seven-day recent records, then
run the full independent E2E journey: two overlapping agents, targeted
coordination, a ref advance, server outage/fallback, restart reconciliation,
dead-run takeover, landed/aborted branches, and cleanup eligibility. This task
owns the mechanism-level E2E evidence and operator runbook.

### Task 9: Run cross-harness Fair Split conformance

**Id:** task-9
**Verifies:** agent-coordination/cross-harness-conformance#ac:two-cli-harnesses-negotiate-bidirectionally, agent-coordination/cross-harness-conformance#ac:accepted-contract-produces-exact-split, agent-coordination/cross-harness-conformance#ac:coordination-evidence-is-auditable-not-prose-graded, agent-coordination/cross-harness-conformance#ac:server-outage-reconciles-the-same-negotiation, agent-coordination/cross-harness-conformance#ac:conformance-leaves-no-cleanup-backlog, agent-coordination/cross-harness-conformance#ac:adapter-matrix-reuses-one-scenario
**Depends-On:** 3, 5, 7, 8
**Status:** planning

Build the reusable Fair Split fixture, typed negotiation grader, and harness
adapter contract. Pass Codex CLI ↔ Claude Code CLI in normal and server-outage
modes first; add GitHub Copilot CLI pairwise coverage next. Attach Claude
Desktop and the Codex desktop app through the same protocol once stable launch
or attachment mechanisms are selected. A pass includes exact behavioral output,
bidirectional audited coordination, store convergence, task→feature→main
landing, and zero orphaned branches/worktrees.

### Task 10: Deliver the portable merger role and deterministic WB merge workflow

**Id:** task-10
**Verifies:** agent-coordination/portable-merger-agent#ac:adapter-contract-is-harness-neutral, agent-coordination/portable-merger-agent#ac:compatible-branches-land-as-one-batch, agent-coordination/portable-merger-agent#ac:drift-invalidates-the-plan, agent-coordination/portable-merger-agent#ac:behavioral-conflict-is-audited-not-guessed, agent-coordination/portable-merger-agent#ac:tested-tree-is-the-landed-tree, agent-coordination/portable-merger-agent#ac:every-target-merge-is-pushed-immediately, agent-coordination/portable-merger-agent#ac:success-requires-zero-related-assets, agent-coordination/portable-merger-agent#ac:wb-outage-cannot-be-bypassed, agent-coordination/cross-harness-conformance#ac:portable-merger-drains-the-fixture
**Depends-On:** 3, 7, 8
**Status:** planning

Implement the hashed, resumable WB plan/apply/status workflow and provider-
neutral landing/CI receipts. Version one canonical merger role and publish thin
Claude Code, Codex, and Copilot adapters plus capability-delivery manifests.
Replace the Claude adapter's raw `git worktree` fallback with WB's audited
Git/Work Log fallback, support task→feature as well as feature→`main`, and make
sealed Work Logs plus zero related claims/backlog a terminal success condition.
Extend the Fair Split scenario so a separately launched merger drains the two
writer branches without the primary session duplicating mechanical work.

### Task 11: Detect and resolve plan overlap before approval

**Id:** task-11
**Verifies:** agent-coordination/plan-overlap-coordination#ac:same-repository-plans-require-explicit-evaluation, agent-coordination/plan-overlap-coordination#ac:one-provider-replaces-duplicate-implementation, agent-coordination/plan-overlap-coordination#ac:semantic-suggestions-are-advisory-and-cited, agent-coordination/plan-overlap-coordination#ac:independent-work-remains-parallel, agent-coordination/plan-overlap-coordination#ac:consumer-waits-only-at-real-dependency, agent-coordination/plan-overlap-coordination#ac:approval-check-is-revision-bound-and-fail-closed, agent-coordination/plan-overlap-coordination#ac:migration-mode-covers-every-consumer-or-is-approved, agent-coordination/plan-overlap-coordination#ac:event-hooks-remain-decoupled-from-policy-results, agent-coordination/plan-overlap-coordination#ac:fair-split-plans-produce-one-allocator
**Depends-On:** 3, 8, 10
**Status:** planning

Define generic SpecScore task-intent metadata and a synchronous, structured
policy-check extension upstream; keep the existing one-way event/outbox for
notifications. Implement Synchestra revision-bound Plan Intents, incremental
repository/capability/structure/dependency matching, advisory cited semantic
matching, typed owner resolutions, approval/claim guards, dependency-aware
scheduling, and project/repository graph views. Treat migration verbs as a
full-cutover default with a deterministic consumer inventory and require an
owner-approved staged/pilot contract before any legacy consumer may remain.
Extend Fair Split so two independent master-agent plans initially duplicate the
allocator, resolve to one provider before workers start, and land with one
implementation and zero cleanup backlog; also prove independent same-repo work
is not unnecessarily serialized.

## Open Questions

1. The current Synchestra `state.Store` is task/chat-centric; Task 1 decides
   whether effort/run/worktree APIs are a composed `Agent()` store or a sibling
   top-level coordinator interface. Both preserve backend-neutral domain code.
2. The initial SQLite server is single-host by design. Cross-host HA and a
   remote SQL backend require a separate consensus/lease-witness decision;
   they are not implicit in the SQLite MVP.
3. The GitHub App's current permissions predate this feature. Task 6 must
   reduce them to the minimum required for verified push hints and document any
   additional permission before requesting it from users.

---
*This document follows the https://specscore.md/plan-specification*
